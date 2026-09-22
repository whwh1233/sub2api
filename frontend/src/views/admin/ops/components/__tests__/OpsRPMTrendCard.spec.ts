import { describe, expect, beforeEach, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'

import OpsRPMTrendCard from '../OpsRPMTrendCard.vue'

const mockGetRPMTrend = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRPMTrend: (...args: any[]) => mockGetRPMTrend(...args)
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: ref('zh-CN')
    })
  }
})

vi.mock('vue-chartjs', () => ({
  Line: {
    props: ['data', 'options'],
    template: '<div class="line-chart" />'
  }
}))

const sampleResponse = {
  generated_at: '2026-08-05T09:01:00Z',
  start_time: '2026-08-05T08:00:00Z',
  end_time: '2026-08-05T09:00:00Z',
  complete_through: '2026-08-05T09:00:00Z',
  dimension: 'platform' as const,
  source_bucket_seconds: 60,
  output_bucket_seconds: 60,
  series: [
    {
      key: 'openai',
      label: 'openai',
      points: [
        {
          bucket_start: '2026-08-05T08:59:00Z',
          success_count: 12,
          error_count: 1,
          success_rpm: 12,
          error_rpm: 1,
          total_rpm: 13
        }
      ]
    }
  ]
}

describe('OpsRPMTrendCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetRPMTrend.mockResolvedValue(sampleResponse)
  })

  it('defaults to the one-hour platform trend', async () => {
    const wrapper = mount(OpsRPMTrendCard, {
      global: { stubs: { EmptyState: true, HelpTooltip: true } }
    })

    await flushPromises()

    expect(mockGetRPMTrend).toHaveBeenCalledWith(
      { time_range: '1h', dimension: 'platform', top_n: 10 },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(wrapper.find('.line-chart').exists()).toBe(true)
  })

  it('requests a new admin breakdown when the account tab is selected', async () => {
    const wrapper = mount(OpsRPMTrendCard, {
      global: { stubs: { EmptyState: true, HelpTooltip: true } }
    })
    await flushPromises()

    const accountTab = wrapper.findAll('[role="tab"]').find((tab) =>
      tab.text().includes('admin.ops.rpm.dimensions.account')
    )
    expect(accountTab).toBeDefined()
    await accountTab!.trigger('click')
    await flushPromises()

    expect(mockGetRPMTrend).toHaveBeenLastCalledWith(
      { time_range: '1h', dimension: 'account', top_n: 10 },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })

  it('switches to the retained two-hour rollup for thirty days', async () => {
    mockGetRPMTrend.mockResolvedValue({
      ...sampleResponse,
      source_bucket_seconds: 300,
      output_bucket_seconds: 7200
    })
    const wrapper = mount(OpsRPMTrendCard, {
      global: { stubs: { EmptyState: true, HelpTooltip: true } }
    })
    await flushPromises()

    const rangeButton = wrapper.findAll('button').find((button) => button.text() === '30d')
    expect(rangeButton).toBeDefined()
    await rangeButton!.trigger('click')
    await flushPromises()

    expect(mockGetRPMTrend).toHaveBeenLastCalledWith(
      { time_range: '30d', dimension: 'platform', top_n: 10 },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(wrapper.text()).toContain('admin.ops.rpm.bucketTwoHours')
  })
})
