<template>
  <AppLayout>
    <div class="space-y-5 pb-10">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">原文请求日志</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            展示测试环境捕获的请求入参、响应出参、响应数据和完整 raw JSONL 原文。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="clearFilters">清空筛选</button>
          <button type="button" class="btn btn-primary btn-sm" :disabled="loading" @click="loadLogs">
            {{ loading ? '加载中' : '刷新' }}
          </button>
        </div>
      </div>

      <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/50 dark:bg-amber-900/20 dark:text-amber-200">
        当前页面按要求展示未脱敏原文，包含 Authorization、Cookie、请求体、响应体和 base64 原始字节字段，请仅在测试环境使用。
      </div>

      <form class="card p-4" @submit.prevent="loadLogs">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <div class="xl:col-span-2">
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">全文搜索</label>
            <input v-model.trim="filters.q" class="input w-full" placeholder="header/body/response 任意原文" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Request ID</label>
            <input v-model.trim="filters.request_id" class="input w-full font-mono text-xs" placeholder="req..." />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Path</label>
            <input v-model.trim="filters.path" class="input w-full font-mono text-xs" placeholder="/v1/chat" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Method</label>
            <select v-model="filters.method" class="input w-full">
              <option value="">全部</option>
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="PATCH">PATCH</option>
              <option value="DELETE">DELETE</option>
              <option value="OPTIONS">OPTIONS</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Status</label>
            <input v-model.trim="filters.status_code" type="number" min="100" max="599" class="input w-full" placeholder="200" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Limit</label>
            <input v-model.number="filters.limit" type="number" min="1" max="200" class="input w-full" />
          </div>
        </div>
        <div class="mt-3 flex justify-end">
          <button type="submit" class="btn btn-secondary btn-sm" :disabled="loading">
            {{ loading ? '查询中' : '查询' }}
          </button>
        </div>
      </form>

      <div
        v-if="errorMessage"
        class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300"
      >
        {{ errorMessage }}
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(360px,440px)_minmax(0,1fr)]">
        <section class="card min-w-0 overflow-hidden">
          <div class="border-b border-gray-100 p-4 dark:border-dark-800">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">请求列表</h2>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ requestGroups.length }} 个请求 / {{ total }} 条记录</span>
            </div>
            <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="logPath">
              {{ logPath || '尚未返回日志文件路径' }}
            </p>
          </div>

          <div v-if="loading && !requestGroups.length" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            正在读取原文日志...
          </div>
          <div v-else-if="!requestGroups.length" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            暂无匹配记录。发起一次接口请求后再刷新。
          </div>
          <div v-else class="max-h-[720px] divide-y divide-gray-100 overflow-y-auto dark:divide-dark-800">
            <button
              v-for="group in requestGroups"
              :key="group.id"
              type="button"
              class="block w-full px-4 py-3 text-left transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 dark:hover:bg-dark-800"
              :class="selectedRequestID === group.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
              @click="selectRequest(group.id)"
            >
              <div class="flex flex-wrap items-center gap-2">
                <span class="rounded px-2 py-0.5 text-xs font-semibold" :class="statusClass(group.client?.status_code || group.latest.status_code)">
                  {{ group.client?.status_code || group.latest.status_code || '-' }}
                </span>
                <span class="font-mono text-xs font-semibold text-gray-700 dark:text-gray-200">{{ group.client?.method || group.latest.method || '-' }}</span>
                <span class="min-w-0 flex-1 truncate font-mono text-xs text-gray-900 dark:text-white" :title="group.client?.request_uri || group.latest.url || group.latest.path">
                  {{ group.client?.request_uri || group.latest.path || '-' }}
                </span>
              </div>
              <div class="mt-2 grid grid-cols-2 gap-2 text-xs text-gray-500 dark:text-gray-400">
                <div class="truncate">
                  <span class="text-gray-400">时间</span>
                  {{ formatDate(group.latest.completed_at) }}
                </div>
                <div class="truncate">
                  <span class="text-gray-400">用户</span>
                  <span class="font-mono">{{ group.client?.user_id || group.latest.user_id || '-' }}</span>
                </div>
                <div class="truncate">
                  <span class="text-gray-400">IP</span>
                  <span class="font-mono">{{ group.client?.client_ip || '-' }}</span>
                </div>
                <div class="truncate">
                  <span class="text-gray-400">上游</span>
                  {{ group.upstream.length }} 次
                </div>
              </div>
              <div class="mt-2 truncate font-mono text-[11px] text-gray-400" :title="group.id">{{ group.id }}</div>
            </button>
          </div>
        </section>

        <section class="card min-w-0 overflow-hidden">
          <div class="border-b border-gray-100 p-4 dark:border-dark-800">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">Claude 请求链路原文</h2>
                <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400" :title="selectedGroup?.id">{{ selectedGroup?.id || '选择左侧请求' }}</p>
              </div>
              <div v-if="selectedGroup" class="flex flex-wrap gap-2">
                <button type="button" class="btn btn-secondary btn-sm" @click="copySelectedRaw">复制完整链路</button>
              </div>
            </div>
          </div>

          <div v-if="detailLoading" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            正在读取所选请求的完整原文...
          </div>

          <div v-else-if="!selectedGroup" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            选择左侧请求查看完整原文。
          </div>

          <div v-else class="space-y-5 p-4">
            <div class="grid grid-cols-2 gap-3 text-xs lg:grid-cols-5">
              <div v-for="summary in selectedSummary" :key="summary.label" class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-gray-400">{{ summary.label }}</div>
                <div class="mt-1 break-all font-mono text-gray-900 dark:text-white">{{ summary.value }}</div>
              </div>
            </div>

            <section v-if="selectedGroup.client" class="space-y-2">
              <div class="flex items-center justify-between gap-3">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">1. 客户端 → sub2api 请求</h3>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatBytes(selectedGroup.client.request_body_bytes) }}</span>
              </div>
              <pre class="max-h-[420px] overflow-auto whitespace-pre-wrap break-all rounded-md bg-gray-950 p-4 font-mono text-xs leading-5 text-gray-100">{{ stageText(selectedGroup.client, 'request') }}</pre>
            </section>

            <section class="space-y-3">
              <div class="flex items-center justify-between gap-3">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">2-3. sub2api ↔ Claude 上游</h3>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ selectedGroup.upstream.length }} 次实际请求</span>
              </div>
              <div v-if="!selectedGroup.upstream.length" class="rounded-md border border-dashed border-gray-300 p-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">尚无上游记录。</div>
              <details v-for="attempt in selectedGroup.upstream" :key="attempt.line" class="rounded-md border border-gray-200 dark:border-dark-700" open>
                <summary class="cursor-pointer px-4 py-3 text-sm font-medium text-gray-800 dark:text-gray-200">
                  Attempt {{ attempt.attempt || 1 }} · {{ attempt.operation || 'messages' }} · {{ attempt.status_code || 'ERR' }} · {{ formatDuration(attempt.latency_ms) }}
                </summary>
                <div class="grid grid-cols-1 gap-px border-t border-gray-200 bg-gray-200 dark:border-dark-700 dark:bg-dark-700 2xl:grid-cols-2">
                  <div class="min-w-0 bg-white p-3 dark:bg-dark-900">
                    <div class="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-300">sub2api → Claude 请求</div>
                    <pre class="max-h-[460px] overflow-auto whitespace-pre-wrap break-all rounded-md bg-gray-950 p-3 font-mono text-xs leading-5 text-gray-100">{{ stageText(attempt, 'request') }}</pre>
                  </div>
                  <div class="min-w-0 bg-white p-3 dark:bg-dark-900">
                    <div class="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-300">Claude → sub2api 响应</div>
                    <pre class="max-h-[460px] overflow-auto whitespace-pre-wrap break-all rounded-md bg-gray-950 p-3 font-mono text-xs leading-5 text-gray-100">{{ stageText(attempt, 'response') }}</pre>
                  </div>
                </div>
              </details>
            </section>

            <section v-if="selectedGroup.client" class="space-y-2">
              <div class="flex items-center justify-between gap-3">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">4. sub2api → 客户端响应</h3>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatBytes(selectedGroup.client.response_body_bytes) }}</span>
              </div>
              <pre class="max-h-[420px] overflow-auto whitespace-pre-wrap break-all rounded-md bg-gray-950 p-4 font-mono text-xs leading-5 text-gray-100">{{ stageText(selectedGroup.client, 'response') }}</pre>
            </section>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { opsAPI, type RawExchangeLogItem, type RawExchangeLogQuery } from '@/api/admin/ops'
