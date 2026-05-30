import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import DailyLeaderboardView from '../DailyLeaderboardView.vue'

const { getDailyLeaderboard } = vi.hoisted(() => ({
  getDailyLeaderboard: vi.fn(),
}))

const messages: Record<string, string> = {
  'leaderboard.title': '每日榜单',
  'leaderboard.description': '看看今日 Token 使用排名，冲击前三名。',
  'leaderboard.today': '今日',
  'leaderboard.refresh': '刷新',
  'leaderboard.refreshing': '刷新中',
  'leaderboard.topThree': '前三名',
  'leaderboard.topTotal': '前三合计',
  'leaderboard.entryPace': '上榜门槛',
  'leaderboard.champion': '冠军',
  'leaderboard.runnerUp': '亚军',
  'leaderboard.thirdPlace': '季军',
  'leaderboard.chasingBoard': '追赶榜',
  'leaderboard.chasingBoardDescription': '第 4-10 名正在冲击领奖台。',
  'leaderboard.topTen': 'Top 10',
  'leaderboard.tokensBehindTopThree': '距离领奖台 {tokens} Token',
  'leaderboard.myRank': '我的排名',
  'leaderboard.currentUser': '你',
  'leaderboard.rank': '排名',
  'leaderboard.tokens': 'Token',
  'leaderboard.requests': '请求',
  'leaderboard.notRanked': '暂未上榜',
  'leaderboard.tokensToTopThree': '距离前三还差 {tokens} Token',
  'leaderboard.onLeaderboard': '你已在榜单中',
  'leaderboard.emptyTitle': '今日暂无用量',
  'leaderboard.emptyDescription': '开始调用 API 后就能参与今日榜单。',
  'leaderboard.loadFailed': '加载榜单失败',
  'leaderboard.loadFailedDetail': '暂时拿不到最新数据，可以留在当前页面重试。',
  'leaderboard.fullTokens': '完整值 {tokens} Token',
  'leaderboard.retry': '重试',
}

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDailyLeaderboard,
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        let value = messages[key] ?? key
        if (params) {
          for (const [paramKey, paramValue] of Object.entries(params)) {
            value = value.replace(`{${paramKey}}`, String(paramValue))
          }
        }
        return value
      },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }

