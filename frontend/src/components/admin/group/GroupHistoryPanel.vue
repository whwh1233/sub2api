<template>
  <section class="space-y-4 rounded-xl border border-gray-200 bg-white p-4 sm:p-5 dark:border-dark-700 dark:bg-dark-800" aria-labelledby="group-history-title" :aria-busy="loading">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h2 id="group-history-title" class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ t('groupRealtime.history.title') }}</h2>
        <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">{{ t('groupRealtime.history.description') }}</p>
      </div>
      <div class="flex w-full flex-wrap gap-3 sm:w-auto">
        <label class="flex min-w-0 flex-1 flex-col gap-1 text-sm sm:flex-none">
          {{ t('groupRealtime.history.range') }}
          <select v-model="windowMinutes" data-testid="history-range" class="input min-h-11">
            <option v-for="minutes in ranges" :key="minutes" :value="minutes">{{ t(`groupRealtime.history.ranges.${minutes}`) }}</option>
          </select>
        </label>

      </div>
    </div>
    <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ t('groupRealtime.history.error') }}</p>
    <p v-if="data?.partial" role="status" class="rounded-lg bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">{{ t('groupRealtime.history.partial') }}</p>
    <p v-if="loading && !data" role="status" class="py-8 text-center text-sm text-gray-500">{{ t('groupRealtime.history.loading') }}</p>
    <template v-if="data">
      <div class="flex flex-wrap gap-x-5 gap-y-2 text-xs text-gray-500 dark:text-gray-400">
        <span>{{ date(data.start_time) }} – {{ date(data.end_time) }}</span>
        <span>{{ t('groupRealtime.history.bucket', { minutes: data.bucket_seconds / 60 }) }}</span>
        <span>{{ t('groupRealtime.history.coverage', { minutes: data.covered_seconds / 60 }) }}</span>
        <span v-if="stale" class="text-amber-700 dark:text-amber-400">{{ t('groupRealtime.stale') }}</span>
      </div>
      <div v-if="data.covered_seconds > 0" class="space-y-4">
        <dl class="grid grid-cols-2 gap-4 rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
          <div><dt class="text-sm text-gray-500">{{ t('groupRealtime.history.averageRPM') }} · {{ t('groupRealtime.history.allGroups') }}</dt><dd class="mt-1 text-2xl font-semibold tabular-nums" data-testid="history-average">{{ number(averageRPM) }}</dd></div>
          <div><dt class="text-sm text-gray-500">{{ t('groupRealtime.history.rate') }} · {{ t('groupRealtime.history.allGroups') }}</dt><dd class="mt-1 text-2xl font-semibold tabular-nums" data-testid="history-rate">{{ rate(successRate) }}</dd></div>
        </dl>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex gap-2" :aria-label="t('groupRealtime.history.metric')">
            <button v-for="item in metrics" :key="item" type="button" class="btn min-h-11" :class="metric === item ? 'btn-primary' : 'btn-secondary'" :aria-pressed="metric === item" :data-testid="`history-metric-${item}`" @click="metric = item">{{ item === 'rpm' ? 'RPM' : t('groupRealtime.history.rate') }}</button>
          </div>
          <div class="flex gap-3 text-sm">
            <button type="button" class="min-h-11 text-primary-600 dark:text-primary-400 hover:underline" @click="setAllGroups(true)">{{ t('groupRealtime.history.showAll') }}</button>
            <button type="button" class="min-h-11 text-gray-500 hover:underline" @click="setAllGroups(false)">{{ t('groupRealtime.history.hideAll') }}</button>
          </div>
        </div>
        <div class="flex max-h-48 flex-wrap gap-2 overflow-y-auto" :aria-label="t('groupRealtime.history.legend')">
          <label v-for="group in data.groups" :key="group.group_id" class="flex min-h-11 max-w-full cursor-pointer items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-left text-xs dark:border-dark-700">
            <input type="checkbox" :data-testid="`history-legend-${group.group_id}`" :checked="isSelected(group)" class="h-4 w-4 rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500" :style="{ accentColor: colorFor(group.group_id) }" @change="toggleGroup(group.group_id)" />
            <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: colorFor(group.group_id) }" aria-hidden="true" />
            <span class="break-words" :class="!isSelected(group) ? 'text-gray-400' : 'text-gray-700 dark:text-gray-200'">{{ name(group) }}</span>
          </label>
        </div>
        <p class="text-xs text-gray-500">{{ t('groupRealtime.history.selectionHint') }}</p>
        <p v-if="!activeGroups.length" role="status" class="text-sm text-gray-500">{{ t('groupRealtime.history.noTraffic') }}</p>
        <p v-else-if="!visibleGroups.length" role="status" class="text-sm text-gray-500">{{ t('groupRealtime.history.noneVisible') }}</p>
        <div class="h-64 sm:h-80" role="img" :aria-label="t('groupRealtime.history.chartLabel')">
          <Line :data="chartData" :options="chartOptions" />
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('groupRealtime.history.chartHint') }}</p>
      </div>
      <p v-else class="rounded-lg bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:bg-dark-900">{{ t('groupRealtime.history.warming') }}</p>
      <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h3 class="font-medium">{{ t('groupRealtime.history.groupsTitle') }}</h3>
          <label class="w-full sm:w-64"><span class="sr-only">{{ t('groupRealtime.search') }}</span><input v-model="search" type="search" class="input min-h-11 w-full" :placeholder="t('groupRealtime.search')" /></label>
        </div>
        <div class="mt-3 overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead class="border-b border-gray-100 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400"><tr>
              <th scope="col" class="py-3 pr-4">{{ t('groupRealtime.group') }}</th>
              <th scope="col" class="whitespace-nowrap px-3 py-3" aria-sort="descending">{{ t('groupRealtime.history.averageRPM') }} ↓</th>
              <th scope="col" class="whitespace-nowrap px-3 py-3">{{ t('groupRealtime.history.rate') }}</th>
              <th scope="col" class="px-3 py-3">{{ t('groupRealtime.success') }}</th>
              <th scope="col" class="px-3 py-3">{{ t('groupRealtime.failed') }}</th>
              <th scope="col" class="px-3 py-3">{{ t('groupRealtime.cancelled') }}</th>
            </tr></thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in rows" :key="row.group_id">
                <td class="py-2 pr-4"><button class="min-h-11 cursor-pointer text-left font-medium text-primary-600 hover:underline focus-visible:outline focus-visible:outline-2 dark:text-primary-400" :aria-label="t('groupRealtime.history.viewGroup', { group: name(row) })" :aria-pressed="isSelected(row)" @click="toggleGroup(row.group_id)"><span class="mr-2 inline-block h-2 w-2 rounded-full" :style="{ backgroundColor: colorFor(row.group_id) }" aria-hidden="true" />{{ name(row) }}</button><p class="text-xs text-gray-500">{{ row.platform || '—' }} · #{{ row.group_id }}</p></td>
                <td class="px-3 py-3 font-semibold tabular-nums">{{ number(row.rpm) }}</td>
                <td class="px-3 py-3 tabular-nums">{{ rate(row.success_rate) }}<p v-if="row.success + row.failed > 0 && row.success + row.failed < 10" class="mt-1 whitespace-nowrap text-xs text-gray-500">{{ t('groupRealtime.sample') }} · n={{ row.success + row.failed }}</p></td>
                <td class="px-3 py-3 tabular-nums">{{ number(row.success, 0) }}</td>
                <td class="px-3 py-3 tabular-nums">{{ number(row.failed, 0) }}</td>
                <td class="px-3 py-3 tabular-nums text-gray-500">{{ number(row.cancelled, 0) }}</td>
              </tr>
              <tr v-if="!rows.length"><td colspan="6" class="py-8 text-center text-gray-500">{{ t('groupRealtime.empty') }}</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutationObserver } from '@vueuse/core'