import { useAppStore } from '@/stores'

const appStore = useAppStore()

const filters = reactive({
  limit: 100,
  q: '',
  request_id: '',
  path: '',
  method: '',
  status_code: '',
})

const logs = ref<RawExchangeLogItem[]>([])
const selectedRequestID = ref('')
const total = ref(0)
const logPath = ref('')
const loading = ref(false)
const detailLoading = ref(false)
const errorMessage = ref('')

interface RequestGroup {
  id: string
  latest: RawExchangeLogItem
  client: RawExchangeLogItem | null
  upstream: RawExchangeLogItem[]
  records: RawExchangeLogItem[]
}

const requestGroups = computed<RequestGroup[]>(() => {
  const grouped = new Map<string, RequestGroup>()
  for (const item of logs.value) {
    const id = item.request_id || item.client_request_id || `line-${item.line}`
    let group = grouped.get(id)
    if (!group) {
      group = { id, latest: item, client: null, upstream: [], records: [] }
      grouped.set(id, group)
    }
    group.records.push(item)
    if (item.stage === 'client_exchange' || (!item.stage && !group.client)) group.client = item
    if (item.stage === 'upstream_exchange') group.upstream.push(item)
  }
  for (const group of grouped.values()) {
    group.upstream.sort((a, b) => (a.attempt || 1) - (b.attempt || 1) || a.line - b.line)
  }
  return [...grouped.values()]
})

