# Sub2API Upgrade Runbook

This runbook records the verified production upgrade procedure established on
2026-07-11. A fresh agent/session must read it before any future upgrade.

## 1. Fixed environment facts

- Production SSH host: `goodserver`.
- Production repository: `/root/sub2api`.
- Production service: `sub2api.service`.
- Production Git origin: `https://github.com/whwh1233/sub2api.git`.
- Production pulls only; do not add a token or deploy key unless explicitly requested.
- Every production mutation, push, restart, deployment, or rollback is a
  separately confirmed step under `AGENTS.md`.

## 2. Fresh production database sync

Every upgrade test starts with a new live production dump. Never reuse an older
dump merely because it exists locally. The required entry point remains:

```powershell
.\.local-dev\sync-prod-db-local.ps1 -StartServer
```

Production access during sync is read-only; the local `sub2api` database is disposable.

### Proven fast transfer design

```text
pg_dump -Fd -j4 --compress=zstd:level=1
→ timestamped remote directory
→ uncompressed tar (contents are already compressed)
→ 25 MiB parts
→ at most 8 staggered SCP/SSH downloads
→ ordered local merge and SHA-256 verification
→ pg_restore -j4
```

For the roughly 3.8 GB production database, the parallel dump took 5–6 seconds,
the archive was about 373 MiB, eight-lane downloads took 42–66 seconds, and the
local restore took about 47 seconds. Four lanes took about 77 seconds. Sixteen
lanes were slower or failed intermittently.

`goodserver` uses `MaxStartups 10:30:100`; sixteen simultaneous handshakes can be
randomly rejected. Use eight lanes and stagger initial handshakes by about 300 ms.
Always merge in numeric part order and compare the original remote/local tar SHA.

The optimized remote-staging/multipart flow was proven during the release, but
`.local-dev` is Git-ignored. Inspect the actual local script before assuming all
optimizations are present. If absent, show the tooling change and obtain approval;
do not silently bypass the repository-required sync script.

## 3. Local restore and migration verification

1. Stop local test processes that can reconnect to PostgreSQL.
2. Drop and recreate only the local `sub2api` database.
3. Restore directory-format dumps with `pg_restore -j4`.
4. Start the candidate with background workers disabled.
5. Verify database size, public table count, and `schema_migrations`.

`schema_migrations` columns are `filename`, `checksum`, and `applied_at`; there is
no `version` column. “Migrated to 173” means migration file
`173_allow_cyber_blocked_usage_request_type.sql` was applied. It does not mean
PostgreSQL version 173 or 173 tables. Migrations are forward-only; binary rollback
does not automatically reverse them.

## 4. Embedded frontend is mandatory

Do not use this as a full-stack validation command:

```text
go run ./cmd/server
```

Without the `embed` tag, `/health` can return 200 while `/` returns 404. Use:

```text
go run -tags embed ./cmd/server
```

Production builds must use `go build -tags=embed` through the official script.
A candidate passes only when all of these succeed:

- `/health`: HTTP 200.
- `/`: HTTP 200 and `Content-Type: text/html`.
- `/setup/status`: HTTP 200.
- Every JavaScript and CSS asset referenced by the HTML: HTTP 200 and non-empty.

Never declare validation complete from `/health` alone.

## 5. Official build and compressed artifact

Run only from the repository root. Use the entry point for the current OS:

```powershell
.\deploy\local-build-push.ps1
```

```bash
./deploy/local-build-push.sh
```

On macOS, the script builds a native candidate and runs it against the fresh
local production database under a network sandbox that allows localhost only.
It then cross-compiles and verifies the Linux artifact. The macOS script stops
before Git staging; staging, commit, push, and deployment are separately
confirmed release steps.

The raw embedded Linux binary exceeded GitHub's 100 MiB limit. Tracked artifacts are:

```text
backend/sub2api-linux.gz
backend/sub2api-linux.sha256
```

The raw `backend/sub2api-linux` is ignored and must not be committed. The Windows
build script must build the frontend and Windows embed candidate, automatically
run it against `.local-server-prodtest` on an alternate loopback port with
background workers disabled, and apply the complete frontend gate. It then
builds Linux with `CGO_ENABLED=0` and `-tags=embed`, gzips it, records the raw
SHA, decompresses to a temporary file, and requires an exact SHA match.

Despite its historical `local-build-push.ps1` name, the script stops after local
artifact creation and verification. It must not stage, commit, push, or deploy;
those remain separately confirmed release steps under `AGENTS.md`. Linux
execution validation in Docker or the isolated `goodserver` flow in section 6
is still mandatory before deployment.

