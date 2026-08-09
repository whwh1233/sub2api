$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..\..')
$buildScript = Get-Content (Join-Path $repoRoot 'deploy\local-build-push.ps1') -Raw
$remoteScript = Get-Content (Join-Path $repoRoot 'deploy\remote-pull-restart.sh') -Raw

function Assert-Contains {
    param([string]$Text, [string]$Pattern, [string]$Message)
    if ($Text -notmatch $Pattern) {
        throw $Message
    }
}

Assert-Contains $buildScript 'sub2api-linux\.gz' 'build script must create the gzip artifact'
Assert-Contains $buildScript 'sub2api-linux\.sha256' 'build script must create the raw binary checksum'
Assert-Contains $buildScript 'GZipStream|gzip' 'build script must gzip the Linux binary'
Assert-Contains $buildScript 'Get-FileHash' 'build script must verify artifact integrity'
Assert-Contains $buildScript '\.local-server-prodtest' 'build script must use the fresh local production clone'
Assert-Contains $buildScript 'SERVER_DISABLE_BACKGROUND_WORKERS' 'candidate tests must disable background workers'
Assert-Contains $buildScript 'Invoke-FrontendGate' 'build script must validate HTML and referenced assets'
Assert-Contains $buildScript 'No Git or production action was performed' 'build script must stop before release mutations'
if ($buildScript -match '(?m)^\s*git\s+(?:add|commit|push)\b') {
    throw 'build script must not stage, commit, or push'
}
Assert-Contains $remoteScript 'sub2api-linux\.gz' 'remote script must consume the gzip artifact'
Assert-Contains $remoteScript 'sub2api-linux\.new' 'remote script must install through a staging file'
Assert-Contains $remoteScript 'sha256sum' 'remote script must verify the raw checksum before replacement'
Assert-Contains $remoteScript 'git pull --ff-only origin main' 'remote script must refuse a non-fast-forward production pull'
Assert-Contains $remoteScript 'BACKUP_STAGED=' 'remote script must stage the live binary backup before replacing .prev'
Assert-Contains $remoteScript 'sha256sum "\$BACKUP_STAGED"' 'remote script must verify the staged backup checksum'
Assert-Contains $remoteScript 'mv -f "\$BACKUP_STAGED" "\$BACKUP"' 'remote script must atomically replace .prev after verification'
if ($remoteScript -match '(?m)^git pull origin main$') {
    throw 'remote script must not use an unguarded production pull'
}

$artifact = Join-Path $repoRoot 'backend\sub2api-linux.gz'
$checksum = Join-Path $repoRoot 'backend\sub2api-linux.sha256'
if ((Test-Path $artifact) -xor (Test-Path $checksum)) {
    throw 'gzip artifact and checksum must be produced together'
}

Write-Host 'compressed release checks passed'
