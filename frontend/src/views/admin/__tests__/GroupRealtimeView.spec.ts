import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { ref } from 'vue'
import GroupRealtimeView from '../GroupRealtimeView.vue'
import messages from '@/i18n/locales/en/groupRealtime'
import type { GroupRealtimeSnapshot } from '@/api/admin/groupRealtime'

const { fetchSnapshot } = vi.hoisted(() => ({ fetchSnapshot: vi.fn() }))
vi.mock('@/api/admin/groupRealtime', () => ({ getGroupRealtime: fetchSnapshot }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({
    locale: ref('en'),
    te: () => true,
    t: (key: string) => key.split('.').reduce<unknown>((value, part) => (value as Record<string, unknown>)?.[part], messages) ?? key,
  }),
}))

function snapshot(): GroupRealtimeSnapshot {
  return {
    start_time: '2026-09-17T12:00:00Z', end_time: '2026-09-17T12:01:00Z', collecting_since: '2026-09-17T11:00:00Z', partial: false,
    groups: [
      { group_id: 1, group_name: 'Alpha', platform: 'openai', status: 'active', rpm: 10, success: 1, failed: 1, cancelled: 3, success_rate: 50, failures: { timeout: 1 } },
      { group_id: 2, group_name: 'Beta', platform: 'claude', status: 'active', rpm: 5, success: 8, failed: 0, cancelled: 0, success_rate: 100, failures: {} },
      { group_id: 3, group_name: 'Idle', platform: 'openai', status: 'inactive', rpm: 0, success: 0, failed: 0, cancelled: 0, success_rate: null, failures: {} },
    ],
  }
}

describe('GroupRealtimeView', () => {
  let wrapper: VueWrapper | undefined
  beforeEach(() => {
    vi.useFakeTimers()
    fetchSnapshot.mockReset().mockResolvedValue(snapshot())
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  })
  afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.useRealTimers() })
  async function render() {
    wrapper = mount(GroupRealtimeView, {
      global: {
        stubs: { AppLayout: { template: '<main><slot /></main>' }, BaseDialog: { props: ['show'], template: '<aside v-if="show"><slot /></aside>' } },
      },
    })
    await flushPromises()
    return wrapper
  }
  it('weights the overall rate, excludes cancellations, and shows empty groups', async () => {
    const view = await render()
    expect(view.text()).toContain('90.0%') // 9 successes / 10 completions, not (50+100)/2
    expect(view.text()).toContain('No completed requests')
    expect(view.text()).toContain('Small sample')
    await view.find('input[type="search"]').setValue('Idle')
    expect(view.find('tbody').text()).not.toContain('Alpha')
    expect(view.find('tbody').text()).toContain('Idle')
  })
  it('preserves row order on refresh and freezes the failure detail window', async () => {
    const view = await render()
    await view.find('button[aria-label="Alpha · Failure reasons"]').trigger('click')
    expect(view.find('aside').text()).toContain('Timeout')
    const next = snapshot()
    next.groups[0].rpm = 1
    next.groups[0].failures = { internal: 4 }
    next.groups[1].rpm = 50
    fetchSnapshot.mockResolvedValue(next)
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(view.find('tbody tr').text()).toContain('Alpha')
    expect(view.find('aside').text()).toContain('Timeout')
    expect(view.find('aside').text()).not.toContain('Internal error')
  })
  it('keeps old data on failure and pauses polling while hidden', async () => {
    const view = await render()
    fetchSnapshot.mockRejectedValue(new Error('offline'))
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(view.text()).toContain('Data is stale')
    expect(view.find('tbody').text()).toContain('Alpha')
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    const calls = fetchSnapshot.mock.calls.length
    await vi.advanceTimersByTimeAsync(15000)
    expect(fetchSnapshot).toHaveBeenCalledTimes(calls)
  })
})
