# 管理员余额总览设计

## 概要

新增一个仅管理员可见的左侧独立菜单，用于进入“余额总览”页面。页面集中展示所有用户的钱包余额，同时继续使用现有余额字段和现有余额调整流程，避免引入第二份余额数据。

## 目标

- 在管理员左侧菜单新增“余额总览”入口。
- 新增仅管理员可访问页面 `/admin/balances`。
- 在分页用户余额表上方展示只读余额汇总指标。
- 复用现有用户余额字段、充值/扣款接口和余额历史流程。
- 不新增余额表，不复制余额字段，降低数据不一致风险。
- 功能完成后，必须先从生产库同步一份新的本地数据再验证。

## 非目标

- 不创建新的余额存储表。
- 不改动用量扣费、支付履约、兑换码兑换、退款或余额扣减链路。
- 第一版不做导出、批量余额调整或财务级流水对账。
- 不向非管理员暴露页面或汇总接口。

## 用户体验

在管理员导航中新增菜单项：

- 中文：`余额总览`
- 英文：`Balance Overview`
- 路由：`/admin/balances`

页面使用现有管理员布局和表格风格，包含：

- 汇总卡片：总余额、正余额用户数、低余额用户数、异常余额用户数。
- 搜索、状态筛选和余额状态筛选。
- 分页表格：用户 ID、邮箱、用户名、角色、状态、余额、最近活跃时间、最近使用时间和操作。
- 行操作：充值、扣款、余额历史，复用现有用户管理页里的余额弹窗或对应逻辑。
- 加载、错误、空状态，保持和其它管理员页面一致。

表格保持偏运营后台的紧凑信息密度，不做多层卡片嵌套。

## 数据模型

余额唯一真实来源仍然是 `users.balance`。

第一版不需要数据库迁移。现有余额变动继续通过以下路径发生：

- 管理员余额调整。
- 兑换码到账。
- 支付履约和退款逻辑。
- 用量计费扣减。
- 邀请返利转入余额。

新页面只读取余额，并调用现有调整和历史能力。

## 后端接口

在现有管理员路由组下新增只读接口：

```text
GET /api/v1/admin/balances/summary
```

响应示例：

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

字段口径：

- `total_balance`：所有未删除用户的 `users.balance` 总和。
- `positive_balance_users`：`balance > 0` 的用户数。
- `low_balance_users`：状态为 active，且 `balance > 0`、`balance <= low_balance_threshold` 的用户数。
- `abnormal_balance_users`：`balance < 0` 的用户数。
- `low_balance_threshold`：优先使用系统余额不足提醒阈值；无法取得时使用保守默认值，例如 `1.0`。

实现时使用单条聚合 SQL，不把全量用户加载到应用内存再统计。

表格继续使用现有管理员用户列表接口：

```text
GET /api/v1/admin/users
```

余额页在该接口上补充最小筛选能力：

- `balance_state=positive`：`balance > 0`。
- `balance_state=low`：状态为 active，且 `balance > 0`、`balance <= low_balance_threshold`。
- `balance_state=abnormal`：`balance < 0`。
- `balance_state=zero`：`balance = 0`。

低余额筛选和汇总卡片使用同一阈值来源。

## 后端实现

新增聚焦的 service/repository 支持，不把 SQL 放到 handler 中。

建议结构：

- 新增小型 `BalanceSummary` service 模型。
- 扩展用户 repository，或新增窄口径管理员余额 repository 方法 `GetBalanceSummary`。
- 新增管理员 handler 方法 `GetBalanceSummary`。
- 在 `backend/internal/server/routes/admin.go` 注册 `GET /admin/balances/summary`。

接口必须位于现有 admin middleware 保护的管理员路由组内。

## 前端实现

在 `frontend/src/api/admin` 下新增 API 客户端支持，可以新建 `balances.ts`，也可以在现有 admin API 命名空间中加入聚焦方法。

新增页面：

```text
frontend/src/views/admin/BalancesView.vue
```

新增路由：

```text
/admin/balances
```

在 `AppSidebar.vue` 管理员菜单中新增入口，位置靠近用户管理，因为该功能面向用户钱包管理。

新增中英文 i18n 文案，包括：

- `nav.balances` 或 `nav.balanceOverview`
- `admin.balances.title`
- `admin.balances.description`
- 汇总卡片标签
- 现有文案无法覆盖的筛选、空状态、加载、刷新和操作标签

金额格式化优先复用现有工具函数。

## 权限与隐私

前端路由必须设置：

- `requiresAuth: true`
- `requiresAdmin: true`

后端接口只注册在受管理员中间件保护的 admin 路由组下。

页面会展示敏感的用户余额，因此不能出现在普通用户导航，也不能通过公开自定义菜单暴露。

## 测试

后端测试：

- 汇总查询能正确返回总余额、正余额用户数、低余额用户数和负余额用户数。
- 管理员路由权限由现有 admin middleware 覆盖；如果现有契约测试未覆盖新路由，需要补充聚焦测试。

前端测试：

- API 客户端调用 `/admin/balances/summary`。
- Router 包含 `/admin/balances`，并设置 `requiresAdmin: true`。
- 侧边栏包含仅管理员可见的余额总览入口。
- 页面能基于 mocked API 响应渲染汇总卡片和表格行。

## 本地验证

实现完成后，在任何用户可见行为验证前，必须从仓库根目录运行：

```powershell
.\.local-dev\sync-prod-db-local.ps1 -StartServer
```

验证要求：

- 脚本必须从 `vps` 流式拉取新的生产库 `pg_dump`。
- 脚本恢复本地 `sub2api` 数据库。
- 脚本写入 `.local-server-prodtest\config.yaml`。
- 脚本启动本地后端。
- 只使用该脚本给出的本地服务地址做页面和接口验证。
- 验证余额总览页面、汇总接口、搜索/筛选/排序、充值/扣款弹窗复用、余额历史入口都能在这份 fresh 生产库副本上正常工作。

## 发布纪律

任何线上生产发布、重启、回滚、push 到 main 或 VPS 侧变更，都不能自主跑完整流程。每一步必须先提出具体操作，等用户明确确认后再执行。
