import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { getDailyLeaderboard } from '@/api/usage'

describe('usage api daily leaderboard', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads the user-facing daily leaderboard endpoint', async () => {
    const response = {
      date: '2026-05-29',
      timezone: 'Asia/Shanghai',
      top: [],
      me: {
        rank: null,
        user_id: 42,
        display_name: 'User #42',
        total_tokens: 0,
        requests: 0,
        is_current_user: true,
        tokens_to_top_three: 1,
      },
    }
    get.mockResolvedValue({ data: response })

    await expect(getDailyLeaderboard()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/usage/leaderboard/daily')
  })
})
