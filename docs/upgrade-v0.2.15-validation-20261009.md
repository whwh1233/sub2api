# v0.2.15 upgrade preparation — 2026-10-09

## Candidate and scope

- Production/local base: `076ba5e65` (v0.2.4 plus custom group RPM monitoring).
- Upstream release: `v0.2.15`, commit `f2669c8cf62555cd92389b3f55920e9e6e7c6ff2`.
- Local branch: `codex/upgrade-v0.2.15-20261009`.
- Backend VERSION and frontend package version are aligned to `0.2.15`; the upstream tag still carried backend VERSION `0.2.14`.
- Production application/configuration were not changed. The user authorized the local staging/commit step on 2026-10-10; push, production pricing changes, deployment and restart remain separately confirmed release steps.

## Merge decisions

- Resolved seven conflicted files. Retained group realtime/history monitoring, balance overview, community/leaderboard navigation, prompt audit extensions, GLM-5.2/Kimi K3 custom pricing, and prominent sanitized payment instructions.
- Combined custom Claude SDK/platform headers and cache betas with upstream runtime CLI version resolution and additional protocol compatibility betas.
- Kept custom navigation ordering while applying the new subscription visibility flag and billing-mode purchase label.
- Added the new Claude CLI version sync provider to the local background-worker disable policy, with a provider regression test.
- Preserved the site's short cooldown for opaque Gemini API-key 429s; updated the conflicting new upstream test. Explicit daily quota reset parsing is retained.
- Repaired Windows-sensitive upstream tests: server timing checks the measured application gap, model catalog cases use distinct cache keys, and the stale Ollama callback fixture uses a distinct reset generation.
- Fixed existing group-history row-close handling/type assertions and an equivalent RPM condition flagged by the new linter.

## Automated verification

- Frontend production build, locale completeness, and ESLint passed.
- Full frontend suite: 373 files, 2,836 tests. Initial run passed 2,835; the remaining sidebar mock was updated for the new feature-flag resolver and all 14 tests in that file passed on rerun.
- Backend unit suite: initial repository timing and three service tests failed as described above; the full repository and service packages passed after corrections. Other packages passed in the initial run.
- Backend integration suite: service and other packages passed. Initial repository startup failed in the Docker reaper; the complete repository suite passed with `SUB2API_TEST_POSTGRES_IMAGE=postgres:17-alpine` and `TESTCONTAINERS_RYUK_DISABLED=true`, including after final repository changes.
- golangci-lint v2.14.0 passed with zero issues. v2.13.0 could not decode Go 1.27 export data and was not counted as passing.
- All caches, temporary files, downloads and validation logs remained on D:.

## Data and migration verification

- The required sync script successfully streamed and restored a fresh ten-minute production-derived sample. Schema and foundational data were preserved; historical records were sampled. This is not a full production backup or a full-history migration/performance test.
- Successful dump: `.local-prod-db-backups/sub2api_live_20261009_221830_recent_10m.sql.gz`, 14,014,855 bytes.
- Dump SHA-256: `d392fb08a353095b7bf66ea4cbaaf291d8ce65f97b28d3494a2ce43b0c147886`.
- Direct SSH suffered substantial packet loss. Incomplete transfers were interrupted; the successful transfer used the workstation's already configured loopback HTTP proxy, keeping the `goodserver` SSH identity and host-key checks. No remote staging files or server configuration changes were needed.
- Local database after migration: 49 MB, 105 public tables, 591 usage rows and 635 audit event rows. Migration records increased from 288 to 296; all embedded SQL file checksums match the local database.
- Real PostgreSQL group realtime/history tests passed, without Redis dependency, including replay of the custom migrations.
- Prompt-audit record-only persistence passed: full prompt retained, scanner invoked zero times, temporary test rows cleaned up.
- Authenticated GET checks passed for groups, balance summary, prompt audit config/runtime/events, realtime group metrics, and all five history windows (15/60/360/1440/10080 minutes). Group visibility count matches the database. Anonymous access returned 401 and ordinary-user access returned 403.

The release adds eight SQL migration files: OpenCode platform support, deletion of unlimited quota placeholder rows, moderation provenance metadata, reasoning-effort multipliers, affiliate operation IDs, recharge bonus amounts, TypeSafe platform support, and removal of two platform constraints now enforced by the application registry. Existing custom RPM migrations keep their original filenames and contents.

## Artifacts

