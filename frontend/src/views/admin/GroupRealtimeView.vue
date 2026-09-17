<template>
  <AppLayout>
    <div class="space-y-6 pb-8">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="page-title">{{ t('groupRealtime.title') }}</h1>
          <p class="page-description mt-1">{{ t('groupRealtime.description') }}</p>
        </div>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary min-h-11" @click="togglePause">{{ t(paused ? 'groupRealtime.resume' : 'groupRealtime.pause') }}</button>
          <button type="button" class="btn btn-primary min-h-11" :disabled="loading" @click="refresh">{{ t('groupRealtime.refresh') }}</button>
        </div>
      </header>

      <div class="rounded-xl border border-gray-200 bg-white p-4 text-sm dark:border-dark-700 dark:bg-dark-800">
        <p class="text-gray-600 dark:text-gray-300">{{ t('groupRealtime.scope') }}</p>
        <div class="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-gray-500 dark:text-gray-400">
          <span v-if="snapshot">{{ t('groupRealtime.window') }}: {{ formatTime(snapshot.start_time) }} – {{ formatTime(snapshot.end_time) }}</span>
          <span>{{ t('groupRealtime.updated') }}: {{ snapshot ? formatTime(snapshot.end_time) : '—' }}</span>
          <span :class="stale ? 'text-amber-700 dark:text-amber-400' : ''">{{ t(stale ? 'groupRealtime.stale' : paused ? 'groupRealtime.paused' : 'groupRealtime.live') }}</span>
        </div>
      </div>
      <p v-if="error" role="alert" class="rounded-xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ t('groupRealtime.loadError') }}</p>
      <p v-if="snapshot?.partial" role="status" class="rounded-xl bg-amber-50 p-4 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">{{ t('groupRealtime.partial') }}</p>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div v-for="card in cards" :key="card.label" class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t(card.label) }}</p>
          <p class="mt-2 text-3xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ snapshot ? card.value : '—' }}</p>
          <p class="mt-2 text-xs text-gray-400">{{ t('groupRealtime.filtered') }}</p>
        </div>
      </div>

      <div class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-end gap-3 p-4">
          <label class="flex w-full min-w-0 flex-1 basis-full flex-col gap-1 text-sm text-gray-600 dark:text-gray-300 sm:basis-0">
            {{ t('groupRealtime.search') }}
            <input v-model="search" type="search" class="input min-h-11 w-full" />
          </label>
          <label class="flex flex-col gap-1 text-sm text-gray-600 dark:text-gray-300">
            {{ t('groupRealtime.platform') }}
            <select v-model="platform" class="input min-h-11">
              <option value="">{{ t('groupRealtime.allPlatforms') }}</option>
              <option v-for="item in platforms" :key="item" :value="item">{{ item }}</option>
            </select>
          </label>
          <label class="flex flex-col gap-1 text-sm text-gray-600 dark:text-gray-300 md:hidden">
            {{ t('groupRealtime.sort') }}
            <select :value="sortKey" class="input min-h-11" @change="sortBy(($event.target as HTMLSelectElement).value as SortKey)">
              <option v-for="column in columns" :key="column.key" :value="column.key">{{ t(column.label) }}</option>
            </select>
          </label>
          <button type="button" class="btn btn-secondary min-h-11 md:hidden" @click="sortBy(sortKey)">{{ t(sortDesc ? 'groupRealtime.descending' : 'groupRealtime.ascending') }}</button>
        </div>
        <p class="px-4 pb-3 text-xs text-gray-500 dark:text-gray-400">{{ t('groupRealtime.sortHint') }}</p>
        <div class="hidden overflow-x-auto md:block">
          <table class="w-full text-left text-sm">
            <thead class="border-y border-gray-100 bg-gray-50 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-400">
              <tr>
                <th v-for="column in columns" :key="column.key" :aria-sort="sortKey === column.key ? (sortDesc ? 'descending' : 'ascending') : 'none'" scope="col" class="px-4">
                  <button class="flex min-h-11 items-center gap-1 whitespace-nowrap font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500" @click="sortBy(column.key)">
                    {{ t(column.label) }} <span v-if="sortKey === column.key" aria-hidden="true">{{ sortDesc ? '↓' : '↑' }}</span>
                  </button>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-if="!snapshot || !rows.length"><td colspan="6" class="px-4 py-12 text-center text-gray-500">{{ t(loading && !snapshot ? 'groupRealtime.loading' : 'groupRealtime.empty') }}</td></tr>
              <tr v-for="row in rows" :key="row.group_id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-4 py-4">
                  <div class="font-medium text-gray-900 dark:text-gray-100">{{ groupName(row) }}</div>
                  <div class="mt-1 text-xs text-gray-500">{{ row.platform || '—' }} · #{{ row.group_id }} <span v-if="row.status === 'inactive'">· {{ t('groupRealtime.inactive') }}</span></div>
                </td>
                <td class="px-4 py-4 text-lg font-semibold tabular-nums">{{ number(row.rpm) }}</td>
                <td class="px-4 py-4 tabular-nums">
                  <span :class="row.failed > 0 ? 'text-amber-700 dark:text-amber-400' : 'text-gray-900 dark:text-gray-100'">{{ rate(row.success_rate) }}</span>
                  <div v-if="row.success + row.failed === 0" class="mt-1 whitespace-nowrap text-xs text-gray-400">{{ t('groupRealtime.noRequests') }}</div>
                  <div v-else-if="row.success + row.failed < 10" class="mt-1 whitespace-nowrap text-xs text-gray-400">{{ t('groupRealtime.sample') }} · n={{ row.success + row.failed }}</div>
                </td>
                <td class="px-4 py-4 tabular-nums">{{ number(row.success) }}</td>
                <td class="px-4 py-4 tabular-nums">
                  <button v-if="row.failed" type="button" class="min-h-11 min-w-11 rounded-lg text-red-600 underline decoration-dotted underline-offset-4 dark:text-red-400" :aria-label="`${groupName(row)} · ${t('groupRealtime.details')}`" @click="openDetails(row)">{{ number(row.failed) }}</button>
                  <span v-else>0</span>
                </td>
                <td class="px-4 py-4 tabular-nums text-gray-500">{{ number(row.cancelled) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-dark-700 md:hidden">
          <p v-if="!snapshot || !rows.length" class="p-6 text-center text-sm text-gray-500">{{ t(loading && !snapshot ? 'groupRealtime.loading' : 'groupRealtime.empty') }}</p>
          <article v-for="row in rows" :key="row.group_id" class="space-y-3 p-4">
            <div>
              <h2 class="break-words text-sm font-medium text-gray-900 dark:text-gray-100">{{ groupName(row) }}</h2>
              <p class="mt-1 text-xs text-gray-500">{{ row.platform || '—' }} · #{{ row.group_id }} <span v-if="row.status === 'inactive'">· {{ t('groupRealtime.inactive') }}</span></p>
            </div>
            <dl class="grid grid-cols-3 gap-2 text-sm">
              <div><dt class="text-xs text-gray-500">{{ t('groupRealtime.rpm') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums">{{ number(row.rpm) }}</dd></div>
              <div><dt class="text-xs text-gray-500">{{ t('groupRealtime.rate') }}</dt><dd class="mt-1 text-lg font-semibold tabular-nums">{{ rate(row.success_rate) }}</dd></div>
              <div><dt class="text-xs text-gray-500">{{ t('groupRealtime.failed') }}</dt><dd class="tabular-nums"><button v-if="row.failed" class="min-h-11 min-w-11 text-left text-lg font-semibold text-red-600 underline dark:text-red-400" :aria-label="`${groupName(row)} · ${t('groupRealtime.details')}`" @click="openDetails(row)">{{ number(row.failed) }}</button><span v-else class="mt-1 block text-lg font-semibold">0</span></dd></div>
            </dl>
            <p class="text-xs text-gray-500">{{ t('groupRealtime.success') }} {{ number(row.success) }} · {{ t('groupRealtime.cancelled') }} {{ number(row.cancelled) }}<span v-if="row.success + row.failed > 0 && row.success + row.failed < 10"> · {{ t('groupRealtime.sample') }}</span></p>
          </article>
        </div>
      </div>
      <BaseDialog :show="details !== null" :title="t('groupRealtime.details')" @close="details = null">
        <template v-if="details">
          <p class="font-medium">{{ groupName(details.row) }}</p>
          <p class="mt-2 text-xs text-gray-500">{{ formatTime(details.start) }} – {{ formatTime(details.end) }}</p>
          <p class="my-4 text-sm text-gray-500">{{ t('groupRealtime.detailsHint') }}</p>
          <table class="w-full text-left text-sm">
            <thead><tr><th class="py-2">{{ t('groupRealtime.reason') }}</th><th class="py-2 text-right">{{ t('groupRealtime.count') }}</th></tr></thead>
            <tbody><tr v-for="[reason, count] in Object.entries(details.row.failures).sort((a, b) => b[1] - a[1])" :key="reason"><td class="py-2">{{ te(`groupRealtime.reasons.${reason}`) ? t(`groupRealtime.reasons.${reason}`) : t('groupRealtime.reasons.other') }}</td><td class="py-2 text-right tabular-nums">{{ number(count) }}</td></tr></tbody>
          </table>
        </template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getGroupRealtime, type GroupRealtimeRow, type GroupRealtimeSnapshot } from '@/api/admin/groupRealtime'

const { t, te, locale } = useI18n()
const snapshot = ref<GroupRealtimeSnapshot | null>(null)
const loading = ref(false)
const error = ref(false)
const paused = ref(false)
const search = ref('')
const platform = ref('')
const clock = ref(Date.now())
const lastReceived = ref(0)
const details = ref<{ row: GroupRealtimeRow; start: string; end: string } | null>(null)
type SortKey = 'group_name' | 'rpm' | 'success_rate' | 'success' | 'failed' | 'cancelled'
const sortKey = ref<SortKey>('rpm')
const sortDesc = ref(true)
const order = ref<number[]>([])
const columns: { key: SortKey; label: string }[] = [
  { key: 'group_name', label: 'groupRealtime.group' }, { key: 'rpm', label: 'groupRealtime.rpm' },
  { key: 'success_rate', label: 'groupRealtime.rate' }, { key: 'success', label: 'groupRealtime.success' },
  { key: 'failed', label: 'groupRealtime.failed' }, { key: 'cancelled', label: 'groupRealtime.cancelled' },
]
const stale = computed(() => !!snapshot.value && (error.value || clock.value - lastReceived.value > 15000))
const platforms = computed(() => [...new Set(snapshot.value?.groups.map(row => row.platform).filter(Boolean) ?? [])].sort())
const filtered = computed(() => (snapshot.value?.groups ?? []).filter(row =>
  (!platform.value || row.platform === platform.value) &&
  `${groupName(row)} ${row.group_id}`.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase()),
))
const rows = computed(() => {
  const ranks = new Map(order.value.map((id, index) => [id, index]))
  return [...filtered.value].sort((a, b) => (ranks.get(a.group_id) ?? Infinity) - (ranks.get(b.group_id) ?? Infinity))
})
const cards = computed(() => {
  const totals = filtered.value.reduce((sum, row) => ({ rpm: sum.rpm + row.rpm, success: sum.success + row.success, failed: sum.failed + row.failed, active: sum.active + Number(row.rpm > 0) }), { rpm: 0, success: 0, failed: 0, active: 0 })
  return [
    { label: 'groupRealtime.totalRPM', value: number(totals.rpm) },
    { label: 'groupRealtime.totalRate', value: rate(totals.success + totals.failed ? 100 * totals.success / (totals.success + totals.failed) : null) },
    { label: 'groupRealtime.activeGroups', value: `${number(totals.active)} / ${number(filtered.value.length)}` },
  ]
})
function number(value: number) { return value.toLocaleString(locale.value) }
function rate(value: number | null) { return value === null ? '—' : `${value.toLocaleString(locale.value, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%` }
function formatTime(value: string) { return new Date(value).toLocaleTimeString(locale.value, { hour12: false }) }
function groupName(row: GroupRealtimeRow) { return row.group_name || t(row.group_id ? 'groupRealtime.removedGroup' : 'groupRealtime.unknownGroup', { id: row.group_id }) }
function reorder() {
  order.value = [...(snapshot.value?.groups ?? [])].sort((a, b) => {
    const av = a[sortKey.value], bv = b[sortKey.value]
    if (av === null) return bv === null ? a.group_id - b.group_id : 1
    if (bv === null) return -1
    const diff = typeof av === 'string' && typeof bv === 'string' ? av.localeCompare(bv) : Number(av) - Number(bv)
    return (sortDesc.value ? -diff : diff) || a.group_id - b.group_id
  }).map(row => row.group_id)
}
function sortBy(key: SortKey) {
  sortDesc.value = key === sortKey.value ? !sortDesc.value : key !== 'group_name'
  sortKey.value = key
  reorder()
}
function openDetails(row: GroupRealtimeRow) {
  if (snapshot.value) details.value = { row: { ...row, failures: { ...row.failures } }, start: snapshot.value.start_time, end: snapshot.value.end_time }
}
let controller: AbortController | null = null
let timer: ReturnType<typeof setInterval> | undefined
let disposed = false
let lastAttempt = 0
async function refresh() {
  if (loading.value || disposed) return
  loading.value = true
  lastAttempt = Date.now()
  controller = new AbortController()
  try {
    const data = await getGroupRealtime(controller.signal)
    if (disposed) return
    snapshot.value = data
    if (!order.value.length) reorder()
    else {
      const present = new Set(data.groups.map(row => row.group_id))
      const existing = new Set(order.value)
      order.value = [...order.value.filter(id => present.has(id)), ...data.groups.filter(row => !existing.has(row.group_id)).map(row => row.group_id)]
    }
    lastReceived.value = Date.now()
    error.value = false
  } catch {
    if (!controller?.signal.aborted && !disposed) error.value = true
  } finally {
    loading.value = false
  }
}
function togglePause() { paused.value = !paused.value; if (!paused.value) void refresh() }
function onVisibility() { if (!document.hidden && !paused.value) void refresh() }
onMounted(() => {
  void refresh()
  timer = setInterval(() => {
    clock.value = Date.now()
    if (!paused.value && !document.hidden && Date.now() - lastAttempt >= 5000) void refresh()
  }, 1000)
  document.addEventListener('visibilitychange', onVisibility)
})
onUnmounted(() => {
  disposed = true
  clearInterval(timer)
  controller?.abort()
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>
