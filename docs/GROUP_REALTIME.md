# Group realtime monitoring

The administrator sidebar has a standalone `/admin/group-realtime` page.
It uses `GET /api/v1/admin/ops/group-realtime`, guarded by the existing admin
authentication and Ops monitoring switch. Migration
`238_ops_group_minute_metrics.sql` adds persistent history storage.

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
refresh. Default to RPM descending and re-sort on every successful refresh. Clicking a
sort control selects another order, also maintained on refresh. Below 10 completed requests, show a small-sample hint.
Clicking a failure count opens reason counts from that frozen snapshot/window;
this is a reason breakdown, not a query of potentially filtered Ops error logs.
Mobile uses cards; desktop uses a sortable table. Chinese and English are included.

## Verification

Unit tests cover rolling boundaries, zero groups, cancellations, entry group
freezing, retry deduplication, WebSocket turns, terminal stream errors and collection
failures. UI tests cover weighted rates, snapshot drilldown, descending ordering after refresh,
stale state and hidden-page polling. Local validation must additionally use the
repository's ten-minute production sample, embedded frontend gate and actual
admin API checks. Never send test inference to a production-derived upstream.

## History and recent group success rates

`GET /api/v1/admin/ops/group-realtime/history?window_minutes=60&group_id=123`
uses the same admin authentication and monitoring switch. Supported windows are
15, 60, 360, and 1440 minutes. Omit `group_id` for all groups; `group_id=0` selects
unassigned traffic. Invalid windows and negative/non-numeric IDs return 400.
Each group includes aligned history points, so a single request supplies all
chart lines. The optional group_id still filters aggregate points for API clients;
the UI requests all groups and toggles lines locally.

The existing atomic Redis increment updates expiring source counters. The Ops
collector persists closed one-minute buckets to PostgreSQL every minute, even
when nobody has the page open. Its startup pass catches up available Redis data
(up to 24 hours) and later passes overlap five minutes. It respects the Ops
monitoring switches and the local background-worker disable flag.

`ops_group_minute_metrics` has primary key `(bucket_start, group_id)`, snapshots
of group name/platform/status, entered/success/failure/cancellation counts,
`partial`, and `updated_at`. Group ID -1 is a per-minute coverage marker in the
same transaction; ID 0 remains unassigned traffic. Quiet groups do not need
individual empty rows. There is no foreign key, so deleting a group preserves
its history. There is no automatic history deletion or retention job yet.

A PostgreSQL transaction advisory lock serializes multi-instance writers. Batch
UPSERT writes absolute counters rather than adding them, so retries are
idempotent; a failure rolls back the entire batch including coverage. Reported
outage intervals also mark earlier stored buckets incomplete. After a Redis
reset, the new collection epoch never overwrites older persisted minutes.

The history API reads only PostgreSQL and stays available if Redis fails or is
cleared. The four query windows remain 15 minutes, 1 hour, 6 hours, and 24 hours;
longer ranges use five-minute display buckets assembled from stored minutes.
A bucket is complete only when every constituent minute has a complete coverage
marker. Missing and not-yet-persisted minutes remain gaps, not zero traffic.
Persistence can trail the realtime view by roughly one to two minutes. Loss of
Redis before a bucket is persisted can still lose that unpersisted interval;
PostgreSQL persistence does not promise per-request exactly-once delivery.

RPM is arrivals divided by the duration in minutes, not completed request count.
Success rate is successes / (successes + final failures); cancellations are
excluded and no completions produce null. Period totals sum complete buckets;
average RPM divides by fully covered time, and overall success rate is weighted
by counts. Per-group history is ordered by average RPM descending, then group ID.

Collection starts when this version first records an event or the persistence
worker initializes it. Existing retained counters can be caught up on startup. Previous traffic is not backfilled from usage/error logs because
those logs do not have the same entry-group, retry, and event-time semantics.
Buckets before collection starts, the first partially collected bucket, and
recorded outage intervals are gaps: RPM and success rate are null, never zero.
Their counts are excluded from period summaries and `covered_seconds`. After
Redis recovers, the collector shares the unavailable interval across instances.
As with realtime, simultaneous eviction of data without the collection marker
cannot be distinguished from no traffic; use a Redis policy appropriate for
retaining monitoring data.

The history panel refreshes every 30 seconds, obeys page visibility and the
page-wide pause button, and marks stale data. One shared time range controls all
groups. The chart defaults to RPM with one colored line per group; the metric
buttons switch all lines to success rates without an API request. Accessible
legend buttons toggle groups individually; Show all / Hide all manage visibility.
Colors and hidden groups remain stable through polling and RPM reordering.
Missing collection remains a gap, and idle groups have zero RPM but null rates.
Summary cards always describe all groups, independent of hidden chart lines.
Search in the history table is independent of the realtime table filters.

历史图表使用复选框列出所有分组，默认仅勾选所选范围内至少一个完整时间桶 RPM 大于 0 的分组。无流量分组默认不勾选，仍可手动选中显示曲线。自动刷新保留手动选择；切换时间范围恢复默认选择。真实采集缺口仍以断点表示，完整采集且无请求的时间桶显示 0。

历史查询最长支持最近 7 天（10080 分钟），按 15 分钟聚合，共 672 个点。仍按分钟持久保存原始计数；查询范围限制不触发数据删除。Redis 的短期缓存仅用于采集与落库，周历史直接从 PostgreSQL 查询。

## 发布前待完成的风险验证

当前仅完成本地功能与持久化测试，尚未完成生产规模压测及数据库故障演练，不应视为可直接发布。
统计与业务仍共用 PostgreSQL 连接池；后台落库设置 20 秒 context 超时，但事务期间还会读取 Redis。请求链路同步写 Redis 计数，失败不拒绝请求，但可能增加延迟。历史表暂无自动清理。
发布前需评估统计连接资源隔离、数据库端 statement/lock 超时、独立停用及故障退避机制，并验证数据库断连、连接池饱和、Redis 延迟和一周数据规模下的业务影响。
