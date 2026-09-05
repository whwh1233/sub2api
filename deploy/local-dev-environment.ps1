# Keep Windows development writes on the repository drive, away from the OS temp/cache.
$developmentRepo = (Resolve-Path (Join-Path $PSScriptRoot '..')).ProviderPath
$developmentCacheRoot = Join-Path ([IO.Path]::GetPathRoot($developmentRepo)) 'codex\.cache'
$developmentPaths = @{
    GOCACHE = Join-Path $developmentCacheRoot 'go-build'
    GOMODCACHE = Join-Path $developmentCacheRoot 'go-mod'
    GOPATH = Join-Path $developmentCacheRoot 'go'
    GOTMPDIR = Join-Path $developmentCacheRoot 'go-tmp'
    GOLANGCI_LINT_CACHE = Join-Path $developmentCacheRoot 'golangci'
    TEMP = Join-Path $developmentCacheRoot 'go-tmp'
    TMP = Join-Path $developmentCacheRoot 'go-tmp'
}
foreach ($developmentEntry in $developmentPaths.GetEnumerator()) {
    New-Item -ItemType Directory -Path $developmentEntry.Value -Force | Out-Null
    [Environment]::SetEnvironmentVariable($developmentEntry.Key, $developmentEntry.Value, 'Process')
}
$developmentGoBin = Join-Path $env:GOPATH 'bin'
if ($developmentGoBin -notin ($env:PATH -split ';')) {
    $env:PATH = "$developmentGoBin;$env:PATH"
}
