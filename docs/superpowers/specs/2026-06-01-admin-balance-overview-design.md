# Admin Balance Overview Design

## Summary

Add an administrator-only sidebar entry for a standalone balance overview page. The page gives admins a focused view of every user's wallet balance while preserving the existing balance source of truth and adjustment workflow.

## Goals

- Add a left sidebar admin menu item for "Balance Overview" / "余额总览".
- Show an admin-only page at `/admin/balances`.
- Display read-only balance summary metrics above a paginated user balance table.
- Reuse the existing user balance field, adjustment endpoints, and balance history flow.
- Keep the feature low-risk by avoiding a new balance table or duplicate stored balance.
- Validate the finished feature locally from a fresh production database sync before any release.

## Non-Goals

- Do not create a new balance storage table.
- Do not change usage billing, payment fulfillment, redeem-code redemption, refunds, or balance deduction behavior.
- Do not add export, batch balance edits, or finance-grade ledger reconciliation in this version.
- Do not expose the page or summary endpoint to non-admin users.

## User Experience

Add a sidebar item in the admin navigation:

- English: `Balance Overview`
- Chinese: `余额总览`
- Route: `/admin/balances`

The page uses the existing admin layout and table style. It contains:

- Summary cards for total balance, positive-balance users, low-balance users, and abnormal-balance users.
- Search and status filters.
- A paginated table with user ID, email, username, role, status, balance, last active/used time, and actions.
- Row actions for deposit, withdraw, and balance history by reusing the existing admin user balance modals or their logic.
- Loading, error, and empty states consistent with other admin pages.

The table should not introduce nested card-heavy layout. It should remain a dense operational admin page.

## Data Model

The only source of truth remains `users.balance`.

No migration is required for the first version. Existing balance changes continue through:

- Admin balance adjustment.
- Redeem-code application.
- Payment fulfillment and refund behavior.
- Usage billing deduction.
- Affiliate transfer into balance.

The new page only reads balances and invokes existing adjustment/history flows.

## Backend API

Add a new admin-only read endpoint under the existing admin route group:

```text
GET /api/v1/admin/balances/summary
```

Example response:

```json
{
  "total_balance": 1234.56,
  "positive_balance_users": 42,
  "low_balance_users": 7,
  "abnormal_balance_users": 0,
  "low_balance_threshold": 1.0,
  "generated_at": "2026-06-01T22:00:00+08:00"
}
```

Definitions:

- `total_balance`: sum of `users.balance` across non-deleted users.
- `positive_balance_users`: users with `balance > 0`.
- `low_balance_users`: active users with `balance > 0` and `balance <= low_balance_threshold`.
- `abnormal_balance_users`: users with `balance < 0`.
- `low_balance_threshold`: system low-balance threshold when available, otherwise a conservative default such as `1.0`.

Implementation should use one aggregate SQL query rather than loading all users into application memory.

The table continues to use the existing admin users list endpoint:

```text
GET /api/v1/admin/users
```

If a required table sort is not already supported, add only the narrow sort/filter support needed for the balance page.

## Backend Implementation

Add focused service/repository support for balance summary rather than placing SQL in handlers.

Suggested shape:

- Add a small `BalanceSummary` service model.
- Extend the user repository or add a narrowly scoped admin balance repository method for `GetBalanceSummary`.
- Add an admin handler method for `GetBalanceSummary`.
- Register `GET /admin/balances/summary` in `backend/internal/server/routes/admin.go`.

The endpoint must be inside the existing admin middleware group.

## Frontend Implementation

Add API client support under `frontend/src/api/admin`, either in a new `balances.ts` module or as a focused method on an existing admin API namespace.

Create:

```text
frontend/src/views/admin/BalancesView.vue
```

Add route:

```text
/admin/balances
```

Add the sidebar item in `AppSidebar.vue` near user management, because the feature is user-wallet focused.

Add i18n keys to English and Chinese locale files for:

- `nav.balances` or `nav.balanceOverview`
- `admin.balances.title`
- `admin.balances.description`
- summary card labels
- filter, empty, loading, refresh, and action labels not already covered by existing keys

Reuse existing formatting helpers for currency where possible.

## Permissions And Privacy

The route requires `requiresAuth: true` and `requiresAdmin: true`.

The API endpoint is registered only under the admin route group protected by admin middleware.

The page displays sensitive user balances, so it must not be linked from user navigation or public settings custom menus.

## Testing

Backend tests:

- Summary query returns correct totals, positive count, low-balance count, and negative-balance count.
- Admin route is protected by existing admin middleware coverage; add focused contract coverage if current tests do not cover the new route.

Frontend tests:

- API client calls `/admin/balances/summary`.
- Router contains `/admin/balances` with `requiresAdmin: true`.
- Sidebar contains the admin-only balance overview item.
- View renders summary cards and table rows from mocked API responses.

Validation:

- Before user-facing behavior testing, run `.\.local-dev\sync-prod-db-local.ps1 -StartServer` from the repository root.
- Confirm the script streams a new live production dump from `vps`, restores local `sub2api`, writes `.local-server-prodtest\config.yaml`, and starts the local backend.
- Use only the local service URL from that script for manual validation.
- Verify the new page, summary endpoint, search/filter/sort behavior, deposit/withdraw modal reuse, and balance history access against the fresh local production copy.

## Release Discipline

No online production release, restart, rollback, push-to-main, or VPS-side change should run autonomously. Each production step must be proposed and confirmed by the user before execution.
