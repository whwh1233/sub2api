# local-build-push.ps1
# 本地构建 Linux 二进制并推到 GitHub
# 用法：在 PowerShell 里 cd 到仓库根目录后跑：
#   powershell -ExecutionPolicy Bypass -File .\deploy\local-build-push.ps1
#
# 或直接双击（如果 ExecutionPolicy 已设为 RemoteSigned/Unrestricted）

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

function Step($n, $total, $msg) {
    Write-Host ""
    Write-Host "==== [$n/$total] $msg ====" -ForegroundColor Cyan
}

# ---- 1. 构建前端 ----
Step 1 4 "构建前端 (pnpm build)"
Set-Location "$RepoRoot\frontend"
pnpm install --frozen-lockfile
if ($LASTEXITCODE -ne 0) { throw "pnpm install 失败" }
pnpm build
if ($LASTEXITCODE -ne 0) { throw "pnpm build 失败" }

# ---- 2. 交叉编译 Linux 二进制 ----
Step 2 4 "交叉编译 Linux amd64 二进制"
Set-Location "$RepoRoot\backend"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
try {
    go build -o sub2api-linux ./cmd/server/
    if ($LASTEXITCODE -ne 0) { throw "go build 失败" }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
$binSize = (Get-Item "$RepoRoot\backend\sub2api-linux").Length / 1MB
Write-Host ("  二进制大小: {0:N1} MB" -f $binSize) -ForegroundColor Green

# ---- 3. 提交二进制 ----
Step 3 4 "提交二进制到 git"
Set-Location $RepoRoot
git add backend/sub2api-linux
$staged = git diff --cached --name-only
if (-not $staged) {
    Write-Host "  二进制未变化，跳过 commit" -ForegroundColor Yellow
} else {
    $ts = Get-Date -Format "yyyy-MM-dd HH:mm"
    git commit -m "build: linux binary $ts"
    if ($LASTEXITCODE -ne 0) { throw "git commit 失败" }
}

# ---- 4. 推送 ----
Step 4 4 "git push origin main"
git push origin main
if ($LASTEXITCODE -ne 0) { throw "git push 失败" }

Write-Host ""
Write-Host "✓ 本地完成。现在去 VPS 跑：" -ForegroundColor Green
Write-Host "  bash /root/sub2api/deploy/remote-pull-restart.sh" -ForegroundColor Green