The successful 2026-07-11 raw binary was about 103.4 MiB and compressed to about
31.0 MiB. Gzip is lossless, but checksum equality alone is not an execution test.

## 6. Candidate execution tests

### Windows

Run the Windows candidate against the fresh local database on an alternate
loopback port with `SERVER_DISABLE_BACKGROUND_WORKERS=true`. Verify the complete
frontend gate in section 4. Never fall back to a config that might reference production.

### Linux locally

Use Docker Desktop Linux/x86_64 when available. Mount the gzip read-only,
decompress and verify SHA inside the container, execute it against the fresh local
database via `host.docker.internal`, use an alternate port, disable background
workers, and verify the complete frontend gate.

Minimal Alpine images require `tzdata` for `Asia/Shanghai`. An `unknown time zone`
failure is an incomplete test container, not binary corruption.

### Linux on goodserver without affecting production

Use a unique directory under:

```text
/var/tmp/sub2api-release-validation/<timestamp>/
```

Upload only gzip and checksum, decompress to a temporary candidate, verify SHA
and ELF amd64 headers, then execute for at most five seconds with a new empty data
directory, invalid database port, background workers disabled, and an unused
loopback-only high port. Do not call `systemctl` or replace the live binary.

The host has `gzip`, `sha256sum`, and `timeout`; it currently lacks `file` and
`readelf`, which are optional and must not become deployment dependencies.

## 7. Production deployment

Before pull, copy—never move—the live binary through a staging path to:

```text
/root/sub2api/backend/sub2api-linux.prev
```

Require live and backup SHA equality while the service remains active.

The verified `deploy/remote-pull-restart.sh` flow is:

1. Preserve the current binary as `.prev` while it is running.
2. `git pull origin main`.
3. Require gzip and the raw checksum.
4. `gzip -t`, decompress to `sub2api-linux.new`, and verify raw SHA.
5. Atomically rename the staged binary to the live path.
6. Restart `sub2api`.
7. Require active state; after replacement failure, restore `.prev` and restart.

During the first transition from tracked raw binary to compressed artifact, run
the guarded `git pull --ff-only` and newly pulled deploy script in one confirmed
SSH step. Do not pause with the tracked raw path deleted and no replacement installed.

Upgrade may briefly restart service. Do not infer permission for stop-first
rollback. Zero-stop rollback remains documented in
`docs/superpowers/plans/2026-07-11-goodserver-release-and-zero-stop-rollback.md`.

## 8. Production completion checks

Do not call a release complete until all are freshly verified:

- Production HEAD equals the intended commit.
- Installed binary SHA equals `backend/sub2api-linux.sha256`.
- systemd reports active/running, success, exit status 0, and no unexpected restarts.
- `/health` returns 200 repeatedly.
- `/` returns 200 HTML and its actual JS/CSS assets return 200.
- `/setup/status` reports setup complete.
- Latest migration records match the release.
- No new error-level journal entries appear after the new process starts.
- Real traffic has resumed where observable without exposing secrets or request bodies.

On 2026-07-11 the old process exited status 1 during stop, producing a transient
historical systemd failure record. The new process started the same second and
reported `Result=success`, `ExecMainStatus=0`, and `NRestarts=0`. Distinguish an
old-process shutdown record from a new-process startup failure.

## 9. Rollback and cleanup

- Preserve `sub2api-linux.prev` until retention expires.
- Binary rollback and database rollback are separate operations.
- Do not assume old code plus the migrated primary database is a complete rollback.
- For no-stop rollback, start old code against its rollback database on an
  alternate port, health-gate it, switch the proxy, then decide whether to stop
  the new service.
- Cleanup is destructive and separately confirmed. List exact paths and sizes first.
- Never delete `.prev` as part of generic temporary cleanup.

## 10. Known failure signatures

- `/health` 200 but `/` 404: missing `-tags embed`.
- GitHub rejects `sub2api-linux` over 100 MiB: raw binary was staged; use the official script.
- `Host key verification failed`: verify against GitHub's official fingerprints;
  never blindly trust `ssh-keyscan` output.
- `Permission denied (publickey)` on goodserver: production has no GitHub outbound
  private key; use its configured public HTTPS origin.
- SCP lanes exit 255 at sixteen-way concurrency: return to eight staggered lanes.
- `schema_migrations.version does not exist`: query `filename` and `applied_at`.