import { CategoryScale, Chart as ChartJS, Legend, LineElement, LinearScale, PointElement, Tooltip, type ChartOptions } from 'chart.js'
import { Line } from 'vue-chartjs'
import { getGroupHistory, type GroupHistory, type GroupHistoryRow, type GroupHistoryWindow } from '@/api/admin/groupRealtime'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)
const props = withDefaults(defineProps<{ paused?: boolean; refreshToken?: number }>(), { paused: false, refreshToken: 0 })
const { t, locale } = useI18n()
const dark = ref(document.documentElement.classList.contains('dark'))
useMutationObserver(document.documentElement, () => { dark.value = document.documentElement.classList.contains('dark') }, { attributes: true, attributeFilter: ['class'] })
const ranges: GroupHistoryWindow[] = [15, 60, 360, 1440, 10080]
const windowMinutes = ref<GroupHistoryWindow>(60)
const metrics = ['rpm', 'success_rate'] as const
const metric = ref<typeof metrics[number]>('rpm')
const selectionOverrides = ref(new Map<number, boolean>())
const palette = ['#3b82f6', '#10b981', '#f59e0b', '#a855f7', '#ef4444', '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#6366f1']
const colors = new Map<number, string>()
function colorFor(id: number): string {
  if (!colors.has(id)) {
    const index = colors.size
    colors.set(id, palette[index] ?? `hsl(${Math.round(index * 137.508) % 360}, 65%, 50%)`)
  }
  return colors.get(id)!
}
function hasRPM(group: GroupHistoryRow) {
  return group.points.some(p => !p.partial && (p.rpm ?? 0) > 0)
}
function isSelected(group: GroupHistoryRow) {
  return selectionOverrides.value.get(group.group_id) ?? hasRPM(group)
}
function toggleGroup(id: number) {
  const group = data.value?.groups.find(g => g.group_id === id)
  if (group) selectionOverrides.value = new Map(selectionOverrides.value).set(id, !isSelected(group))
}
function setAllGroups(selected: boolean) {
  selectionOverrides.value = new Map((data.value?.groups ?? []).map(g => [g.group_id, selected]))
}
const data = ref<GroupHistory | null>(null)
const search = ref('')
const loading = ref(false)
const error = ref(false)
const receivedAt = ref(0)
const clock = ref(Date.now())
const stale = computed(() => !!data.value && (error.value || clock.value - receivedAt.value > 65000))
function name(row: Pick<GroupHistoryRow, 'group_id' | 'group_name'>) { return row.group_name || t(row.group_id ? 'groupRealtime.removedGroup' : 'groupRealtime.unknownGroup', { id: row.group_id }) }

