#!/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
script="$repo_root/deploy/remote-pull-restart.sh"

bash -n "$script"
grep -q 'sub2api-linux.gz' "$script"
grep -q 'sub2api-linux.new' "$script"
grep -q 'sha256sum' "$script"
grep -q 'mv.*STAGED.*BINARY' "$script"

echo 'remote artifact install checks passed'
