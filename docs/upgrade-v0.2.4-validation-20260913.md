# v0.2.4 upgrade validation — 2026-09-13

## Candidate and scope

- Base: `b6e1a02b8` (production and origin/main at inspection time).
- Upstream: release tag `v0.2.4`, commit `5de5e2bed`.
- Branch: `codex/upgrade-v0.2.4-20260913`.
- Backend VERSION and frontend package version aligned to `0.2.4`; the upstream tag still carried backend VERSION `0.2.3`.
- Production was not changed. Push and deployment require separate user confirmation.

## Merge decisions

- Resolved eight conflicted files. Preserved the site's GLM-5.2 and Kimi K3 pricing, prompt audit extensions, balance navigation, and prominent payment instructions.
- Combined the payment layout with sanitized Markdown rendering; avoided a duplicate help panel.
- Kept custom Claude headers/cache betas and adopted the upstream CLI 2.1.258 baseline and thinking binding beta.
- Preserved plugin archive close error handling and the upstream idempotent close guard.
- Adopted group model allowlists and simple-mode group navigation. Updated frontend mocks to the renamed API.
- Added missing Gemini/Antigravity landing locale keys and repaired two upstream test assumptions (provider count and auth store setup).
- Windows development environment now discovers Git's bundled `sh`, required by upstream backup-lock tests. Caches and temporary build files remain on D:.

## Production-derived local sample

The required sync script streamed a fresh ten-minute sample from goodserver, preserving schema and foundational data. This is not a full production backup.

- Dump: `.local-prod-db-backups/sub2api_live_20260913_214028_recent_10m.sql.gz`.
- SHA-256: `b339737d79a2a89e65eab9b235d5e80106a999a5cee388134bac986f8ef9d619`.
- Before migrations: 38 MB, 101 public tables, 232 usage rows from 21:30:35 through 21:40:33 China time.
- Applied migrations: `235_group_model_allowlist.sql`, `236_group_model_allowlist_repair.sql`, `237_add_minimax_platform.sql`.
- Grok Heavy (group 48) has the model list enabled. It becomes an enforced request allowlist after upgrade. Other seven nonempty stored lists are disabled.
- The sample has no group 48 traffic, so it cannot establish coverage of that group's actual client model names. Review its model allowlist before publishing; no group configuration was changed by this task.
- Migration 235 renames a column: reverting the binary alone does not reverse this database change.

## Verification

- Frontend type checking and production build passed.
- Frontend lint passed; later test/locale edits passed targeted lint.
- Full frontend suite: 275 files / 2,001 tests exercised. Initial run passed 1,999; the two failing files were corrected and all three tests in those files passed on rerun.
- Backend unit and integration suites completed. Initial repository failures were caused by missing `sh`; the full repository package passed again in both modes after correcting PATH. Other packages, including service, passed in the original runs.
- Integration database harness used its supported `SUB2API_TEST_POSTGRES_IMAGE=postgres:17-alpine` override, consistent with the previous upgrade's working local image.
- Backend golangci-lint v2.13.0: 0 issues.
- Prompt audit record-only persistence passed against the local production-derived PostgreSQL database, including full prompt storage and zero scanner calls; test rows were cleaned up.
- Portable deployment checks passed for Docker Compose security, runtime resources, and Caddy cache/stream handling. Apple script syntax passed; its macOS-specific permission test could not pass on Windows (`stat -f %Lp`), and was not represented as validated.
- Official `deploy/local-build-push.ps1` built Windows and Linux embed candidates and verified the gzip round trip.
- Windows and Linux/amd64 Docker candidates each passed health 200 three times, HTML root 200, setup complete, and all six referenced JS/CSS assets nonempty and 200.
- Both platforms logged `Background workers disabled for local validation`.
- Candidate version command reports `0.2.4`.
- Preview authenticated GET checks passed for admin groups, group 48, subscriptions, balance summary, and prompt audit config/runtime/events. Tokens and response bodies were not saved in the validation report.

## Artifacts and preview

- Compressed artifact: `backend/sub2api-linux.gz`, 38,281,480 bytes.
- Raw Linux SHA-256: `5222824bead2820262b3b7ebce2f08af5fefed211ca30fc9c3ab0214217c48d7`.
- The raw Linux binary remains ignored and must not be committed.
- Local preview: http://127.0.0.1:18081/ (Windows candidate, background workers disabled).
- Preview PID is recorded in `.local-run/upgrade-v024-preview.pid`.
- Logs and HTTP gate summaries: `.local-run/upgrade-v024-*`; Windows build gate logs also use `.local-run/release-windows.*.log`.

## Remaining release steps

Review the local preview and group 48 allowlist, then confirm publishing to origin/main. Production backup, pull/restart, and completion checks remain separately confirmed steps under AGENTS.md and docs/UPGRADE_RUNBOOK.md.
