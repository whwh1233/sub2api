<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip
} from 'chart.js'
import { Line } from 'vue-chartjs'
import {
  opsAPI,
  type OpsRPMDimension,
  type OpsRPMTrendResponse
} from '@/api/admin/ops'
import EmptyState from '@/components/common/EmptyState.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

interface Props {
  refreshToken?: number
}

const props = withDefaults(defineProps<Props>(), { refreshToken: 0 })
const { t, locale } = useI18n()

type RPMTimeRange = '30m' | '1h' | '6h' | '24h' | '7d' | '30d'
type RPMMetric = 'total' | 'success' | 'error'

const dimensions: OpsRPMDimension[] = ['platform', 'model', 'account', 'user']
const timeRanges: RPMTimeRange[] = ['30m', '1h', '6h', '24h', '7d', '30d']
const metrics: RPMMetric[] = ['total', 'success', 'error']

const dimension = ref<OpsRPMDimension>('platform')
const timeRange = ref<RPMTimeRange>('1h')
const metric = ref<RPMMetric>('total')
const search = ref('')
const response = ref<OpsRPMTrendResponse | null>(null)
const loading = ref(true)
const errorMessage = ref('')
let controller: AbortController | null = null

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))
const palette = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
  '#06b6d4', '#f97316', '#ec4899', '#84cc16', '#6366f1', '#64748b'
]

const visibleSeries = computed(() => {
  const items = response.value?.series ?? []
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) return items
  return items.filter((series) =>
    `${series.label} ${series.key}`.toLocaleLowerCase().includes(query)
  )
})

const formatPlatformLabel = (key: string, label: string) => {
  if (key === '__other__') return t('admin.ops.rpm.other')
  if (key === 'anthropic') return t('admin.ops.rpm.platformAnthropic')
  if (key === 'openai') return 'OpenAI'
  if (key === 'grok') return 'Grok'
  return label || key
}

const formatSeriesLabel = (key: string, label: string) => {
  if (key === '__other__') return t('admin.ops.rpm.other')
  if (dimension.value === 'platform') return formatPlatformLabel(key, label)
  return label || key
}

const pointValue = (point: { total_rpm: number; success_rpm: number; error_rpm: number }) => {
  if (metric.value === 'success') return point.success_rpm
  if (metric.value === 'error') return point.error_rpm
  return point.total_rpm
}

const formatBucketLabel = (iso: string) => {
  const date = new Date(iso)
  const longRange = timeRange.value === '7d' || timeRange.value === '30d'
  return new Intl.DateTimeFormat(locale.value, {
    month: longRange ? '2-digit' : undefined,
    day: longRange ? '2-digit' : undefined,
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(date)
}

const chartData = computed(() => {
  const series = visibleSeries.value
  if (!series.length) return null
  const bucketSet = new Set<string>()
  series.forEach((item) => item.points.forEach((point) => bucketSet.add(point.bucket_start)))
  const buckets = Array.from(bucketSet).sort()
  if (!buckets.length) return null

  return {
    labels: buckets.map(formatBucketLabel),
    datasets: series.map((item, index) => {
      const values = new Map(item.points.map((point) => [point.bucket_start, pointValue(point)]))
      const color = palette[index % palette.length]
      return {
        label: formatSeriesLabel(item.key, item.label),
        data: buckets.map((bucket) => values.get(bucket) ?? 0),
        borderColor: color,
        backgroundColor: `${color}18`,
        fill: false,
        tension: 0.25,
        pointRadius: 0,
        pointHitRadius: 10,
        borderWidth: item.key === '__other__' ? 1.5 : 2,
        borderDash: item.key === '__other__' ? [5, 4] : []
      }
    })
  }
})

const chartOptions = computed(() => {
  const text = isDarkMode.value ? '#9ca3af' : '#6b7280'
  const grid = isDarkMode.value ? '#374151' : '#f3f4f6'
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: 'index' as const },
    plugins: {
      legend: {
        position: 'top' as const,
        align: 'start' as const,
        labels: { color: text, usePointStyle: true, boxWidth: 7, font: { size: 10 } }
      },
      tooltip: {
        backgroundColor: isDarkMode.value ? '#1f2937' : '#ffffff',
        titleColor: isDarkMode.value ? '#f3f4f6' : '#111827',
        bodyColor: isDarkMode.value ? '#d1d5db' : '#4b5563',
        borderColor: grid,
        borderWidth: 1,
        callbacks: {
          label: (context: any) => `${context.dataset.label}: ${Number(context.parsed.y || 0).toFixed(1)} RPM`
        }
      }
    },
    scales: {
      x: {
        grid: { display: false },
        ticks: { color: text, maxTicksLimit: 10, autoSkip: true, font: { size: 10 } }
      },
      y: {
        beginAtZero: true,
        grid: { color: grid },
        ticks: { color: text, font: { size: 10 } },
        title: { display: true, text: 'RPM', color: text }
      }
    }
  }
})

