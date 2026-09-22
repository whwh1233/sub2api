import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { ref } from 'vue'
import GroupHistoryPanel from '../GroupHistoryPanel.vue'
import messages from '@/i18n/locales/en/groupRealtime'
import type { GroupHistory } from '@/api/admin/groupRealtime'

const { fetchHistory } = vi.hoisted(() => ({ fetchHistory: vi.fn() }))
vi.mock('@/api/admin/groupRealtime', () => ({ getGroupHistory: fetchHistory }))
vi.mock('vue-chartjs', () => ({ Line: { name: 'Line', props: ['data', 'options'], template: '<div data-testid="history-chart" />' } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en'), t: (key: string) => key.split('.').reduce<unknown>((v, k) => (v as Record<string, unknown>)?.[k], messages) ?? key }) }))
function history(): GroupHistory {
  const result: GroupHistory = {
    start_time: '2026-09-22T11:00:00Z', end_time: '2026-09-22T12:00:00Z', collecting_since: '2026-09-22T11:00:00Z', bucket_seconds: 60, covered_seconds: 120, partial: true, group_id: null,
    groups: [
      { points: [], group_id: 1, group_name: 'Alpha', platform: 'openai', status: 'active', rpm: 1, started: 2, success: 1, failed: 1, cancelled: 20, success_rate: 50 },
      { points: [], group_id: 2, group_name: 'Beta', platform: 'claude', status: 'active', rpm: 9, started: 18, success: 8, failed: 0, cancelled: 0, success_rate: 100 },
    ],
    points: [
      { bucket_start: '2026-09-22T11:00:00Z', rpm: null, success_rate: null, partial: true, started: 0, success: 0, failed: 0, cancelled: 0 },
      { bucket_start: '2026-09-22T11:01:00Z', rpm: 2, success_rate: 50, partial: false, started: 2, success: 1, failed: 1, cancelled: 20 },
      { bucket_start: '2026-09-22T11:02:00Z', rpm: 18, success_rate: 100, partial: false, started: 18, success: 8, failed: 0, cancelled: 0 },
    ],
  }
  result.groups[0].points = result.points.map((p, i) => i === 2 ? { ...p, rpm: 0, started: 0, success: 0, failed: 0, cancelled: 0, success_rate: null } : { ...p })
  result.groups[1].points = result.points.map((p, i) => i === 1 ? { ...p, rpm: 0, started: 0, success: 0, failed: 0, cancelled: 0, success_rate: null } : { ...p })
  return result
}
describe('GroupHistoryPanel', () => {
  let wrapper: VueWrapper | undefined
  beforeEach(() => { vi.useFakeTimers(); fetchHistory.mockReset().mockResolvedValue(history()); Object.defineProperty(document, 'hidden', { configurable: true, value: false }) })
  afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.useRealTimers() })
  async function render() { wrapper = mount(GroupHistoryPanel); await flushPromises(); return wrapper }
  it('weights success by completions, uses covered time, sorts groups and preserves gaps', async () => {
    const view = await render()
    expect(view.get('[data-testid="history-rate"]').text()).toBe('90.0%')
    expect(view.get('[data-testid="history-average"]').text()).toBe('10')
    expect(view.find('tbody tr').text()).toContain('Beta')
    const chart = view.findComponent({ name: 'Line' })
    // Stub component keeps the actual data prop, including null gaps.
    const chartData = chart.props('data')
    expect(chartData.datasets.map((d: { label: string }) => d.label)).toEqual(['Alpha', 'Beta'])
    expect(chartData.datasets[0].data).toEqual([null, 2, 0])
    expect(chartData.datasets[1].data).toEqual([null, 0, 18])
    expect(chartData.datasets[0].borderColor).not.toEqual(chartData.datasets[1].borderColor)
  })
  it('toggles multiple groups and metrics locally without fetching again', async () => {
    const view = await render()
    const chart = view.findComponent({ name: 'Line' })
    const originalColor = chart.props('data').datasets[1].borderColor
    await view.get('[data-testid="history-legend-1"]').setValue(false)
    expect(chart.props('data').datasets.map((d: { label: string }) => d.label)).toEqual(['Beta'])
    expect((view.get('[data-testid="history-legend-1"]').element as HTMLInputElement).checked).toBe(false)
    await view.get('[data-testid="history-metric-success_rate"]').trigger('click')
    expect(chart.props('data').datasets[0].data).toEqual([null, null, 100])
    expect(chart.props('data').datasets[0].borderColor).toBe(originalColor)
    expect(fetchHistory).toHaveBeenCalledTimes(1)
    const reordered = history(); reordered.groups.reverse()
    fetchHistory.mockResolvedValue(reordered)
    await vi.advanceTimersByTimeAsync(30000)
    await flushPromises()
    expect(chart.props('data').datasets).toHaveLength(1)
    expect(chart.props('data').datasets[0].borderColor).toBe(originalColor)
    await view.get('[data-testid="history-legend-2"]').setValue(false)
    expect(view.text()).toContain('No groups selected')
    await view.findAll('button').find(b => b.text() === 'Select all')!.trigger('click')
    expect(chart.props('data').datasets).toHaveLength(2)
  })
  it('lists idle groups unchecked, allows selecting them, and resets defaults for a new range', async () => {
    const initial = history()
    const idle = { ...initial.groups[0], group_id: 3, group_name: 'Idle', started: 0, rpm: 0, points: initial.points.map(p => ({ ...p, rpm: p.partial ? null : 0, started: 0 })) }
    initial.groups.push(idle)
    fetchHistory.mockResolvedValue(initial)
    const view = await render()
    expect((view.get('[data-testid="history-legend-3"]').element as HTMLInputElement).checked).toBe(false)
    expect(view.find('tbody').text()).toContain('Idle')
    expect(view.findComponent({ name: 'Line' }).props('data').datasets).toHaveLength(2)
    await view.get('[data-testid="history-legend-3"]').setValue(true)
    expect(view.findComponent({ name: 'Line' }).props('data').datasets).toHaveLength(3)
    await vi.advanceTimersByTimeAsync(30000)
    await flushPromises()
    expect((view.get('[data-testid="history-legend-3"]').element as HTMLInputElement).checked).toBe(true)
    expect((view.get('[data-testid="history-legend-1"]').element as HTMLInputElement).checked).toBe(true)
    await view.get('[data-testid="history-legend-1"]').setValue(false)
    await view.get('[data-testid="history-range"]').setValue('360')
    await flushPromises()
    expect((view.get('[data-testid="history-legend-1"]').element as HTMLInputElement).checked).toBe(true)
    expect((view.get('[data-testid="history-legend-3"]').element as HTMLInputElement).checked).toBe(false)
    const empty = history(); empty.groups = [idle]
    fetchHistory.mockResolvedValue(empty)
    await view.get('[data-testid="history-range"]').setValue('15')
    await flushPromises()
    expect(view.findComponent({ name: 'Line' }).props('data').datasets).toHaveLength(0)
    expect(view.text()).toContain('No groups have RPM')
  })
  it('uses one time range for every group', async () => {
    const view = await render()
    expect(view.find('[data-testid="history-group"]').exists()).toBe(false)
    await view.get('[data-testid="history-range"]').setValue('10080')
    await flushPromises()
    expect(fetchHistory).toHaveBeenLastCalledWith(10080, null, expect.any(AbortSignal))
    expect(view.findComponent({ name: 'Line' }).props('data').datasets).toHaveLength(2)
  })
  it('retains stale results on errors and obeys pause and visibility', async () => {
    const view = await render()
    fetchHistory.mockRejectedValue(new Error('offline'))
    await vi.advanceTimersByTimeAsync(30000)
    await flushPromises()
    expect(view.text()).toContain('Data is stale')
    expect(view.find('tbody').text()).toContain('Alpha')
    await view.setProps({ paused: true })
    const calls = fetchHistory.mock.calls.length
    await vi.advanceTimersByTimeAsync(60000)
    expect(fetchHistory).toHaveBeenCalledTimes(calls)
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    await view.setProps({ paused: false })
    await vi.advanceTimersByTimeAsync(30000)
    expect(fetchHistory).toHaveBeenCalledTimes(calls)
  })
  it('ignores a stale request after the user selects another time range', async () => {
    let finish: (data: GroupHistory) => void = () => undefined
    fetchHistory.mockImplementationOnce(() => new Promise<GroupHistory>(resolve => { finish = resolve }))
    const view = await render()
    const newer = history(); newer.covered_seconds = 0; newer.groups = []; newer.points = []
    fetchHistory.mockResolvedValue(newer)
    await view.get('[data-testid="history-range"]').setValue('15')
    await flushPromises()
    finish(history())
    await flushPromises()
    expect(view.text()).toContain('No complete history yet')
    expect(view.find('tbody').text()).not.toContain('Alpha')
  })
})