const activeGroups = computed(() => (data.value?.groups ?? []).filter(hasRPM))
const rows = computed(() => [...(data.value?.groups ?? [])].filter(g => `${name(g)} ${g.group_id}`.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())).sort((a, b) => (b.rpm ?? -1) - (a.rpm ?? -1) || a.group_id - b.group_id))
const totals = computed(() => (data.value?.points ?? []).filter(p => !p.partial).reduce((sum, p) => ({ started: sum.started + p.started, success: sum.success + p.success, failed: sum.failed + p.failed }), { started: 0, success: 0, failed: 0 }))
const averageRPM = computed(() => data.value?.covered_seconds ? totals.value.started * 60 / data.value.covered_seconds : null)
const successRate = computed(() => totals.value.success + totals.value.failed ? 100 * totals.value.success / (totals.value.success + totals.value.failed) : null)
function number(value: number | null, digits = 1) { return value === null ? '—' : value.toLocaleString(locale.value, { maximumFractionDigits: digits }) }
function rate(value: number | null) { return value === null ? '—' : `${value.toLocaleString(locale.value, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%` }
function date(value: string) { return new Date(value).toLocaleString(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }) }
const visibleGroups = computed(() => (data.value?.groups ?? []).filter(isSelected))
const chartData = computed(() => ({
  labels: data.value?.points.map(p => date(p.bucket_start)) ?? [],
  datasets: visibleGroups.value.map(group => ({
    label: name(group),
    data: group.points.map(p => p.partial ? null : p[metric.value]),
    borderColor: colorFor(group.group_id), backgroundColor: colorFor(group.group_id),
    borderWidth: 2, pointRadius: 0, pointHitRadius: 12, spanGaps: false,
  })),
}))
const chartOptions = computed<ChartOptions<'line'>>(() => {
  const color = dark.value ? '#9ca3af' : '#6b7280'
  return {
    responsive: true, maintainAspectRatio: false, animation: false,
    interaction: { intersect: false, mode: 'index' },
    plugins: { legend: { display: false }, tooltip: { callbacks: { label: context => `${context.dataset.label}: ${metric.value === 'success_rate' ? rate(context.parsed.y) : number(context.parsed.y) + ' RPM'}` } } },
    scales: {
      x: { grid: { display: false }, ticks: { color, maxTicksLimit: 6, maxRotation: 0 } },
      y: { beginAtZero: true, ...(metric.value === 'success_rate' ? { max: 100 } : {}), title: { display: true, text: metric.value === 'rpm' ? 'RPM' : '%', color }, ticks: { color }, grid: { color: dark.value ? '#374151' : '#f3f4f6' } },
    },
  }
})
let controller: AbortController | null = null
let timer: ReturnType<typeof setInterval> | undefined
let generation = 0
let disposed = false
let attemptedAt = 0
async function refresh(reset = false) {
  if (disposed || (loading.value && !reset)) return
  controller?.abort()
  const request = ++generation
  controller = new AbortController()
  loading.value = true
  error.value = false
  attemptedAt = Date.now()
  if (reset) data.value = null
  try {
    const result = await getGroupHistory(windowMinutes.value, null, controller.signal)
    if (disposed || request !== generation) return
    data.value = result
    // Assign each group a stable color independent of RPM reordering.
    for (const group of [...result.groups].sort((a, b) => a.group_id - b.group_id)) colorFor(group.group_id)
    receivedAt.value = Date.now()
  } catch {
    if (!disposed && request === generation && !controller?.signal.aborted) error.value = true
  } finally {
    if (request === generation) loading.value = false
  }
}
watch(windowMinutes, () => { selectionOverrides.value = new Map(); void refresh(true) })
watch(() => props.refreshToken, () => void refresh())
watch(() => props.paused, paused => { if (!paused && !document.hidden) void refresh() })
function visibility() { if (!document.hidden && !props.paused) void refresh() }
onMounted(() => {
  void refresh()
  timer = setInterval(() => {
    clock.value = Date.now()
    if (!props.paused && !document.hidden && Date.now() - attemptedAt >= 30000) void refresh()
  }, 1000)
  document.addEventListener('visibilitychange', visibility)
})
onUnmounted(() => { disposed = true; controller?.abort(); clearInterval(timer); document.removeEventListener('visibilitychange', visibility) })
</script>