const bucketDescription = computed(() => {
  const seconds = response.value?.output_bucket_seconds
  if (!seconds) return ''
  if (seconds === 60) return t('admin.ops.rpm.bucketMinute')
  if (seconds === 300) return t('admin.ops.rpm.bucketFiveMinutes')
  if (seconds === 1800) return t('admin.ops.rpm.bucketThirtyMinutes')
  if (seconds === 7200) return t('admin.ops.rpm.bucketTwoHours')
  return t('admin.ops.rpm.bucketSeconds', { seconds })
})

const completeThrough = computed(() => {
  if (!response.value?.complete_through) return ''
  return new Intl.DateTimeFormat(locale.value, {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
  }).format(new Date(response.value.complete_through))
})

async function fetchRPMTrend() {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  errorMessage.value = ''
  try {
    response.value = await opsAPI.getRPMTrend({
      time_range: timeRange.value,
      dimension: dimension.value,
      top_n: 10
    }, { signal: controller.signal })
  } catch (error: any) {
    if (error?.code === 'ERR_CANCELED') return
    response.value = null
    errorMessage.value = error?.message || t('admin.ops.rpm.failedToLoad')
  } finally {
    loading.value = false
  }
}

watch([dimension, timeRange], fetchRPMTrend)
watch(() => props.refreshToken, fetchRPMTrend)
onMounted(fetchRPMTrend)
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <section
    class="min-w-0 rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700 sm:p-6"
    data-testid="rpm-trend-card"
  >
    <header class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
      <div class="min-w-0">
        <h3 class="flex items-center gap-2 text-sm font-bold text-gray-900 dark:text-white">
          <svg class="h-4 w-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 19V9m5 10V5m5 14v-7m5 7V3" />
          </svg>
          {{ t('admin.ops.rpm.title') }}
          <HelpTooltip :content="t('admin.ops.rpm.help')" />
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ bucketDescription }}
          <span v-if="completeThrough"> · {{ t('admin.ops.rpm.completeThrough', { time: completeThrough }) }}</span>
        </p>
      </div>

      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <label class="sr-only" for="rpm-series-search">{{ t('admin.ops.rpm.search') }}</label>
        <input
          id="rpm-series-search"
          v-model="search"
          type="search"
          class="w-36 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-800 outline-none transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100"
          :placeholder="t('admin.ops.rpm.search')"
        />
        <div class="inline-flex flex-wrap rounded-lg bg-gray-100 p-0.5 dark:bg-dark-900" role="group" :aria-label="t('admin.ops.rpm.timeRange')">
          <button
            v-for="range in timeRanges"
            :key="range"
            type="button"
            class="cursor-pointer rounded-md px-2 py-1 text-[11px] font-semibold transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            :class="timeRange === range ? 'bg-white text-blue-600 shadow-sm dark:bg-dark-700 dark:text-blue-400' : 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-pressed="timeRange === range"
            @click="timeRange = range"
          >
            {{ range }}
          </button>
        </div>
      </div>
    </header>

    <div class="mt-4 flex flex-col gap-3 border-b border-gray-100 pb-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 overflow-x-auto" role="tablist" :aria-label="t('admin.ops.rpm.dimension')" data-testid="rpm-dimension-tabs">
        <button
          v-for="item in dimensions"
          :key="item"
          type="button"
          role="tab"
          class="cursor-pointer whitespace-nowrap border-b-2 px-4 py-2 text-xs font-semibold transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500"
          :class="dimension === item ? 'border-blue-500 text-blue-600 dark:text-blue-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
          :aria-selected="dimension === item"
          @click="dimension = item"
        >
          {{ t(`admin.ops.rpm.dimensions.${item}`) }}
        </button>
      </div>

      <div class="inline-flex self-start rounded-lg bg-gray-100 p-0.5 dark:bg-dark-900" role="group" :aria-label="t('admin.ops.rpm.metric')">
        <button
          v-for="item in metrics"
          :key="item"
          type="button"
          class="cursor-pointer rounded-md px-2.5 py-1 text-[11px] font-semibold transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          :class="metric === item ? 'bg-white text-blue-600 shadow-sm dark:bg-dark-700 dark:text-blue-400' : 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
          :aria-pressed="metric === item"
          @click="metric = item"
        >
          {{ t(`admin.ops.rpm.metrics.${item}`) }}
        </button>
      </div>
    </div>

    <div v-if="errorMessage" class="mt-4 rounded-xl bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400" role="alert">
      {{ errorMessage }}
    </div>

    <div class="mt-4 h-80 min-w-0">
      <div v-if="loading" class="flex h-full items-center justify-center">
        <span class="animate-pulse text-sm text-gray-400">{{ t('common.loading') }}</span>
      </div>
      <Line v-else-if="chartData" :data="chartData" :options="chartOptions" />
      <div v-else class="flex h-full items-center justify-center">
        <EmptyState :title="t('common.noData')" :description="t('admin.ops.rpm.noData')" />
      </div>
    </div>
  </section>
</template>
