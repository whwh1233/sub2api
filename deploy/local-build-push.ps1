# local-build-push.ps1
# Historical filename retained for compatibility. This script only builds and
# validates local release artifacts. Git and production actions are separate,
# explicitly confirmed steps under AGENTS.md and docs/UPGRADE_RUNBOOK.md.

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

& (Join-Path $PSScriptRoot 'local-dev-environment.ps1')

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$BackendDir = Join-Path $RepoRoot 'backend'
$FrontendDir = Join-Path $RepoRoot 'frontend'
$LocalDataDir = Join-Path $RepoRoot '.local-server-prodtest'
$LocalConfig = Join-Path $LocalDataDir 'config.yaml'
$RunDir = Join-Path $RepoRoot '.local-run'
$WindowsPort = 18080

function Step {
    param([int]$Number, [string]$Message)
    Write-Host ''
    Write-Host "==== [$Number/4] $Message ====" -ForegroundColor Cyan
}

function Get-ConfigScalar {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Section,
        [Parameter(Mandatory = $true)][string]$Key
    )

    $inside = $false
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -match "^\s*$([regex]::Escape($Section))\s*:\s*$") {
            $inside = $true
            continue
        }
        if ($inside -and $line -match '^[A-Za-z0-9_-]+\s*:') { break }
        if ($inside -and $line -match "^\s{2}$([regex]::Escape($Key))\s*:\s*(.+?)\s*$") {
            return $Matches[1].Trim().Trim('"').Trim("'")
        }
    }
    return ''
}

function Assert-LocalProductionCloneConfig {
    if (-not (Test-Path -LiteralPath $LocalConfig)) {
        throw "Fresh local production-clone config is missing: $LocalConfig. Run .\.local-dev\sync-prod-db-local.ps1 -StartServer first."
    }
    $dbHost = Get-ConfigScalar -Path $LocalConfig -Section 'database' -Key 'host'
    $dbName = Get-ConfigScalar -Path $LocalConfig -Section 'database' -Key 'dbname'
    if ($dbHost -notin @('127.0.0.1', 'localhost', '::1')) {
        throw "Refusing candidate test because database.host is not loopback: $dbHost"
    }
    if ($dbName -ne 'sub2api') {
        throw "Refusing candidate test because database.dbname is not sub2api: $dbName"
    }
}

function Invoke-FrontendGate {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$Label
    )

    $healthCodes = @()
    1..3 | ForEach-Object {
        $healthCodes += (Invoke-WebRequest -UseBasicParsing "$BaseUrl/health" -TimeoutSec 10).StatusCode
    }
    if ($healthCodes | Where-Object { $_ -ne 200 }) {
        throw "$Label health gate failed: $($healthCodes -join ',')"
    }

    $root = Invoke-WebRequest -UseBasicParsing "$BaseUrl/" -TimeoutSec 15
    if ($root.StatusCode -ne 200 -or $root.Headers['Content-Type'] -notmatch '^text/html') {
        throw "$Label root gate failed: status=$($root.StatusCode) content-type=$($root.Headers['Content-Type'])"
    }
    $setup = Invoke-WebRequest -UseBasicParsing "$BaseUrl/setup/status" -TimeoutSec 15
    if ($setup.StatusCode -ne 200) { throw "$Label setup gate failed: status=$($setup.StatusCode)" }

    $matches = [regex]::Matches($root.Content, '(?:src|href)="([^"?#]+\.(?:js|css)(?:\?[^"#]*)?)"')
    $assets = @($matches | ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique)
    if ($assets.Count -eq 0) { throw "$Label root HTML referenced no JavaScript or CSS assets" }
    foreach ($asset in $assets) {
        $uri = [Uri]::new([Uri]"$BaseUrl/", $asset)
        $response = Invoke-WebRequest -UseBasicParsing $uri.AbsoluteUri -TimeoutSec 20
        if ($response.StatusCode -ne 200 -or $response.Content.Length -le 0) {
            throw "$Label asset gate failed: $asset status=$($response.StatusCode) bytes=$($response.Content.Length)"
        }
    }
    Write-Host "  $Label gate OK: health=200x3 root=200 setup=200 assets=$($assets.Count)" -ForegroundColor Green
}