const selectedGroup = computed(() => requestGroups.value.find((group) => group.id === selectedRequestID.value) ?? null)
const selectedRawJSON = computed(() => JSON.stringify(selectedGroup.value?.records.map((item) => item.raw || {}) ?? [], null, 2))
const selectedSummary = computed(() => {
  const group = selectedGroup.value
  if (!group) return []
  const anchor = group.client || group.latest
  return [
    { label: '用户 ID', value: String(anchor.user_id || '-') },
    { label: '客户端 IP', value: anchor.client_ip || '-' },
    { label: '模型', value: anchor.model || '-' },
    { label: '账号 ID', value: String(group.upstream[0]?.account_id || anchor.account_id || '-') },
    { label: '总耗时', value: formatDuration(group.client?.latency_ms ?? anchor.latency_ms) },
  ]
})

function buildQuery(): RawExchangeLogQuery {
  const query: RawExchangeLogQuery = {
    limit: clampLimit(Number(filters.limit) || 100),
  }

  if (filters.q) query.q = filters.q
  if (filters.request_id) query.request_id = filters.request_id
  if (filters.path) query.path = filters.path
  if (filters.method) query.method = filters.method
  if (filters.status_code) query.status_code = Number(filters.status_code)

  return query
}

function clampLimit(limit: number): number {
  if (!Number.isFinite(limit)) return 100
  return Math.min(200, Math.max(1, Math.trunc(limit)))
}

