#!/usr/bin/env bash
# Build and validate release artifacts on macOS.
#
# This script intentionally stops after producing a verified artifact. Staging,
# committing, pushing, and production deployment are separate confirmed steps.

set -euo pipefail

SKIP_SMOKE_TEST=0

usage() {
	cat <<'EOF'
Usage: ./deploy/local-build-push.sh [--skip-smoke-test]

  --skip-smoke-test  Skip the isolated macOS candidate smoke test. Use only
                     when that variant was explicitly approved.
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--skip-smoke-test)
		SKIP_SMOKE_TEST=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		exit 2
		;;
	esac
done

[[ "$(uname -s)" == "Darwin" ]] || {
	echo "ERROR: this release entry point is for macOS" >&2
	exit 1
}

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/frontend"
BACKEND_DIR="$REPO_ROOT/backend"
LOCAL_TEST_DIR="$REPO_ROOT/.local-server-prodtest"
LOCAL_CONFIG="$LOCAL_TEST_DIR/config.yaml"
SANDBOX_PROFILE="$LOCAL_TEST_DIR/network.sb"
LOCAL_BINARY="$LOCAL_TEST_DIR/sub2api-local"
LOCAL_LOG="$LOCAL_TEST_DIR/release-smoke.log"
LOCAL_BASE_URL="http://127.0.0.1:18080"
LINUX_BINARY="$BACKEND_DIR/sub2api-linux"
LINUX_ARTIFACT="$LINUX_BINARY.gz"
LINUX_CHECKSUM="$LINUX_BINARY.sha256"
VERIFY_BINARY="$LINUX_BINARY.verify"
CANDIDATE_PID=""

TOTAL_STEPS=5
if [[ "$SKIP_SMOKE_TEST" == "1" ]]; then
	TOTAL_STEPS=4
fi

step() {
	printf '\n==== [%s/%s] %s ====\n' "$1" "$TOTAL_STEPS" "$2"
}