function Wait-ForCandidate {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][System.Diagnostics.Process]$Process,
        [Parameter(Mandatory = $true)][string]$StdoutLog,
        [Parameter(Mandatory = $true)][string]$StderrLog
    )

    $deadline = (Get-Date).AddSeconds(90)
    while ((Get-Date) -lt $deadline) {
        if ($Process.HasExited) {
            $details = @()
            if (Test-Path -LiteralPath $StdoutLog) { $details += Get-Content -LiteralPath $StdoutLog -Tail 60 }
            if (Test-Path -LiteralPath $StderrLog) { $details += Get-Content -LiteralPath $StderrLog -Tail 60 }
            throw "Candidate exited before health gate (exit $($Process.ExitCode)):`n$($details -join "`n")"
        }
        try {
            $response = Invoke-WebRequest -UseBasicParsing "$BaseUrl/health" -TimeoutSec 3
            if ($response.StatusCode -eq 200) { return }
        }
        catch {
        }
        Start-Sleep -Seconds 2
    }
    throw "Candidate did not become healthy within 90 seconds: $BaseUrl"
}

function Invoke-WindowsCandidateSmoke {
    param([Parameter(Mandatory = $true)][string]$Executable)

    $stdoutLog = Join-Path $RunDir 'release-windows.stdout.log'
    $stderrLog = Join-Path $RunDir 'release-windows.stderr.log'
    Remove-Item -LiteralPath $stdoutLog, $stderrLog -Force -ErrorAction SilentlyContinue

    $names = @('DATA_DIR', 'SERVER_HOST', 'SERVER_PORT', 'SERVER_DISABLE_BACKGROUND_WORKERS')
    $previous = @{}
    foreach ($name in $names) { $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
    $process = $null
    try {
        $env:DATA_DIR = $LocalDataDir
        $env:SERVER_HOST = '127.0.0.1'
        $env:SERVER_PORT = [string]$WindowsPort
        $env:SERVER_DISABLE_BACKGROUND_WORKERS = 'true'
        $process = Start-Process -FilePath $Executable -WorkingDirectory $BackendDir -WindowStyle Hidden -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog -PassThru
    }
    finally {
        foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $previous[$name], 'Process') }
    }

    try {
        $baseUrl = "http://127.0.0.1:$WindowsPort"
        Wait-ForCandidate -BaseUrl $baseUrl -Process $process -StdoutLog $stdoutLog -StderrLog $stderrLog
        Invoke-FrontendGate -BaseUrl $baseUrl -Label 'Windows candidate'
    }
    finally {
        if ($null -ne $process -and -not $process.HasExited) {
            Stop-Process -Id $process.Id -Force
            $process.WaitForExit()
        }
    }
}

New-Item -ItemType Directory -Force -Path $RunDir | Out-Null
Assert-LocalProductionCloneConfig

Step 1 'Build frontend with locked dependencies'
Push-Location $FrontendDir
try {
    pnpm install --frozen-lockfile
    if ($LASTEXITCODE -ne 0) { throw 'pnpm install failed' }
    pnpm build
    if ($LASTEXITCODE -ne 0) { throw 'pnpm build failed' }
}
finally { Pop-Location }

Step 2 'Build Windows amd64 embed candidate'
$windowsBinary = Join-Path $BackendDir 'sub2api-windows.exe'
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
try {
    Push-Location $BackendDir
    try { go build -tags=embed -ldflags='-s -w' -o $windowsBinary ./cmd/server/ }
    finally { Pop-Location }
    if ($LASTEXITCODE -ne 0) { throw 'Windows go build failed' }
}
finally { Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue }
Write-Host ('  Windows binary: {0:N1} MB' -f ((Get-Item -LiteralPath $windowsBinary).Length / 1MB)) -ForegroundColor Green
Invoke-WindowsCandidateSmoke -Executable $windowsBinary

Step 3 'Build Linux amd64 embed candidate'
$linuxBinary = Join-Path $BackendDir 'sub2api-linux'
$env:GOOS = 'linux'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
try {
    Push-Location $BackendDir
    try { go build -tags=embed -ldflags='-s -w' -o $linuxBinary ./cmd/server/ }
    finally { Pop-Location }
    if ($LASTEXITCODE -ne 0) { throw 'Linux go build failed' }
}
finally { Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue }

Step 4 'Create and round-trip verify Linux gzip artifact'
$linuxArtifact = Join-Path $BackendDir 'sub2api-linux.gz'
$linuxChecksum = Join-Path $BackendDir 'sub2api-linux.sha256'
$verifyBinary = "$linuxBinary.verify"
Remove-Item -LiteralPath $linuxArtifact, $linuxChecksum, $verifyBinary -Force -ErrorAction SilentlyContinue

$input = [System.IO.File]::OpenRead($linuxBinary)
$output = [System.IO.File]::Create($linuxArtifact)
try {
    $gzip = [System.IO.Compression.GZipStream]::new($output, [System.IO.Compression.CompressionLevel]::SmallestSize, $true)
    try { $input.CopyTo($gzip) } finally { $gzip.Dispose() }
}
finally { $input.Dispose(); $output.Dispose() }

$rawHash = (Get-FileHash -LiteralPath $linuxBinary -Algorithm SHA256).Hash.ToLowerInvariant()
Set-Content -LiteralPath $linuxChecksum -Value "$rawHash  sub2api-linux" -Encoding ASCII
$compressedInput = [System.IO.File]::OpenRead($linuxArtifact)
$verifyOutput = [System.IO.File]::Create($verifyBinary)
try {
    $gzip = [System.IO.Compression.GZipStream]::new($compressedInput, [System.IO.Compression.CompressionMode]::Decompress, $true)
    try { $gzip.CopyTo($verifyOutput) } finally { $gzip.Dispose() }
}
finally { $compressedInput.Dispose(); $verifyOutput.Dispose() }
$verifyHash = (Get-FileHash -LiteralPath $verifyBinary -Algorithm SHA256).Hash.ToLowerInvariant()
Remove-Item -LiteralPath $verifyBinary -Force
if ($verifyHash -ne $rawHash) { throw "gzip round-trip checksum mismatch: expected $rawHash but got $verifyHash" }
if ((Get-Item -LiteralPath $linuxArtifact).Length -ge 100MB) { throw 'Compressed Linux artifact exceeds the GitHub 100 MiB limit' }

Write-Host ('  Linux gzip: {0:N1} MB; raw SHA256: {1}' -f ((Get-Item -LiteralPath $linuxArtifact).Length / 1MB), $rawHash) -ForegroundColor Green
Write-Warning 'Linux execution validation remains required in Docker or the isolated goodserver test before deployment.'
Write-Host ''
Write-Host '[OK] Local release artifacts built and verified. No Git or production action was performed.' -ForegroundColor Green
Write-Host "  Artifact: $linuxArtifact"
Write-Host "  Checksum: $linuxChecksum"
