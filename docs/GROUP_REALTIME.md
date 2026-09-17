# Group realtime monitoring

The administrator sidebar has a standalone `/admin/group-realtime` page.
It uses `GET /api/v1/admin/ops/group-realtime`, guarded by the existing admin
authentication and Ops monitoring switch. No migration or release artifact is
required for the source change.

## Metric definitions

- RPM: inference requests admitted to key lookup in the last rolling 60 seconds.
  The entry group is frozen immediately after key lookup, before routing changes
  it. Rejections of a recognized key still belong to that key's group. Requests
  without a recognized key are reported under group ID 0 on completion.
- Success rate: successes / (successes + final failures) completed in that
  window. In-flight requests and client cancellations are excluded. RPM and the
  completed total therefore need not match. Zero completions return JSON `null`.
- HTTP/SSE: final HTTP status, captured terminal SSE errors, explicit stream
  error markers, cancellation and cyber policy determine the outcome. Upstream
  attempts that recover are not separate requests.
- Responses WebSocket: count logical turns, not the handshake or connection.
  Retryable failures remain pending until retry success or connection exit.
  Idle connection close does not add a success; an unfinished turn does not
  become a success merely because the handshake returned 101.
- Metadata, token counting, task polling, cancellation endpoints, custom voice
  administration and live voice sessions are outside the inference counter.
  Async image/batch/video creation measures submission success, not eventual
  background task completion.
- Overall success rate is weighted by completed counts, never an average of
  group percentages. UI summary cards use the current search/platform filter.

## Collection and storage

Shared Redis keys `ops:group-realtime:v1:<unix-second>` hold bounded group / outcome
fields, expiring after 120 seconds. Redis TIME is the common clock. The API reads
the previous 60 complete seconds: `[floor(now)-60, floor(now))`, at most one second
behind wall time. It joins all non-deleted groups so inactive and idle groups are
still visible. Counters for a recently deleted group retain its numeric ID.
The `since` marker records the beginning of collection, without backfilling old
usage logs. The first incomplete window is labeled partial.

The collector does not store prompts, credentials, error bodies or individual
request identifiers. A failed Redis write does not reject inference; the local
instance exposes a partial-window warning for its collection gap and publishes
a shared one-minute gap marker when Redis recovers. Reads fail
visibly rather than returning synthetic zeroes. Admin snapshots are cached for
three seconds per instance; counters themselves are shared across instances.

## UI behavior

Poll every five seconds while visible and unpaused. Keep previous data on errors
and label it stale; also label data stale after 15 seconds without a successful
refresh. Do not reorder existing rows during polling. Clicking a sort control
explicitly reorders. Below 10 completed requests, show a small-sample hint.
Clicking a failure count opens reason counts from that frozen snapshot/window;
this is a reason breakdown, not a query of potentially filtered Ops error logs.
Mobile uses cards; desktop uses a sortable table. Chinese and English are included.

## Verification

Unit tests cover rolling boundaries, zero groups, cancellations, entry group
freezing, retry deduplication, WebSocket turns, terminal stream errors and collection
failures. UI tests cover weighted rates, snapshot drilldown, stable ordering,
stale state and hidden-page polling. Local validation must additionally use the
repository's ten-minute production sample, embedded frontend gate and actual
admin API checks. Never send test inference to a production-derived upstream.
