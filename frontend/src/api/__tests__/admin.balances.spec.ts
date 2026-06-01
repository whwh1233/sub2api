import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import {
  getSummary,
  type AdminBalanceSummary,
} from '@/api/admin/balances'

describe('admin balances api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads the admin balance summary from the dedicated endpoint', async () => {
    const summary: AdminBalanceSummary = {
      total_balance: 88.5,
      positive_balance_users: 12,
      low_balance_users: 3,
      abnormal_balance_users: 1,
      low_balance_threshold: 1,
      generated_at: '2026-06-01T00:00:00Z',
    }
    get.mockResolvedValue({ data: summary })

    const result = await getSummary()

    expect(get).toHaveBeenCalledWith('/admin/balances/summary')
    expect(result).toEqual(summary)
  })
})
