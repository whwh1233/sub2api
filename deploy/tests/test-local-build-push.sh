#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
script="$repo_root/deploy/local-build-push.sh"

bash -n "$script"
grep -q 'pnpm install --frozen-lockfile' "$script"
grep -q 'go build -tags=embed' "$script"
grep -q 'sandbox-exec' "$script"
grep -q '/setup/status' "$script"
grep -q 'GOOS=linux GOARCH=amd64 CGO_ENABLED=0' "$script"
grep -q 'gzip -t' "$script"
grep -q 'shasum -a 256' "$script"
grep -q 'gzip round-trip checksum mismatch' "$script"
grep -q '100 \* 1024 \* 1024' "$script"

if grep -Eq 'git +(add|commit|push)' "$script"; then
	echo 'macOS build script must not combine Git mutations with artifact creation' >&2
	exit 1
fi

echo 'macOS local build checks passed'