describe('DailyLeaderboardView', () => {
  beforeEach(() => {
    getDailyLeaderboard.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders top three rows and highlights the current user', async () => {
    getDailyLeaderboard.mockResolvedValue({
      date: '2026-05-29',
      timezone: 'Asia/Shanghai',
      top: [
        { rank: 1, user_id: 2, display_name: 'beta', total_tokens: 900, requests: 9, is_current_user: false },
        { rank: 2, user_id: 42, display_name: 'me', total_tokens: 800, requests: 8, is_current_user: true },
        { rank: 3, user_id: 3, display_name: 'User #3', total_tokens: 300, requests: 5, is_current_user: false },
      ],
      me: {
        rank: 2,
        user_id: 42,
        display_name: 'me',
        total_tokens: 800,
        requests: 8,
        is_current_user: true,
        tokens_to_top_three: 0,
      },
    })

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('每日榜单')
    expect(text).toContain('beta')
    expect(text).toContain('900')
    expect(text).toContain('me')
    expect(text).toContain('你')
    expect(text).toContain('你已在榜单中')
  })

  it('shows the current user gap when outside the top three', async () => {
    getDailyLeaderboard.mockResolvedValue({
      date: '2026-05-29',
      timezone: 'Asia/Shanghai',
      top: [
        { rank: 1, user_id: 2, display_name: 'beta', total_tokens: 900, requests: 9, is_current_user: false },
        { rank: 2, user_id: 1, display_name: 'alpha', total_tokens: 800, requests: 8, is_current_user: false },
        { rank: 3, user_id: 3, display_name: 'gamma', total_tokens: 300, requests: 5, is_current_user: false },
      ],
      me: {
        rank: 4,
        user_id: 42,
        display_name: 'me',
        total_tokens: 100,
        requests: 2,
        is_current_user: true,
        tokens_to_top_three: 201,
      },
    })

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('距离前三还差 201 Token')
  })

  it('formats high-volume token values with compact K/M/B units', async () => {
    getDailyLeaderboard.mockResolvedValue({
      date: '2026-05-29',
      timezone: 'Asia/Shanghai',
      top: [
        { rank: 1, user_id: 2, display_name: 'beta', total_tokens: 1_250_000_000, requests: 2_832, is_current_user: false },
        { rank: 2, user_id: 1, display_name: 'alpha', total_tokens: 389_744_572, requests: 1_436, is_current_user: false },
        { rank: 3, user_id: 3, display_name: 'gamma', total_tokens: 12_400_000, requests: 788, is_current_user: false },
      ],
      me: {
        rank: 4,
        user_id: 42,
        display_name: 'me',
        total_tokens: 900_000,
        requests: 1200,
        is_current_user: true,
        tokens_to_top_three: 11_500_001,
      },
    })

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('1.25B')
    expect(text).toContain('389.7M')
    expect(text).toContain('12.4M')
    expect(text).toContain('前三合计')
    expect(text).toContain('距离前三还差 11.5M Token')
  })

  it('presents the top three as a podium with distinct champion, runner-up, and third-place icons', async () => {
    getDailyLeaderboard.mockResolvedValue({
      date: '2026-05-29',
      timezone: 'Asia/Shanghai',
      top: [
        { rank: 1, user_id: 2, display_name: 'beta', total_tokens: 1_250_000, requests: 900, is_current_user: false },
        { rank: 2, user_id: 1, display_name: 'alpha', total_tokens: 880_000, requests: 700, is_current_user: false },
        { rank: 3, user_id: 3, display_name: 'gamma', total_tokens: 500_000, requests: 500, is_current_user: false },
      ],
      me: {
        rank: 8,
        user_id: 42,
        display_name: 'me',
        total_tokens: 120_000,
        requests: 80,
        is_current_user: true,
        tokens_to_top_three: 380_001,
      },
    })

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    const podium = wrapper.find('[data-testid="leaderboard-podium"]')
    expect(podium.exists()).toBe(true)
    expect(podium.find('[data-podium-rank="1"].podium-place-champion').exists()).toBe(true)
    expect(podium.find('[data-rank-icon="champion"]').exists()).toBe(true)
    expect(podium.find('[data-rank-icon="runner-up"]').exists()).toBe(true)
    expect(podium.find('[data-rank-icon="third-place"]').exists()).toBe(true)

    const text = podium.text()
    expect(text).toContain('冠军')
    expect(text).toContain('亚军')
    expect(text).toContain('季军')
  })

  it('renders ranks four through ten as a chase list below the podium', async () => {
    const top = Array.from({ length: 10 }, (_, index) => {
      const rank = index + 1
      return {
        rank,
        user_id: rank,
        display_name: `runner-${rank}`,
        total_tokens: 1_100_000 - rank * 75_000,
        requests: 200 - rank,
        is_current_user: rank === 7,
      }
    })

    getDailyLeaderboard.mockResolvedValue({
      date: '2026-05-29',
      timezone: 'Asia/Shanghai',
      top,
      me: {
        rank: 7,
        user_id: 7,
        display_name: 'runner-7',
        total_tokens: top[6].total_tokens,
        requests: top[6].requests,
        is_current_user: true,
        tokens_to_top_three: top[2].total_tokens - top[6].total_tokens + 1,
      },
    })

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    const podium = wrapper.find('[data-testid="leaderboard-podium"]')
    const chasers = wrapper.find('[data-testid="leaderboard-chasers"]')

    expect(podium.findAll('[data-podium-rank]')).toHaveLength(3)
    expect(chasers.exists()).toBe(true)
    expect(chasers.text()).toContain('追赶榜')
    expect(chasers.text()).toContain('第 4-10 名正在冲击领奖台。')
    expect(chasers.find('[data-chaser-rank="4"]').text()).toContain('runner-4')
    expect(chasers.find('[data-chaser-rank="10"]').text()).toContain('runner-10')
    expect(chasers.find('[data-chaser-rank="3"]').exists()).toBe(false)
    expect(chasers.find('[data-chaser-rank="7"]').text()).toContain('你')
    expect(wrapper.text()).toContain('前三合计 2.85M Token')
  })

  it('renders an empty state when there is no usage today', async () => {
    getDailyLeaderboard.mockResolvedValue({
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
    })

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('今日暂无用量')
  })

  it('uses a skeleton instead of an empty spinner while the first request is pending', async () => {
    getDailyLeaderboard.mockReturnValue(new Promise(() => {}))

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.leaderboard-skeleton').exists()).toBe(true)
    expect(wrapper.text()).toContain('刷新中')
  })

  it('shows a calm retry state when the first request fails', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    getDailyLeaderboard.mockRejectedValue(new Error('boom'))

    const wrapper = mount(DailyLeaderboardView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('加载榜单失败')
    expect(wrapper.text()).toContain('暂时拿不到最新数据，可以留在当前页面重试。')
  })
})