- Built through `deploy/local-build-push.ps1`, without flags.
- Windows embedded candidate reports `0.2.15`; fresh-sample health (three times), HTML root, setup complete and all six referenced nonempty JS/CSS assets passed.
- Linux gzip round trip passed.
- Raw Linux SHA-256: `b3ed56cff47b5e8edbcaf5124257180d532f770425c491c6bdfcce46124ac24b`.
- Compressed artifact: `backend/sub2api-linux.gz`, 38,944,288 bytes (approximately 37.1 MiB).
- The raw Linux executable is ignored and must not be committed.
- The Linux/amd64 candidate executed in a local `postgres:17-alpine` Docker runtime against the same production-derived local PostgreSQL/Redis. Decompressed SHA matched; health (three times), HTML root, setup complete and all six referenced assets passed. The temporary runtime container was removed after verification.
- Both platforms explicitly logged `Background workers disabled for local validation`.
- Validation preview: http://127.0.0.1:18081/ (Windows candidate, background workers disabled). It was no longer listening at the 2026-10-10 follow-up; start the local candidate again before expecting this URL to work.
- Local Windows startup could not fetch the remote model-price hash before its timeout. It successfully loaded 269 local/fallback models. The local catalog also warns that six Gemini cache-creation entries lack base cache prices; those fields remain zero unless overridden. No live paid inference request or end-to-end payment was performed.

## Handoff

This local merge commit packages the integrated code, validated compressed artifacts, explicit max-pricing plan and regression tests. Production remains at `076ba5e65` / v0.2.4 at the last read-only check. Remote push, the production pricing gate below, backups, deployment and completion checks still require confirmation in sequence.

High-traffic release acceptance remains separate from the functional tests: no sustained production-load test or long-stream drain/cutover test was performed. The live shutdown path waits only five seconds for HTTP requests, and the normal deploy script restarts a single instance. The 2026-10-10 read-only check found a 106 GB production database and a burst of 403/429 error logs. The new migrations target much smaller tables, but the 49 MB development sample does not establish full-scale lock/backup/rollback timings. Do not interpret this commit as approval for an immediate high-traffic restart.

## Release behavior to review

- Billing follow-up (2026-10-10): v0.2.4 implicitly applies an extra 3x multiplier to Claude Fable 5.1 at `max` reasoning effort. v0.2.15 removes that implicit default and only applies explicitly configured reasoning-effort multipliers. The user explicitly requested preserving the 3x tariff. `deploy/release-pricing-v0.2.15.json` records the legacy/new pricing cards and the three current Anthropic groups (7, 9, 47). The new card is installed only in the local production-derived database and preserves inherited token prices, group multipliers and long-context settings. Production pricing has not been changed.
- Mandatory release gate: in a separately confirmed production step, review existing pricing and explicitly set the legacy Fable 5.1 `max_reasoning_effort_multiplier: 3` before replacing the old service. Migration 239 converts this to `reasoning_effort_multipliers: {"max": 3}`; verify the result before accepting traffic. Do not postpone this until after the new service starts. Merge rather than replace unrelated pricing cards and review any additional model aliases. Do not apply 3x to all models.
- Pricing regressions passed: the explicit Fable card preserves a reconciled max charge of $0.43480575 and leaves the high-effort equivalent at $0.14493525; the dated-model variant matches too. A separate Haiku 5.5 long-context case (132,534 input/context tokens, above the 100,000 threshold) retains its $0.1037384375 charge at group multiplier 1.25. Its catalog tier is 5x, independent of the Fable max multiplier. Disabling long-context pricing reduces that case to $0.0207476875 as expected. Evidence: `.local-run/upgrade-v0215-pricing-record-regression.log` and `backend/internal/service/billing_upgrade_tariff_test.go`.
- API Key creation defaults: at most 200 active keys per user and 60 creations per hour, configurable.
- In-flight balance reservation is enabled by default; low-balance concurrent requests can be rejected earlier.
- Unused pre-upgrade password reset links become invalid.
- EasyPay callback forgery protection and other upstream security fixes are included.
- Migrations are forward-only; replacing the binary does not roll back the database.

## Evidence

Local logs and helper scripts are under `.local-run/upgrade-v0215-*`, `.local-run/validate-linux-upgrade-v0215.ps1`, and `.local-run/verify-upgrade-v0215.py`. No tokens or production request bodies are included in this report.
