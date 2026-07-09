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

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,560px)_minmax(0,1fr)]">
        <section class="card min-w-0 overflow-hidden">
          <div class="border-b border-gray-100 p-4 dark:border-dark-800">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">请求列表</h2>
              <span class="text-xs text-gray-500 dark:text-gray-400">共 {{ total }} 条</span>
            </div>
            <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="logPath">
              {{ logPath || '尚未返回日志文件路径' }}
            </p>
          </div>

          <div v-if="loading && !logs.length" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            正在读取原文日志...
          </div>
          <div v-else-if="!logs.length" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            暂无匹配记录。发起一次接口请求后再刷新。
          </div>
          <div v-else class="max-h-[720px] divide-y divide-gray-100 overflow-y-auto dark:divide-dark-800">
            <button
              v-for="item in logs"
              :key="item.line"
              type="button"
              class="block w-full px-4 py-3 text-left transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 dark:hover:bg-dark-800"
              :class="selectedLine === item.line ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
              @click="selectedLine = item.line"
            >
              <div class="flex flex-wrap items-center gap-2">
                <span class="rounded px-2 py-0.5 text-xs font-semibold" :class="statusClass(item.status_code)">
                  {{ item.status_code || '-' }}
                </span>
                <span class="font-mono text-xs font-semibold text-gray-700 dark:text-gray-200">{{ item.method || '-' }}</span>
                <span class="min-w-0 flex-1 truncate font-mono text-xs text-gray-900 dark:text-white" :title="item.request_uri || item.path">
                  {{ item.request_uri || item.path || '-' }}
                </span>
              </div>
              <div class="mt-2 grid grid-cols-2 gap-2 text-xs text-gray-500 dark:text-gray-400">
                <div class="truncate">
                  <span class="text-gray-400">time</span>
                  {{ formatDate(item.completed_at) }}
                </div>
                <div class="truncate">
                  <span class="text-gray-400">latency</span>
                  {{ formatDuration(item.latency_ms) }}
                </div>
                <div class="truncate">
                  <span class="text-gray-400">rid</span>
                  <span class="font-mono">{{ item.request_id || '-' }}</span>
                </div>
                <div class="truncate">
                  <span class="text-gray-400">body</span>
                  {{ formatBytes(item.request_body_bytes) }} / {{ formatBytes(item.response_body_bytes) }}
                </div>
              </div>
            </button>
          </div>
        </section>

        <section class="card min-w-0 overflow-hidden">
          <div class="border-b border-gray-100 p-4 dark:border-dark-800">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">完整原文详情</h2>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">raw JSON 保留所有已捕获字段，不做脱敏。</p>
              </div>
              <div v-if="selectedLog" class="flex flex-wrap gap-2">
                <button type="button" class="btn btn-secondary btn-sm" @click="copySelectedRaw">复制 JSON</button>
                <button type="button" class="btn btn-secondary btn-sm" @click="copySelectedBodies">复制 Body</button>
              </div>
            </div>
          </div>

          <div v-if="!selectedLog" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            选择左侧请求查看完整原文。
          </div>

          <div v-else class="space-y-4 p-4">
            <div class="grid grid-cols-2 gap-3 text-xs lg:grid-cols-4">
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-gray-400">Request ID</div>
                <div class="mt-1 truncate font-mono text-gray-900 dark:text-white" :title="selectedLog.request_id">{{ selectedLog.request_id || '-' }}</div>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-gray-400">Client IP</div>
                <div class="mt-1 truncate font-mono text-gray-900 dark:text-white">{{ selectedLog.client_ip || '-' }}</div>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-gray-400">Platform / Model</div>
                <div class="mt-1 truncate font-mono text-gray-900 dark:text-white">{{ selectedLog.platform || '-' }} / {{ selectedLog.model || '-' }}</div>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-gray-500 dark:text-gray-400">Line</div>
                <div class="mt-1 truncate font-mono text-gray-900 dark:text-white">{{ selectedLog.line }}</div>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <div class="min-w-0">
                <div class="mb-2 flex items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">请求 Body 预览</h3>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    {{ formatBytes(selectedLog.request_body_bytes) }}
                    <span v-if="selectedLog.request_body_truncated"> / truncated</span>
                  </span>
                </div>
                <pre class="max-h-[300px] overflow-auto rounded-lg bg-gray-950 p-3 text-xs leading-relaxed text-gray-100">{{ requestBodyPreview }}</pre>
              </div>
              <div class="min-w-0">
                <div class="mb-2 flex items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">响应 Body 预览</h3>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    {{ formatBytes(selectedLog.response_body_bytes) }}
                    <span v-if="selectedLog.response_body_truncated"> / truncated</span>
                  </span>
                </div>
                <pre class="max-h-[300px] overflow-auto rounded-lg bg-gray-950 p-3 text-xs leading-relaxed text-gray-100">{{ responseBodyPreview }}</pre>
              </div>
            </div>

            <div class="min-w-0">
              <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">完整 raw JSON</h3>
                <span class="text-xs text-gray-500 dark:text-gray-400">包含 headers、query、body、base64 bytes、耗时和响应信息</span>
              </div>
              <pre class="max-h-[560px] overflow-auto rounded-lg bg-gray-950 p-4 text-xs leading-relaxed text-gray-100">{{ selectedRawJSON }}</pre>
            </div>
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
const selectedLine = ref<number | null>(null)
const total = ref(0)
const logPath = ref('')
const loading = ref(false)
const errorMessage = ref('')

const selectedLog = computed(() => logs.value.find((item) => item.line === selectedLine.value) ?? null)
const selectedRawJSON = computed(() => (selectedLog.value ? JSON.stringify(selectedLog.value.raw, null, 2) : ''))
const requestBodyPreview = computed(() => bodyPreview(selectedLog.value, 'request_body', 'request_body_base64'))
const responseBodyPreview = computed(() => bodyPreview(selectedLog.value, 'response_body', 'response_body_base64'))

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

    if (!logs.value.some((item) => item.line === selectedLine.value)) {
      selectedLine.value = logs.value[0]?.line ?? null
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
  await copyText(selectedRawJSON.value, '已复制完整 raw JSON')
}

async function copySelectedBodies(): Promise<void> {
  const text = [
    '--- request_body ---',
    requestBodyPreview.value,
    '',
    '--- response_body ---',
    responseBodyPreview.value,
  ].join('\n')
  await copyText(text, '已复制请求和响应 Body')
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
