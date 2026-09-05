# v0.2.0 local upgrade verification — 2026-09-05

Production baseline: `335616de865bab92d0d80cb113a23183ae7ec5f3` on `goodserver`.
Upstream release: `v0.2.0`, commit `aa236488351eb71e120fc2b6fb32e36b0374c918`.
Local branch: `codex/upgrade-v0.2.0-20260905`.

The previous uncommitted request-capture feature and reports are preserved in
stash `8edb0c4fdcc3ff2e73ecf9cc67e1155829b1eae9` (19 modified and 104 untracked
files). They are excluded from this upgrade.

## Dataset

The user explicitly requested a small dataset to check migrations instead of
waiting for all historical audit payloads. A fresh production export was made at
2026-09-05 14:55:20 +08:00 through the local sync script's `-MigrationSample`
option, with no remote staging files or production database writes.

- Complete schema and remaining business/configuration data were restored.
- Historical `prompt_audit_events`, `prompt_audit_jobs`, `usage_billing_dedup`,
  `usage_billing_dedup_archive`, and `billing_usage_entries` rows were excluded.
- `usage_logs` was restored with 2,000 recent production rows older than one hour.
  These rows were imported before starting the new candidate and its migrations.
- The core dump and usage sample are separate fresh reads, not a single shared
  transaction snapshot. This dataset is for migration/functional checks, not
  financial reconciliation or production-scale performance measurements.
- Core dump: 28,843,127 bytes; SHA-256
  `504445e09a687de0026bf7519fbcfaa459b10216c42ad8e8e3a4faa15b45f324`.
- Usage sample SHA-256:
  `2277923b861d920a065db19d405b589cda22f9dbd73e7dd4eccd70b928333539`.
- Migration records increased from 268 to 277; public tables increased from 99
  to 101. All migration filenames are present. Historical audit tables are empty;
  the stashed `full_request_captures` table is absent.

The nine new SQL migrations do not alter `prompt_audit_events`. Its table and
`full_prompt` column both originated upstream; our existing customizations add
record-only operation and list excerpts.

## Merge adjustments

- Preserve the production customizations, including record-only prompt audit,
  exact-time usage filtering, Claude Code group restrictions, and custom pricing.
- Resolve five text conflicts while retaining both exact-time and native
  compaction filters and their tests.
- Set embedded VERSION to `0.2.0`; upstream's release tag contained `0.1.185`.
- Implement the validation-only `SERVER_DISABLE_BACKGROUND_WORKERS` switch.
  Prompt audit still loads configuration, but does not consume queued jobs.
- Close plugin ZIP readers before renaming their files, fixing Windows upload
  failures; keep Unix executable-mode assertions scoped to Unix.
- Update two frontend tests for the existing Pinia dependency and scroll layout.

## Completed checks

- Frontend lint and TypeScript checks passed. The full Vitest run covered 259
  files / 1,853 tests; the two outdated tests were corrected, and all four tests
  in their affected files passed on rerun.
- `go test -p 2 -tags=unit -timeout=15m ./...` passed.
- `go test -p 2 -tags=integration -timeout=15m ./...` passed with
  `SUB2API_TEST_POSTGRES_IMAGE=postgres:17-alpine`.
- golangci-lint 2.13 reported zero issues; affected packages were rechecked after
  subsequent fixes.
- The repository's `deploy/local-build-push.ps1` built the final artifacts.
- Windows and Linux/amd64 candidates passed repeated health checks, HTML root,
  completed setup status, and all six referenced nonempty JS/CSS assets.
- Linux ran locally in Docker with the production sample database, not on the VPS.
  Windows checksum-file CRLF was normalized before BusyBox checksum verification.
- Local admin API checks passed for version `0.2.0`, new group policies, native
  compaction filtering, exact-time usage queries, and custom record-only config.
  A temporary local admin API key was restored/removed after each check.
- Docker Compose security, gateway environment, runtime resource, and Caddy
  cache-policy shell checks passed using an LF source archive.
- The macOS deployment script passed syntax checking. Its BSD-stat-dependent
  runtime test requires the macOS CI runner and is not validated by Linux.

Final Linux raw SHA-256:
`45dac6a7f1375a6e0c8013ed96e4194674d89a9d12d4656e4a13ae98310bd87a`.
The gzip round-trip matched; the compressed artifact is approximately 35.9 MiB.

Local preview: <http://127.0.0.1:8080>. Startup confirms background workers are
disabled. This preview uses the reduced dataset; its historical totals must not
be treated as a complete copy of production reporting.

No production restart, deployment, or Git push was performed. Production release
steps still require separate confirmation under AGENTS.md.
