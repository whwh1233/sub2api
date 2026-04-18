# local-build-push.ps1
# Build Linux binary locally, commit, and push to GitHub.
# Usage (PowerShell):
#   powershell -ExecutionPolicy Bypass -File .\deploy\local-build-push.ps1

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

function Step($n, $total, $msg) {
    Write-Host ""
    Write-Host "==== [$n/$total] $msg ====" -ForegroundColor Cyan
}

# ---- 1. Build frontend ----
Step 1 4 "Build frontend (pnpm build)"
Set-Location "$RepoRoot\frontend"
pnpm install --frozen-lockfile
if ($LASTEXITCODE -ne 0) { throw "pnpm install failed" }
pnpm build
if ($LASTEXITCODE -ne 0) { throw "pnpm build failed" }

# ---- 2. Cross-compile Linux binary ----
Step 2 4 "Cross-compile Linux amd64 binary (with -tags=embed -ldflags -s -w)"
Set-Location "$RepoRoot\backend"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
try {
    go build -tags=embed -ldflags="-s -w" -o sub2api-linux ./cmd/server/
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
$binSize = (Get-Item "$RepoRoot\backend\sub2api-linux").Length / 1MB
Write-Host ("  Binary size: {0:N1} MB" -f $binSize) -ForegroundColor Green

# ---- 3. Commit binary ----
Step 3 4 "Commit binary to git"
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

# ---- 4. Push ----
Step 4 4 "git push origin main"
git push origin main
if ($LASTEXITCODE -ne 0) { throw "git push failed" }

Write-Host ""
Write-Host "[OK] Local build and push complete. On VPS run:" -ForegroundColor Green
Write-Host "  bash /root/sub2api/deploy/remote-pull-restart.sh" -ForegroundColor Green
