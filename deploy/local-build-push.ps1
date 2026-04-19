# local-build-push.ps1
# Build frontend, build Windows binary for local smoke test, then build Linux
# binary and push to GitHub.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File .\deploy\local-build-push.ps1
#   powershell -ExecutionPolicy Bypass -File .\deploy\local-build-push.ps1 -SkipSmokeTest

param(
    [switch]$SkipSmokeTest
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

$TotalSteps = if ($SkipSmokeTest) { 5 } else { 6 }

function Step($n, $msg) {
    Write-Host ""
    Write-Host "==== [$n/$TotalSteps] $msg ====" -ForegroundColor Cyan
}

# ---- 1. Build frontend ----
Step 1 "Build frontend (pnpm build)"
Set-Location "$RepoRoot\frontend"
pnpm install --frozen-lockfile
if ($LASTEXITCODE -ne 0) { throw "pnpm install failed" }
pnpm build
if ($LASTEXITCODE -ne 0) { throw "pnpm build failed" }

# ---- 2. Build Windows binary for local verification ----
Step 2 "Build Windows amd64 binary for local smoke test"
Set-Location "$RepoRoot\backend"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
try {
    go build -tags=embed -ldflags="-s -w" -o sub2api-windows.exe ./cmd/server/
    if ($LASTEXITCODE -ne 0) { throw "windows go build failed" }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
$winSize = (Get-Item "$RepoRoot\backend\sub2api-windows.exe").Length / 1MB
Write-Host ("  Windows binary: {0:N1} MB (gitignored via *.exe)" -f $winSize) -ForegroundColor Green

# ---- 3. Manual smoke test ----
if (-not $SkipSmokeTest) {
    Step 3 "Smoke test Windows binary (manual)"
    Write-Host ""
    Write-Host "  In another terminal run:" -ForegroundColor Yellow
    Write-Host "    cd $RepoRoot\backend" -ForegroundColor Yellow
    Write-Host "    .\sub2api-windows.exe" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Then verify at http://localhost:8080/ (expect HTML homepage, not 404)." -ForegroundColor Yellow
    Write-Host "  NOTE: backend binds :8080 and reads backend\config.yaml (may hit prod DB)." -ForegroundColor DarkYellow
    Write-Host ""
    Write-Host "  Press ENTER to continue (build Linux + push), or Ctrl+C to abort." -ForegroundColor Cyan
    [void](Read-Host)
    $nextStep = 4
} else {
    Write-Host ""
    Write-Host "  -SkipSmokeTest set, skipping manual verification" -ForegroundColor DarkYellow
    $nextStep = 3
}

# ---- 4. Cross-compile Linux binary ----
Step $nextStep "Cross-compile Linux amd64 binary"
Set-Location "$RepoRoot\backend"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
try {
    go build -tags=embed -ldflags="-s -w" -o sub2api-linux ./cmd/server/
    if ($LASTEXITCODE -ne 0) { throw "linux go build failed" }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
$linSize = (Get-Item "$RepoRoot\backend\sub2api-linux").Length / 1MB
Write-Host ("  Linux binary: {0:N1} MB" -f $linSize) -ForegroundColor Green
$nextStep++

# ---- 5. Commit Linux binary ----
Step $nextStep "Commit Linux binary to git"
Set-Location $RepoRoot
git add backend/sub2api-linux
$staged = git diff --cached --name-only
if (-not $staged) {
    Write-Host "  Binary unchanged, skipping commit" -ForegroundColor Yellow
} else {
    $ts = Get-Date -Format "yyyy-MM-dd HH:mm"
    git commit -m "build: linux binary $ts"
    if ($LASTEXITCODE -ne 0) { throw "git commit failed" }
}
$nextStep++

# ---- 6. Push ----
Step $nextStep "git push origin main"
git push origin main
if ($LASTEXITCODE -ne 0) { throw "git push failed" }

Write-Host ""
Write-Host "[OK] Local build and push complete. On VPS run:" -ForegroundColor Green
Write-Host "  bash /root/sub2api/deploy/remote-pull-restart.sh" -ForegroundColor Green