fail() {
	echo "ERROR: $*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

activate_node20() {
	local major=""
	if command -v node >/dev/null 2>&1; then
		major="$(node -p 'process.versions.node.split(".")[0]')"
	fi
	if [[ "$major" != "20" ]]; then
		local nvm_dir="${NVM_DIR:-${HOME}/.nvm}"
		[[ -s "$nvm_dir/nvm.sh" ]] || fail "Node.js 20 is required and nvm was not found"
		set +u
		# shellcheck disable=SC1090
		source "$nvm_dir/nvm.sh"
		nvm use 20 >/dev/null
		set -u
	fi
	[[ "$(node -p 'process.versions.node.split(".")[0]')" == "20" ]] || fail "Node.js 20 activation failed"
	corepack enable
	corepack prepare pnpm@9 --activate >/dev/null
}

stop_candidate() {
	if [[ -n "$CANDIDATE_PID" ]] && kill -0 "$CANDIDATE_PID" 2>/dev/null; then
		kill -INT "$CANDIDATE_PID" 2>/dev/null || true
		wait "$CANDIDATE_PID" 2>/dev/null || true
	fi
	CANDIDATE_PID=""
}

cleanup() {
	stop_candidate
	rm -f "$VERIFY_BINARY"
}

trap cleanup EXIT INT TERM

activate_node20
for command_name in pnpm go gzip shasum curl grep awk sed sort seq stat sandbox-exec nc file; do
	require_command "$command_name"
done

step 1 "Build frontend with the frozen pnpm lockfile"
(
	cd "$FRONTEND_DIR"
	pnpm install --frozen-lockfile
	pnpm run build
)
[[ -s "$BACKEND_DIR/internal/web/dist/index.html" ]] || fail "frontend dist/index.html was not produced"

step 2 "Build native macOS candidate with embedded frontend"
mkdir -p "$LOCAL_TEST_DIR"
(
	cd "$BACKEND_DIR"
	CGO_ENABLED=0 go build -tags=embed -ldflags="-s -w" -o "$LOCAL_BINARY" ./cmd/server/
)
[[ -x "$LOCAL_BINARY" ]] || fail "native candidate was not produced"
file "$LOCAL_BINARY"

next_step=3
if [[ "$SKIP_SMOKE_TEST" == "0" ]]; then
	step 3 "Run isolated full-stack smoke test"
	[[ -s "$LOCAL_CONFIG" ]] || fail "missing isolated config: $LOCAL_CONFIG"
	[[ -s "$SANDBOX_PROFILE" ]] || fail "missing macOS network sandbox: $SANDBOX_PROFILE"
	if nc -z -w 1 127.0.0.1 18080; then
		fail "local smoke-test port 18080 is already in use"
	fi

	if sandbox-exec -f "$SANDBOX_PROFILE" nc -z -w 2 1.1.1.1 443; then
		fail "sandbox unexpectedly allows public network access"
	fi
	sandbox-exec -f "$SANDBOX_PROFILE" nc -z -w 2 127.0.0.1 5432 || fail "sandbox cannot reach local PostgreSQL"
	sandbox-exec -f "$SANDBOX_PROFILE" nc -z -w 2 127.0.0.1 6379 || fail "sandbox cannot reach local Redis"

	: >"$LOCAL_LOG"
	env DATA_DIR="$LOCAL_TEST_DIR" sandbox-exec -f "$SANDBOX_PROFILE" "$LOCAL_BINARY" >"$LOCAL_LOG" 2>&1 &
	CANDIDATE_PID=$!

	ready=0
	for _ in $(seq 1 90); do
		if curl -fsS --max-time 2 "$LOCAL_BASE_URL/health" >/dev/null 2>&1; then
			ready=1
			break
		fi
		kill -0 "$CANDIDATE_PID" 2>/dev/null || {
			tail -120 "$LOCAL_LOG" >&2
			fail "native candidate exited before becoming healthy"
		}
		sleep 1
	done
	[[ "$ready" == "1" ]] || {
		tail -120 "$LOCAL_LOG" >&2
		fail "native candidate health check timed out"
	}

	SMOKE_DIR="$LOCAL_TEST_DIR/release-smoke"
	mkdir -p "$SMOKE_DIR"
	for route in /health / /setup/status; do
		http_code="$(curl -sS -o "$SMOKE_DIR/response.json" -w '%{http_code}' --max-time 10 "$LOCAL_BASE_URL$route")"
		[[ "$http_code" == "200" ]] || fail "$route returned HTTP $http_code"
		[[ -s "$SMOKE_DIR/response.json" ]] || fail "$route returned an empty body"
	done

	curl -fsS --max-time 10 -D "$SMOKE_DIR/index.headers" "$LOCAL_BASE_URL/" -o "$SMOKE_DIR/index.html"
	grep -Eiq '^content-type: *text/html' "$SMOKE_DIR/index.headers" || fail "homepage is not HTML"
	grep -Eo '(src|href)="[^"]+\.(js|css)"' "$SMOKE_DIR/index.html" \
		| sed -E 's/^(src|href)="([^"]+)"$/\2/' \
		| sort -u >"$SMOKE_DIR/assets.list"
	[[ -s "$SMOKE_DIR/assets.list" ]] || fail "homepage referenced no JavaScript or CSS assets"
	while IFS= read -r asset; do
		[[ -n "$asset" ]] || continue
		case "$asset" in
		http://* | https://*) asset_url="$asset" ;;
		/*) asset_url="$LOCAL_BASE_URL$asset" ;;
		*) asset_url="$LOCAL_BASE_URL/$asset" ;;
		esac
		http_code="$(curl -sS -o "$SMOKE_DIR/asset.bin" -w '%{http_code}' --max-time 15 "$asset_url")"
		[[ "$http_code" == "200" ]] || fail "asset $asset returned HTTP $http_code"
		[[ -s "$SMOKE_DIR/asset.bin" ]] || fail "asset $asset returned an empty body"
	done <"$SMOKE_DIR/assets.list"

	stop_candidate
	echo "  Isolated /health, /, /setup/status, and referenced assets passed"
	next_step=4
else
	echo "  WARNING: isolated native smoke test explicitly skipped"
fi

step "$next_step" "Cross-compile Linux amd64 binary with embedded frontend"
(
	cd "$BACKEND_DIR"
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags=embed -ldflags="-s -w" -o "$LINUX_BINARY" ./cmd/server/
)
[[ -s "$LINUX_BINARY" ]] || fail "Linux binary was not produced"
file "$LINUX_BINARY" | grep -q 'ELF 64-bit.*x86-64' || fail "Linux candidate is not an amd64 ELF binary"

next_step=$((next_step + 1))
step "$next_step" "Create and verify compressed Linux artifact"
rm -f "$LINUX_ARTIFACT" "$LINUX_CHECKSUM" "$VERIFY_BINARY"
gzip -9 -c "$LINUX_BINARY" >"$LINUX_ARTIFACT"
raw_hash="$(shasum -a 256 "$LINUX_BINARY" | awk '{print $1}')"
printf '%s  sub2api-linux\n' "$raw_hash" >"$LINUX_CHECKSUM"
gzip -t "$LINUX_ARTIFACT"
gzip -dc "$LINUX_ARTIFACT" >"$VERIFY_BINARY"
verify_hash="$(shasum -a 256 "$VERIFY_BINARY" | awk '{print $1}')"
[[ "$verify_hash" == "$raw_hash" ]] || fail "gzip round-trip checksum mismatch"
artifact_size="$(stat -f '%z' "$LINUX_ARTIFACT")"
(( artifact_size < 100 * 1024 * 1024 )) || fail "compressed artifact exceeds GitHub's 100 MiB limit"
rm -f "$VERIFY_BINARY"

printf '  Linux raw SHA-256: %s\n' "$raw_hash"
printf '  Linux artifact: %s bytes\n' "$artifact_size"
printf '\n[OK] Verified artifacts are ready.\n'
printf 'Git staging, commit, push, and VPS deployment require separate confirmations.\n'