async function loadLogs(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await opsAPI.listRawExchangeLogs(buildQuery())
    logs.value = response.items ?? []
    total.value = response.total ?? logs.value.length
    logPath.value = response.path || ''

    if (!requestGroups.value.some((group) => group.id === selectedRequestID.value)) {
      const firstID = requestGroups.value[0]?.id ?? ''
      selectedRequestID.value = firstID
      if (firstID) await loadRequestDetails(firstID)
    }
  } catch (error) {
    const message = (error as { message?: string })?.message || '读取原文日志失败'
    errorMessage.value = message
    appStore.showError(message)
  } finally {
    loading.value = false
  }
}

function clearFilters(): void {
  filters.limit = 100
  filters.q = ''
  filters.request_id = ''
  filters.path = ''
  filters.method = ''
  filters.status_code = ''
  void loadLogs()
}

function selectRequest(requestID: string): void {
  selectedRequestID.value = requestID
  void loadRequestDetails(requestID)
}

async function loadRequestDetails(requestID: string): Promise<void> {
  const group = requestGroups.value.find((item) => item.id === requestID)
  if (!group) return
  const missing = group.records.filter((item) => !item.raw)
  if (!missing.length) return
  detailLoading.value = true
  try {
    const details = await Promise.all(missing.map((item) => opsAPI.getRawExchangeLog(item.offset)))
    for (const detail of details) {
      const summary = logs.value.find((item) => item.offset === detail.offset)
      if (summary) Object.assign(summary, detail)
    }
  } catch (error) {
    const message = (error as { message?: string })?.message || '读取完整原文失败'
    appStore.showError(message)
  } finally {
    detailLoading.value = false
  }
}

function statusClass(statusCode: number): string {
  if (statusCode >= 500) return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-200'
  if (statusCode >= 400) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200'
  if (statusCode >= 300) return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-200'
  if (statusCode >= 200) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

function formatDate(value: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatDuration(value: number): string {
  if (!Number.isFinite(value)) return '-'
  return `${Math.round(value)} ms`
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

function bodyPreview(log: RawExchangeLogItem | null, bodyKey: string, base64Key: string): string {
  if (!log?.raw) return ''
  const rawBody = log.raw[bodyKey]
  if (typeof rawBody === 'string' && rawBody.length > 0) return rawBody

  const decoded = decodeBase64Text(log.raw[base64Key])
  if (decoded) return decoded

  const base64 = log.raw[base64Key]
  if (typeof base64 === 'string' && base64.length > 0) return `[base64]\n${base64}`
  return ''
}

function stageText(log: RawExchangeLogItem, direction: 'request' | 'response'): string {
  const raw = log.raw || {}
  const bodyKey = `${direction}_body`
  const base64Key = `${bodyKey}_base64`
  const payload: Record<string, unknown> = direction === 'request'
    ? {
        method: log.method,
        url: log.url || log.request_uri || log.path,
        headers: raw.request_headers || {},
        body: bodyPreview(log, bodyKey, base64Key),
        body_base64: raw[base64Key] || '',
        body_bytes: raw[`${bodyKey}_bytes`] || 0,
      }
    : {
        status_code: log.status_code,
        headers: raw.response_headers || {},
        body: bodyPreview(log, bodyKey, base64Key),
        body_base64: raw[base64Key] || '',
        body_bytes: raw[`${bodyKey}_bytes`] || 0,
        transport_error: raw.transport_error || '',
        read_error: raw.response_body_read_error || '',
      }
  return JSON.stringify(payload, null, 2)
}

function decodeBase64Text(value: unknown): string {
  if (typeof value !== 'string' || value.length === 0) return ''
  try {
    const binary = window.atob(value)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    return new TextDecoder().decode(bytes)
  } catch {
    return ''
  }
}

async function copySelectedRaw(): Promise<void> {
  await copyText(selectedRawJSON.value, '已复制完整请求链路')
}

async function copyText(text: string, successMessage: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess(successMessage)
  } catch {
    appStore.showError('复制失败')
  }
}

onMounted(() => {
  void loadLogs()
})
</script>
