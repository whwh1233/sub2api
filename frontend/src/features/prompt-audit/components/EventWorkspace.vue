<template>
  <section aria-labelledby="prompt-events-title" class="py-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <div class="flex flex-wrap items-center gap-2.5">
          <h2 id="prompt-events-title" class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.promptAudit.events.title') }}</h2>
          <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium tabular-nums text-gray-600 dark:bg-dark-800 dark:text-dark-300">
            {{ t('admin.promptAudit.events.totalCount', { count: total }) }}
          </span>
        </div>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('admin.promptAudit.events.description') }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="applyFilters">
          {{ loading ? t('admin.promptAudit.events.refreshing') : t('admin.promptAudit.events.refresh') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="selectedIds.length === 0" @click="$emit('batch-delete')">
          {{ t('admin.promptAudit.events.deleteSelected', { count: selectedIds.length }) }}
        </button>
        <button type="button" class="btn btn-danger btn-sm" data-test="filter-delete" @click="$emit('preview-delete')">
          {{ t('admin.promptAudit.events.deleteByFilter') }}
        </button>
      </div>
    </div>

    <form class="mt-5 space-y-3" @submit.prevent="applyFilters">
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div class="text-xs text-gray-600 dark:text-dark-200">
          <label for="prompt-event-group">{{ t('admin.promptAudit.events.group') }}</label>
          <Select id="prompt-event-group" class="mt-1" :model-value="localFilters.group_id" :options="groupOptions" :searchable="true" :clearable="true" :loading="groupsLoading" :aria-label="t('admin.promptAudit.events.group')" :search-placeholder="t('admin.promptAudit.events.searchGroup')" @update:model-value="selectGroup" />
          <p v-if="groupsError" role="alert" class="mt-1 text-amber-700 dark:text-amber-300">{{ groupsError }}</p>
        </div>
        <label class="text-xs text-gray-600 dark:text-dark-200">
          <span>{{ t('admin.promptAudit.events.model') }}</span>
          <input v-model="localFilters.model" class="input mt-1 w-full" list="prompt-event-models" :aria-label="t('admin.promptAudit.events.model')" :placeholder="t('admin.promptAudit.events.modelHint')" @change="filtersChanged" />
          <datalist id="prompt-event-models"><option v-for="model in modelSuggestions" :key="model" :value="model" /></datalist>
        </label>
        <label class="text-xs text-gray-600 dark:text-dark-200">
          <span>{{ t('admin.promptAudit.events.callerSearch') }}</span>
          <input v-model="localFilters.keyword" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.callerSearch')" :placeholder="t('admin.promptAudit.events.callerSearchHint')" @change="filtersChanged" />
        </label>
        <label class="text-xs text-gray-600 dark:text-dark-200">
          <span>{{ t('admin.promptAudit.events.upstreamStatus') }}</span>
          <select v-model="localFilters.upstream_status" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.upstreamStatus')" data-test="upstream-status-filter" @change="filtersChanged">
            <option value="">{{ t('common.all') }}</option>
            <option value="403">403</option>
          </select>
        </label>
      </div>
      <div class="flex flex-wrap items-end gap-3">
        <label class="text-xs text-gray-600 dark:text-dark-200">
          <span>{{ t('admin.promptAudit.events.startAt') }}</span>
          <input v-model="localFilters.start_at" type="datetime-local" step="0.001" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.startAt')" @change="filtersChanged" />
        </label>
        <label class="text-xs text-gray-600 dark:text-dark-200">
          <span>{{ t('admin.promptAudit.events.endAt') }}</span>
          <input v-model="localFilters.end_at" type="datetime-local" step="0.001" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.endAt')" @change="filtersChanged" />
        </label>
        <div class="flex flex-wrap gap-1.5">
          <button v-for="range in timeRanges" :key="range.minutes" type="button" class="btn btn-secondary btn-sm" :data-test="`recent-${range.minutes}`" @click="selectRecentRange(range.minutes)">{{ t(range.label) }}</button>
        </div>
      </div>
      <details class="rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-700" data-test="advanced-event-filters" :open="advancedFilterCount > 0">
        <summary class="cursor-pointer text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.promptAudit.events.moreConditions') }}<span v-if="advancedFilterCount"> · {{ advancedFilterCount }}</span></summary>
        <div class="mt-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <label class="text-xs text-gray-600 dark:text-dark-200">
        <span>{{ t('admin.promptAudit.events.decision') }}</span>
        <select v-model="localFilters.decision" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.decision')" @change="filtersChanged">
          <option value="">{{ t('common.all') }}</option>
          <option value="pass">{{ t('admin.promptAudit.decisions.pass') }}</option>
          <option value="flag">{{ t('admin.promptAudit.decisions.flag') }}</option>
          <option value="critical">{{ t('admin.promptAudit.decisions.critical') }}</option>
        </select>
      </label>
      <label class="text-xs text-gray-600 dark:text-dark-200">
        <span>{{ t('admin.promptAudit.events.risk') }}</span>
        <select v-model="localFilters.risk_level" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.risk')" @change="filtersChanged">
          <option value="">{{ t('common.all') }}</option>
          <option value="low">{{ t('admin.promptAudit.riskLevels.low') }}</option>
          <option value="medium">{{ t('admin.promptAudit.riskLevels.medium') }}</option>
          <option value="high">{{ t('admin.promptAudit.riskLevels.high') }}</option>
          <option value="critical">{{ t('admin.promptAudit.riskLevels.critical') }}</option>
        </select>
      </label>
      <FilterInput v-model="localFilters.endpoint" :label="t('admin.promptAudit.events.endpoint')" @change="filtersChanged" />
          <FilterInput v-model="localFilters.user_id" :label="t('admin.promptAudit.events.userId')" type="number" @change="filtersChanged" />
          <FilterInput v-model="localFilters.api_key_id" :label="t('admin.promptAudit.events.apiKeyId')" type="number" @change="filtersChanged" />
          <FilterInput v-model="localFilters.request_id" :label="t('admin.promptAudit.events.requestId')" @change="filtersChanged" />
          <FilterInput v-model="localFilters.prompt_hash" :label="t('admin.promptAudit.events.promptHash')" @change="filtersChanged" />
        </div>
      </details>
      <div class="flex items-center gap-2">
        <button type="submit" class="btn btn-primary btn-sm">{{ t('common.search') }}</button>
        <button type="button" class="btn btn-ghost btn-sm" @click="resetFilters">{{ t('common.reset') }}</button>
      </div>
    </form>
    <div v-if="error" role="alert" class="mt-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ error }}</div>
    <div class="mt-5 flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm shadow-gray-100/60 dark:border-dark-700/60 dark:bg-transparent dark:shadow-none lg:h-[70dvh] lg:min-h-[30rem] lg:max-h-[52rem]">
      <div ref="tableScrollRef" class="min-h-0 flex-1 overflow-auto" data-test="event-table-scroll">
        <table class="min-w-[1320px] w-full table-fixed text-left text-sm" data-test="event-table">
        <thead class="sticky top-0 z-10 border-b border-gray-200 bg-gray-50 text-xs text-gray-500 shadow-[0_1px_0_rgba(229,231,235,1)] dark:border-dark-700 dark:bg-dark-900 dark:text-dark-400 dark:shadow-[0_1px_0_rgba(55,65,81,1)]">
          <tr>
            <th class="w-10 px-3 py-3"><input type="checkbox" :checked="allSelected" :aria-label="t('admin.promptAudit.events.selectAll')" @change="toggleAll" /></th>
            <th class="w-36 px-4 py-3 font-medium">{{ t('admin.promptAudit.events.time') }}</th>
            <th class="w-64 px-4 py-3 font-medium">{{ t('admin.promptAudit.events.identity') }}</th>
            <th class="w-80 px-4 py-3 font-medium">{{ t('admin.promptAudit.events.latestInput') }}</th>
            <th class="px-4 py-3 font-medium">{{ t('admin.promptAudit.events.requestRoute') }}</th>
            <th class="w-40 px-4 py-3 font-medium">{{ t('admin.promptAudit.events.result') }}</th>
            <th class="w-40 px-4 py-3 text-right font-medium">{{ t('admin.promptAudit.common.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-transparent">
          <tr v-if="loading"><td colspan="7" class="px-4 py-12 text-center text-gray-500" aria-busy="true">{{ t('common.loading') }}</td></tr>
          <tr v-else-if="events.length === 0"><td colspan="7" class="px-4 py-12 text-center text-gray-500">{{ t('admin.promptAudit.events.empty') }}</td></tr>
          <tr
            v-for="event in events"
            v-else
            :key="event.id"
            :data-test="`event-${event.id}`"
            :aria-selected="selectedIds.includes(event.id)"
            class="group align-middle transition-colors duration-150 hover:bg-gray-50/80 aria-selected:bg-primary-50/50 dark:hover:bg-dark-800/70 dark:aria-selected:bg-primary-950/20"
          >
            <td class="px-3 py-4"><input type="checkbox" :checked="selectedIds.includes(event.id)" :aria-label="t('admin.promptAudit.events.selectEvent', { id: event.id })" @change="toggleOne(event.id)" /></td>
            <td class="px-4 py-4">
              <p class="whitespace-nowrap font-medium tabular-nums text-gray-800 dark:text-dark-100">{{ formatTime(event.created_at) }}</p>
              <p class="mt-1 whitespace-nowrap text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ formatDay(event.created_at) }} · #{{ event.id }}</p>
            </td>
            <td class="px-4 py-4">
              <p class="truncate font-medium text-gray-900 dark:text-white" :title="identityTitle(event)">{{ identityTitle(event) }}</p>
              <p v-if="identitySubtitle(event)" class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400" :title="identitySubtitle(event)">{{ identitySubtitle(event) }}</p>
              <div class="mt-2 flex flex-wrap items-center gap-1.5">
                <span class="max-w-40 truncate rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-800 dark:text-dark-300" :title="event.snapshot.api_key_name || ''">
                  {{ event.snapshot.api_key_name || t('admin.promptAudit.events.unknownApiKey') }}
                </span>
                <span v-if="event.snapshot.user_id" class="text-[11px] tabular-nums text-gray-400 dark:text-dark-500">UID {{ event.snapshot.user_id }}</span>
              </div>
            </td>
            <td class="px-4 py-4 align-top">
              <p class="line-clamp-3 whitespace-pre-wrap break-words font-medium leading-5 text-gray-800 dark:text-dark-100" :title="event.snapshot.request_excerpt || ''">
                {{ event.snapshot.request_excerpt || t('admin.promptAudit.events.noInputExcerpt') }}
              </p>
              <p class="mt-2 font-mono text-[11px] text-gray-400 dark:text-dark-500">{{ shortHash(event.snapshot.prompt_hash) }}</p>
            </td>
            <td class="px-4 py-4">
              <div class="flex min-w-0 items-center gap-2">
                <p class="truncate font-semibold text-gray-900 dark:text-white" :title="event.snapshot.model">{{ event.snapshot.model || '—' }}</p>
                <span class="flex-none rounded-md border border-gray-200 bg-gray-50 px-1.5 py-0.5 text-[11px] font-medium text-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-400">
                  {{ event.snapshot.provider || '—' }}
                </span>
              </div>
              <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400" :title="routeMeta(event)">{{ routeMeta(event) }}</p>
              <p class="mt-2 truncate text-xs font-medium text-gray-700 dark:text-dark-200" :title="event.snapshot.group_name || ''">{{ event.snapshot.group_name || '—' }}</p>
            </td>
            <td class="px-4 py-4">
              <span class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold" :class="resultClass(event)">{{ resultLabel(event) }}</span>
              <p v-if="!isRecordOnly(event)" class="mt-2 truncate text-xs text-gray-500 dark:text-dark-400" :title="formatCategories(event.categories)">{{ formatCategories(event.categories) }}</p>
              <p v-else class="mt-2 text-xs text-gray-400 dark:text-dark-500">{{ t('admin.promptAudit.events.noGuardCall') }}</p>
            </td>
            <td class="whitespace-nowrap px-4 py-4 text-right">
              <button type="button" class="inline-flex min-h-9 items-center rounded-lg border border-primary-200 bg-white px-3 py-1.5 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:border-primary-800 dark:bg-dark-900 dark:text-primary-300 dark:hover:bg-primary-950/40" @click="$emit('view', event.id)">{{ t('common.view') }}</button>
              <button type="button" class="ml-1 inline-flex min-h-9 items-center rounded-lg px-2.5 py-1.5 text-sm text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 dark:hover:bg-red-950/30 dark:hover:text-red-300" @click="$emit('delete', event.id)">{{ t('common.delete') }}</button>
            </td>
          </tr>
        </tbody>
        </table>
      </div>
      <Pagination class="flex-none" :total="total" :page="page" :page-size="pageSize" @update:page="handlePageChange" @update:page-size="handlePageSizeChange" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import type { PromptAuditEvent, PromptAuditGroup, PromptEventFilters } from '../types'
import { cloneData, emptyEventFilters, SCANNER_CATALOG } from '../viewModel'

const props = defineProps<{
  groups?: PromptAuditGroup[]; groupsLoading?: boolean; groupsError?: string
  events: PromptAuditEvent[]; total: number; page: number; pageSize: number
  filters: PromptEventFilters; selectedIds: number[]; loading: boolean; error: string
}>()
const emit = defineEmits<{
  (event: 'filters-change', value: PromptEventFilters): void
  (event: 'search', value: PromptEventFilters): void
  (event: 'selection', value: number[]): void
  (event: 'page', value: number): void
  (event: 'page-size', value: number): void
  (event: 'view', id: number): void
  (event: 'delete', id: number): void
  (event: 'batch-delete'): void
  (event: 'preview-delete'): void
}>()
const { t, locale } = useI18n()
const localFilters = reactive<PromptEventFilters>(cloneData(props.filters))
const tableScrollRef = ref<HTMLElement | null>(null)
watch(() => props.filters, (value) => Object.assign(localFilters, cloneData(value)), { deep: true })
const allSelected = computed(() => props.events.length > 0 && props.events.every((event) => props.selectedIds.includes(event.id)))
const groupOptions = computed(() => {
  const names = new Map<string, string>()
  for (const group of props.groups || []) names.set(String(group.id), group.name)
  for (const event of props.events) {
    const id = event.snapshot.group_id
    if (id && !names.has(String(id))) names.set(String(id), event.snapshot.group_name || `#${id}`)
  }
  if (localFilters.group_id && !names.has(localFilters.group_id)) names.set(localFilters.group_id, `#${localFilters.group_id}`)
  return [{ value: '', label: t('common.all') }, ...Array.from(names, ([value, name]) => ({ value, label: `${name} · #${value}` }))]
})
const modelSuggestions = computed(() => [...new Set(props.events.map(event => event.snapshot.model).filter(Boolean))].sort())
const advancedFilterCount = computed(() => [localFilters.decision, localFilters.risk_level, localFilters.endpoint, localFilters.user_id, localFilters.api_key_id, localFilters.request_id, localFilters.prompt_hash].filter(Boolean).length)
const timeRanges = [
  { minutes: 10, label: 'admin.promptAudit.events.recent10m' },
  { minutes: 60, label: 'admin.promptAudit.events.recent1h' },
  { minutes: 1440, label: 'admin.promptAudit.events.recent24h' },
  { minutes: 0, label: 'common.all' },
]

function selectGroup(value: string | number | boolean | null) {
  localFilters.group_id = value == null ? '' : String(value)
  filtersChanged()
}
function selectRecentRange(minutes: number) {
  const now = Date.now()
  const localTime = (timestamp: number) => {
    const date = new Date(timestamp)
    return new Date(timestamp - date.getTimezoneOffset() * 60000).toISOString().slice(0, 23)
  }
  localFilters.start_at = minutes ? localTime(now - minutes * 60000) : ''
  localFilters.end_at = minutes ? localTime(now) : ''
  applyFilters()
}

const FilterInput = defineComponent({
  props: { modelValue: { type: String, required: true }, label: { type: String, required: true }, type: { type: String, default: 'text' } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit: componentEmit }) {
    return () => h('label', { class: 'text-xs text-gray-600 dark:text-dark-200' }, [
      h('span', componentProps.label),
      h('input', {
        value: componentProps.modelValue, type: componentProps.type, class: 'input mt-1 w-full', 'aria-label': componentProps.label,
        onInput: (event: Event) => componentEmit('update:modelValue', (event.target as HTMLInputElement).value),
        onChange: () => componentEmit('change'),
      }),
    ])
  },
})

function filtersChanged() {
  emit('filters-change', cloneData(localFilters))
}
function applyFilters() {
  resetTableScroll()
  const value = cloneData(localFilters)
  emit('filters-change', value)
  emit('search', value)
}
function resetFilters() {
  Object.assign(localFilters, emptyEventFilters())
  applyFilters()
}
function toggleOne(id: number) {
  const selected = new Set(props.selectedIds)
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
  emit('selection', [...selected])
}
function toggleAll() {
  emit('selection', allSelected.value ? [] : props.events.map((event) => event.id))
}
function resetTableScroll() {
  tableScrollRef.value?.scrollTo?.({ top: 0, behavior: 'auto' })
}
function handlePageChange(nextPage: number) {
  resetTableScroll()
  emit('page', nextPage)
}
function handlePageSizeChange(nextPageSize: number) {
  resetTableScroll()
  emit('page-size', nextPageSize)
}
function formatTime(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(value))
}
function formatDay(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(value))
}
function identityTitle(event: PromptAuditEvent): string {
  return event.snapshot.username || event.snapshot.user_email || (event.snapshot.user_id ? `User #${event.snapshot.user_id}` : '—')
}
function identitySubtitle(event: PromptAuditEvent): string {
  if (!event.snapshot.user_email || event.snapshot.user_email === event.snapshot.username) return ''
  return event.snapshot.user_email
}
function routeMeta(event: PromptAuditEvent): string {
  return [event.snapshot.endpoint, event.snapshot.protocol, event.snapshot.stage || 'http'].filter(Boolean).join(' · ')
}
function shortHash(value: string): string {
  if (!value) return '—'
  return `${value.slice(0, 8)}…`
}
function isRecordOnly(event: PromptAuditEvent): boolean {
  return event.scanner_backend === 'record-only'
}
function resultClass(event: PromptAuditEvent): string {
  if (isRecordOnly(event)) return 'bg-sky-100 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300'
  if (event.decision === 'critical') return 'bg-red-100 text-red-700 dark:bg-red-950/50 dark:text-red-300'
  if (event.decision === 'flag') return 'bg-amber-100 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300'
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300'
}
function resultLabel(event: PromptAuditEvent): string {
  return isRecordOnly(event) ? t('admin.promptAudit.events.recorded') : formatDecisionRisk(event.decision, event.risk_level)
}
const DECISIONS = new Set(['pass', 'flag', 'critical'])
const RISK_LEVELS = new Set(['low', 'medium', 'high', 'critical'])

function translateDecision(decision: string): string {
  return DECISIONS.has(decision) ? t(`admin.promptAudit.decisions.${decision}`) : decision
}
function translateRiskLevel(riskLevel: string): string {
  return RISK_LEVELS.has(riskLevel) ? t(`admin.promptAudit.riskLevels.${riskLevel}`) : riskLevel
}
function translateCategory(category: string): string {
  return SCANNER_CATALOG.some((scanner) => scanner.id === category)
    ? t(`admin.promptAudit.scanners.${category}`)
    : category
}
function formatDecisionRisk(decision: string, riskLevel: string): string {
  return `${translateDecision(decision)} · ${translateRiskLevel(riskLevel)}`
}
function formatCategories(categories: string[]): string {
  if (!categories.length) return '—'
  return categories.map(translateCategory).join(', ')
}
</script>
