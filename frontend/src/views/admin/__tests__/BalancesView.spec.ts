import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import BalancesView from '../BalancesView.vue'

const { getSummary, listUsers } = vi.hoisted(() => ({
  getSummary: vi.fn(),
  listUsers: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    balances: {
      getSummary,
    },
    users: {
      list: listUsers,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.balances.summary.totalBalance': '总余额',
    'admin.balances.summary.positiveUsers': '有余额用户',
    'admin.balances.summary.lowBalanceUsers': '低余额用户',
    'admin.balances.summary.abnormalUsers': '异常余额用户',
    'admin.balances.searchPlaceholder': '搜索用户',
    'admin.balances.filters.allBalanceStates': '全部余额',
    'admin.balances.filters.positive': '有余额',
    'admin.balances.filters.low': '低余额',
    'admin.balances.filters.abnormal': '异常余额',
    'admin.balances.filters.zero': '零余额',
    'admin.balances.columns.user': '用户',
    'admin.balances.columns.balance': '余额',
    'admin.balances.columns.status': '状态',
    'admin.balances.columns.lastActive': '最后活跃',
    'admin.balances.columns.lastUsed': '最后使用',
    'admin.balances.columns.actions': '操作',
    'admin.balances.actions.deposit': '充值',
    'admin.balances.actions.history': '明细',
    'admin.balances.empty.title': '暂无余额数据',
    'common.active': '启用',
    'common.refresh': '刷新',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const mountView = () =>
  mount(BalancesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: {
          props: ['columns', 'data'],
          template: `
            <table>
              <thead><tr><th v-for="column in columns" :key="column.key">{{ column.label }}</th></tr></thead>
              <tbody>
                <tr v-for="row in data" :key="row.id">
                  <td v-for="column in columns" :key="column.key">
                    <slot :name="'cell-' + column.key" :row="row" :value="row[column.key]">
                      {{ row[column.key] }}
                    </slot>
                  </td>
                </tr>
              </tbody>
            </table>
          `,
        },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue', 'change'],
          template: `
            <select
              :value="modelValue"
              @change="$emit('update:modelValue', $event.target.value); $emit('change', $event.target.value)"
            >
              <option v-for="option in options" :key="String(option.value)" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          `,
        },
        Pagination: true,
        UserBalanceModal: true,
        UserBalanceHistoryModal: true,
        Icon: true,
      },
    },
  })

describe('admin BalancesView', () => {
  beforeEach(() => {
    getSummary.mockReset()
    listUsers.mockReset()
    getSummary.mockResolvedValue({
      total_balance: 88.5,
      positive_balance_users: 12,
      low_balance_users: 3,
      abnormal_balance_users: 1,
      low_balance_threshold: 1,
      generated_at: '2026-06-01T00:00:00Z',
    })
    listUsers.mockResolvedValue({
      items: [
        {
          id: 1,
          email: 'alpha@example.com',
          username: 'alpha',
          role: 'user',
          balance: 12.25,
          concurrency: 5,
          status: 'active',
          allowed_groups: [],
          balance_notify_enabled: true,
          balance_notify_threshold: null,
          balance_notify_extra_emails: [],
          created_at: '2026-06-01T00:00:00Z',
          updated_at: '2026-06-01T00:00:00Z',
          last_active_at: null,
          last_used_at: null,
          notes: '',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
  })

  it('loads summary metrics and the balance-focused user list', async () => {
    const wrapper = mountView()

    await flushPromises()

    expect(getSummary).toHaveBeenCalledTimes(1)
    expect(listUsers).toHaveBeenCalledWith(1, 20, expect.objectContaining({
      include_subscriptions: false,
      sort_by: 'balance',
      sort_order: 'desc',
    }))
    expect(wrapper.text()).toContain('总余额')
    expect(wrapper.text()).toContain('$88.50')
    expect(wrapper.text()).toContain('alpha@example.com')
    expect(wrapper.text()).toContain('$12.25')
  })

  it('passes the selected balance state to the user list request', async () => {
    const wrapper = mountView()

    await flushPromises()
    await wrapper.find('[data-testid="balance-state-filter"]').setValue('abnormal')
    await flushPromises()

    expect(listUsers).toHaveBeenLastCalledWith(1, 20, expect.objectContaining({
      balance_state: 'abnormal',
      include_subscriptions: false,
      sort_by: 'balance',
      sort_order: 'desc',
    }))
  })
})
