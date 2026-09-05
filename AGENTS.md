# Agent Operating Rules

## Windows Development Storage

Keep build caches, dependency downloads, temporary files, logs, and validation artifacts on D: on this workstation. Before Windows builds or tests, run `& .\deploy\local-dev-environment.ps1` to configure Go caches, Go dependencies, lint cache, and process-local TEMP/TMP under `D:\codex\.cache`. The release and database-sync scripts call this helper automatically. Do not change the Windows system TEMP/TMP directories or revert development caches to C:.

## Required Upgrade Runbook

Before planning or executing any `sub2api` upgrade, production database sync,
release, deployment, or rollback, read `docs/UPGRADE_RUNBOOK.md` completely.
Treat its verification gates and known pitfalls as mandatory unless the user
explicitly approves a different procedure.

## Production Host Target

For this repository, live `sub2api` services are on SSH host `goodserver`.
Use `goodserver` as the default and first production target for service status,
logs, database checks, deployment verification, and troubleshooting. Do not
probe `vps`, `myvps2`, or other SSH hosts for `sub2api` production service
state unless the user explicitly asks for those hosts.

## Production Data Sync Before Upgrade Testing

For routine development and feature validation, use a local production-derived
development database. Sync only the most recent **10 minutes** of time-series,
usage, error, and audit records; preserve complete schema and foundational data
(settings, users, accounts, groups, API keys, and related configuration).
Do not repeatedly sync during one development task: reuse the current local copy
for build/test iterations. Refresh the ten-minute sample when starting a new task
that needs current production records, or when the existing sample is insufficient.
Full production copies are no longer the default. Use a larger time window or a
full dump only when the user explicitly requests it or approves a concrete need.
This rule was explicitly changed by the user on 2026-09-05 for development efficiency.

Required flow:

1. Run `.\.local-dev\sync-prod-db-local.ps1 -StartServer` from the repository root.
2. The script defaults to a fresh ten-minute sample from `goodserver`, streamed directly without remote staging. Reuse that local sample throughout the task; do not describe it as a full production backup.
3. The script saves the new dump into `.local-prod-db-backups\`, records its checksum, drops and recreates the local `sub2api` database, restores the dump, writes `.local-server-prodtest\config.yaml`, and starts the local backend.
4. The local `sub2api` database is disposable for this workflow and may be overwritten.
5. Online database access in this workflow is read-only. The script must not depend on remote backup directory capacity for fresh data sync.

After the sync, use the local service for testing and give the user the local URL.

Do not report local validation complete from `/health` alone. The local backend
must be built or run with the `embed` build tag, and validation must include the
HTML root page plus its referenced JavaScript and CSS assets as described in
`docs/UPGRADE_RUNBOOK.md`.

## Production Release Confirmation

For any online production release, deployment, restart, rollback, push-to-main, or VPS-side change, do not execute the whole flow autonomously.

Required flow:

1. Present the next concrete release step to the user.
2. Wait for the user's explicit confirmation before running that step.
3. After the step finishes, report the result and the next proposed step.
4. Repeat until the user confirms the release is complete.

This applies even when the user says to start publishing. Treat that as permission to begin the guided release process, not permission to finish every production action without further confirmations.

## Production Build Script Discipline

Production packaging must use the repository release script for the current OS
from the repository root:

```powershell
.\deploy\local-build-push.ps1
```

```bash
./deploy/local-build-push.sh
```

Use the script's documented flags only when the user explicitly confirms that variant. Do not manually build, stage, commit, or push `backend/sub2api-linux` as the normal release path. If a manual build artifact was created during troubleshooting, treat it as a temporary local artifact and return to the scripted flow before continuing the release.

The macOS script intentionally stops after artifact creation and verification.
Git staging, commit, push, and production deployment remain separately confirmed
steps under the production release confirmation rules above.

The tracked release artifact is now `backend/sub2api-linux.gz` together with
`backend/sub2api-linux.sha256`; the raw `backend/sub2api-linux` is a local/runtime
file and must not be committed. Before deployment, require a gzip round-trip
SHA-256 match and actual Windows and Linux candidate startup tests.
