<template>
  <AppLayout>
    <section class="mx-auto max-w-5xl space-y-6">
      <header class="page-header">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="flex gap-4">
            <div class="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-xl border border-primary-200 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-200">
              <svg class="h-8 w-8" fill="none" viewBox="0 0 32 32" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 25v-6.5A1.5 1.5 0 0110.5 17H14v8M14 25V13.5A1.5 1.5 0 0115.5 12h1A1.5 1.5 0 0118 13.5V25M18 25v-8h3.5a1.5 1.5 0 011.5 1.5V25M7 25h18" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M9.5 10.5l2.2-5.5L16 9l4.3-4 2.2 5.5h-13zM12 15h8" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 19h3M4 15h4M24 15h4" />
              </svg>
            </div>
            <div>
              <p class="text-sm font-medium text-primary-600 dark:text-primary-400">
                {{ t('leaderboard.today') }} · {{ leaderboard?.date || todayLabel }}
              </p>
              <h1 class="page-title">{{ t('leaderboard.title') }}</h1>
              <p class="page-description max-w-2xl">{{ t('leaderboard.description') }}</p>
            </div>
          </div>
          <button
            type="button"
            class="btn-secondary inline-flex min-h-11 items-center gap-2 px-4"
            :disabled="loading"
            :title="t('leaderboard.refresh')"
            @click="loadLeaderboard"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            <span>{{ loading ? t('leaderboard.refreshing') : t('leaderboard.refresh') }}</span>
          </button>
        </div>
      </header>

      <div
        v-if="loading && !leaderboard"
        class="leaderboard-skeleton grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]"
        aria-busy="true"
      >
        <article class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="h-5 w-28 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="row in 3"
              :key="row"
              class="grid grid-cols-[2.75rem_minmax(0,1fr)] gap-3 px-5 py-4 sm:grid-cols-[2.75rem_minmax(0,1fr)_9rem_7rem]"
            >
              <div class="h-11 w-11 animate-pulse rounded-lg bg-gray-200 dark:bg-dark-700"></div>
              <div class="min-w-0 space-y-2">
                <div class="h-4 w-40 max-w-full animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
                <div class="h-3 w-24 animate-pulse rounded bg-gray-100 dark:bg-dark-700/70"></div>
              </div>
              <div class="hidden space-y-2 sm:block">
                <div class="ml-auto h-3 w-14 animate-pulse rounded bg-gray-100 dark:bg-dark-700/70"></div>
                <div class="ml-auto h-4 w-24 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
              </div>
              <div class="hidden space-y-2 sm:block">
                <div class="ml-auto h-3 w-12 animate-pulse rounded bg-gray-100 dark:bg-dark-700/70"></div>
                <div class="ml-auto h-4 w-10 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
              </div>
            </div>
          </div>
        </article>

        <aside class="card p-5">
          <div class="h-4 w-20 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          <div class="mt-5 h-9 w-24 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          <div class="mt-5 grid grid-cols-2 gap-3">
            <div class="h-20 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
            <div class="h-20 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
          </div>
          <div class="mt-4 h-12 animate-pulse rounded-lg bg-primary-100/70 dark:bg-primary-500/10"></div>
        </aside>
      </div>

      <div v-else-if="error && !leaderboard" class="card p-6" role="status" aria-live="polite">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex gap-3">
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300">
              <Icon name="exclamationTriangle" size="md" />
            </div>
            <div>
              <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.loadFailed') }}</p>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('leaderboard.loadFailedDetail') }}</p>
            </div>
          </div>
          <button type="button" class="btn-secondary inline-flex min-h-11 items-center gap-2 px-4" @click="loadLeaderboard">
            <Icon name="refresh" size="sm" />
            <span>{{ t('leaderboard.retry') }}</span>
          </button>
        </div>
      </div>

      <template v-else-if="leaderboard">
        <div
          v-if="error"
          class="flex gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200"
          role="status"
          aria-live="polite"
        >
          <Icon name="exclamationTriangle" size="sm" class="mt-0.5 flex-shrink-0" />
          <div>
            <p class="font-semibold">{{ t('leaderboard.loadFailed') }}</p>
            <p class="mt-0.5">{{ t('leaderboard.loadFailedDetail') }}</p>
          </div>
        </div>

        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
          <article class="card overflow-hidden">
            <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.topThree') }}</h2>
                <p v-if="leaderboard.top.length > 0" class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                  {{ t('leaderboard.topTotal') }} {{ formatCompactTokens(topTotalTokens) }} {{ t('leaderboard.tokens') }}
                </p>
              </div>
              <div
                v-if="entryPaceTokens > 0"
                class="inline-flex w-fit items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200"
              >
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                {{ t('leaderboard.entryPace') }} {{ formatCompactTokens(entryPaceTokens) }} {{ t('leaderboard.tokens') }}
              </div>
            </div>

            <div v-if="leaderboard.top.length === 0" class="p-8 text-center">
              <p class="text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.emptyTitle') }}</p>
              <p class="mt-2 text-sm text-gray-500 dark:text-dark-300">{{ t('leaderboard.emptyDescription') }}</p>
            </div>

            <div
              v-else
              data-testid="leaderboard-podium"
              class="grid gap-4 px-5 py-5 md:grid-cols-3 md:items-end"
            >
              <article
                v-for="item in podiumItems"
                :key="item.user_id"
                class="podium-place flex min-h-[16rem] flex-col rounded-lg border p-4 shadow-sm transition-transform duration-200 hover:-translate-y-0.5"
                :class="[podiumPlaceClass(item.rank), podiumVisualOrderClass(item.rank)]"
                :data-podium-rank="item.rank"
              >
                <div class="flex items-start justify-between gap-3">
                  <div
                    class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-lg border shadow-sm"
                    :class="rankIconClass(item.rank)"
                    :data-rank-icon="rankMedal(item.rank)"
                    :aria-label="rankLabel(item.rank)"
                    role="img"
                  >
                    <svg v-if="item.rank === 1" class="h-7 w-7" fill="none" viewBox="0 0 32 32" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M7 21h18M9 25h14M8 18l2.8-9 5.2 5 5.2-5L24 18H8z" />
                      <path stroke-linecap="round" stroke-linejoin="round" d="M16 5v3M11 7.5l1.5 2M21 7.5l-1.5 2" />
                    </svg>
                    <svg v-else-if="item.rank === 2" class="h-7 w-7" fill="none" viewBox="0 0 32 32" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 5h8l-2 7h-4L12 5zM16 12v4" />
                      <circle cx="16" cy="21" r="6" />
                      <path stroke-linecap="round" stroke-linejoin="round" d="M14 21h4M14 24h4" />
                    </svg>
                    <svg v-else class="h-7 w-7" fill="none" viewBox="0 0 32 32" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M16 5l8 3v7c0 5-3.4 9.4-8 11-4.6-1.6-8-6-8-11V8l8-3z" />
                      <path stroke-linecap="round" stroke-linejoin="round" d="M13 14h6l-3 3h1.2a3 3 0 11-2.7 4.3" />
                    </svg>
                  </div>
                  <div class="rounded-lg px-2.5 py-1 text-sm font-black tabular-nums" :class="rankBadgeClass(item.rank)">
                    P{{ item.rank }}
                  </div>
                </div>

                <div class="mt-4 min-w-0">
                  <p class="text-xs font-bold uppercase text-gray-500 dark:text-dark-300">{{ rankLabel(item.rank) }}</p>
                  <div class="mt-2 flex flex-wrap items-center gap-2">
                    <p class="truncate text-lg font-black text-gray-900 dark:text-white">{{ item.display_name }}</p>
                    <span
                      v-if="item.is_current_user"
                      class="rounded-full bg-primary-100 px-2 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-500/20 dark:text-primary-200"
                    >
                      {{ t('leaderboard.currentUser') }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                    {{ formatCompactCount(item.requests) }} {{ t('leaderboard.requests') }}
                  </p>
                </div>

                <div class="mt-5">
                  <p class="text-xs font-medium text-gray-500 dark:text-dark-300">{{ t('leaderboard.tokens') }}</p>
                  <p
                    class="mt-1 text-3xl font-black tabular-nums text-gray-900 dark:text-white"
                    :title="t('leaderboard.fullTokens', { tokens: formatFullNumber(item.total_tokens) })"
                  >
                    {{ formatCompactTokens(item.total_tokens) }}
                  </p>
                </div>

                <div class="mt-auto pt-5">
                  <div
                    class="flex flex-col justify-end rounded-lg border px-4 py-3"
                    :class="podiumStepClass(item.rank)"
                  >
                    <div class="flex items-end justify-between gap-3">
                      <span class="text-4xl font-black leading-none tabular-nums">#{{ item.rank }}</span>
                      <span class="text-xs font-semibold text-gray-500 dark:text-dark-300">{{ t('leaderboard.rank') }}</span>
                    </div>
                    <div class="mt-3 h-2 overflow-hidden rounded-full bg-white/70 dark:bg-dark-950/40">
                      <div
                        class="h-full rounded-full transition-all duration-500"
                        :class="rankBarClass(item.rank)"
                        :style="{ width: `${rankSpeedPercent(item)}%` }"
                      ></div>
                    </div>
                  </div>
                </div>
              </article>
            </div>

            <section
              v-if="chaseItems.length > 0"
              data-testid="leaderboard-chasers"
              class="border-t border-gray-100 px-5 py-5 dark:border-dark-700"
            >
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.chasingBoard') }}</h3>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ t('leaderboard.chasingBoardDescription') }}</p>
                </div>
                <span class="inline-flex w-fit items-center rounded-lg border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-bold text-primary-700 dark:border-primary-400/30 dark:bg-primary-500/10 dark:text-primary-200">
                  {{ t('leaderboard.topTen') }}
                </span>
              </div>

              <div class="mt-4 overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700">
                <div
                  v-for="item in chaseItems"
                  :key="item.user_id"
                  class="grid gap-3 border-b border-gray-100 px-4 py-3 last:border-b-0 dark:border-dark-700 sm:grid-cols-[3.25rem_minmax(0,1fr)_8rem_8.5rem] sm:items-center"
                  :class="item.is_current_user ? 'bg-primary-50/70 dark:bg-primary-500/10' : 'bg-white dark:bg-dark-900'"
                  :data-chaser-rank="item.rank"
                >
                  <div class="flex h-10 w-10 items-center justify-center rounded-lg border border-gray-200 bg-gray-50 text-sm font-black tabular-nums text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-100">
                    #{{ item.rank }}
                  </div>

                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ item.display_name }}</p>
                      <span
                        v-if="item.is_current_user"
                        class="rounded-full bg-primary-100 px-2 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-500/20 dark:text-primary-200"
                      >
                        {{ t('leaderboard.currentUser') }}
                      </span>
                    </div>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                      {{ t('leaderboard.tokensBehindTopThree', { tokens: formatCompactTokens(tokensBehindPodium(item)) }) }}
                    </p>
                  </div>

                  <div>
                    <p class="text-xs text-gray-500 dark:text-dark-300">{{ t('leaderboard.tokens') }}</p>
                    <p
                      class="mt-1 text-sm font-bold tabular-nums text-gray-900 dark:text-white"
                      :title="t('leaderboard.fullTokens', { tokens: formatFullNumber(item.total_tokens) })"
                    >
                      {{ formatCompactTokens(item.total_tokens) }}
                    </p>
                  </div>

                  <div>
                    <div class="flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-dark-300">
                      <span>{{ formatCompactCount(item.requests) }} {{ t('leaderboard.requests') }}</span>
                      <span class="font-semibold text-gray-700 dark:text-dark-100">P{{ item.rank }}</span>
                    </div>
                    <div class="mt-2 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
                      <div
                        class="h-full rounded-full bg-primary-500 transition-all duration-500"
                        :style="{ width: `${rankSpeedPercent(item)}%` }"
                      ></div>
                    </div>
                  </div>
                </div>
              </div>
            </section>
          </article>

          <aside class="card p-5">
            <p class="text-sm font-medium text-primary-600 dark:text-primary-400">{{ t('leaderboard.myRank') }}</p>
            <div class="mt-4 space-y-4">
              <div>
                <p class="truncate text-sm text-gray-500 dark:text-dark-300">{{ leaderboard.me.display_name }}</p>
                <p class="mt-1 text-3xl font-bold tabular-nums text-gray-900 dark:text-white">
                  {{ leaderboard.me.rank ? `#${leaderboard.me.rank}` : t('leaderboard.notRanked') }}
                </p>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                  <p class="text-xs text-gray-500 dark:text-dark-300">{{ t('leaderboard.tokens') }}</p>
                  <p
                    class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white"
                    :title="t('leaderboard.fullTokens', { tokens: formatFullNumber(leaderboard.me.total_tokens) })"
                  >
                    {{ formatCompactTokens(leaderboard.me.total_tokens) }}
                  </p>
                </div>
                <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                  <p class="text-xs text-gray-500 dark:text-dark-300">{{ t('leaderboard.requests') }}</p>
                  <p class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatCompactCount(leaderboard.me.requests) }}</p>
                </div>
              </div>
              <p class="rounded-lg bg-primary-50 p-3 text-sm font-medium text-primary-700 dark:bg-primary-500/10 dark:text-primary-200">
                {{ leaderboard.me.tokens_to_top_three > 0
                  ? t('leaderboard.tokensToTopThree', { tokens: formatCompactTokens(leaderboard.me.tokens_to_top_three) })
                  : t('leaderboard.onLeaderboard') }}
              </p>
            </div>
          </aside>
        </div>
      </template>
    </section>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI, type DailyLeaderboardResponse } from '@/api/usage'

const { t } = useI18n()

const leaderboard = ref<DailyLeaderboardResponse | null>(null)
const loading = ref(false)
const error = ref(false)

const numberFormatter = new Intl.NumberFormat()
const compactUnits = [
  { value: 1_000_000_000_000, suffix: 'T' },
  { value: 1_000_000_000, suffix: 'B' },
  { value: 1_000_000, suffix: 'M' },
  { value: 1_000, suffix: 'K' },
]

const todayLabel = computed(() => {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
})

const podiumItems = computed(() => leaderboard.value?.top.slice(0, 3) || [])
const chaseItems = computed(() => leaderboard.value?.top.slice(3, 10) || [])
const topMaxTokens = computed(() => Math.max(...(leaderboard.value?.top.map((item) => item.total_tokens) || [0]), 1))
const topTotalTokens = computed(() => podiumItems.value.reduce((sum, item) => sum + item.total_tokens, 0))
const entryPaceTokens = computed(() => {
  const lastPodiumItem = podiumItems.value[podiumItems.value.length - 1]
  return lastPodiumItem?.total_tokens || 0
})

const formatFullNumber = (value?: number | null) => numberFormatter.format(value || 0)

const formatCompactValue = (value?: number | null) => {
  const numericValue = Math.abs(value || 0)
  const unit = compactUnits.find((candidate) => numericValue >= candidate.value)
  if (!unit) {
    return formatFullNumber(value)
  }
  const scaled = numericValue / unit.value
  const maximumFractionDigits = scaled >= 1000 ? 0 : scaled >= 10 ? 1 : 2
  const formatted = new Intl.NumberFormat(undefined, {
    maximumFractionDigits,
    minimumFractionDigits: 0,
  }).format(scaled)
  return `${value && value < 0 ? '-' : ''}${formatted}${unit.suffix}`
}

const formatCompactTokens = (value?: number | null) => formatCompactValue(value)
const formatCompactCount = (value?: number | null) => formatCompactValue(value)

const rankSpeedPercent = (item: { total_tokens: number }) => {
  if (item.total_tokens <= 0) {
    return 8
  }
  return Math.max(12, Math.min(100, Math.round((item.total_tokens / topMaxTokens.value) * 100)))
}

const tokensBehindPodium = (item: { total_tokens: number }) => {
  if (entryPaceTokens.value <= 0) {
    return 0
  }
  return Math.max(0, entryPaceTokens.value - item.total_tokens + 1)
}

const rankLabel = (rank?: number | null) => {
  if (rank === 1) return t('leaderboard.champion')
  if (rank === 2) return t('leaderboard.runnerUp')
  if (rank === 3) return t('leaderboard.thirdPlace')
  return t('leaderboard.rank')
}

const rankMedal = (rank?: number | null) => {
  if (rank === 1) return 'champion'
  if (rank === 2) return 'runner-up'
  if (rank === 3) return 'third-place'
  return 'rank'
}

const podiumVisualOrderClass = (rank?: number | null) => {
  if (rank === 1) return 'md:order-2'
  if (rank === 2) return 'md:order-1'
  if (rank === 3) return 'md:order-3'
  return ''
}

const rankBadgeClass = (rank?: number | null) => {
  if (rank === 1) return 'border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-400/40 dark:bg-amber-500/10 dark:text-amber-200'
  if (rank === 2) return 'border-slate-300 bg-slate-50 text-slate-700 dark:border-slate-400/40 dark:bg-slate-500/10 dark:text-slate-100'
  if (rank === 3) return 'border-orange-300 bg-orange-50 text-orange-700 dark:border-orange-400/40 dark:bg-orange-500/10 dark:text-orange-200'
  return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-100'
}

const rankIconClass = (rank?: number | null) => {
  if (rank === 1) return 'border-amber-300 bg-amber-100 text-amber-700 dark:border-amber-400/50 dark:bg-amber-400/15 dark:text-amber-200'
  if (rank === 2) return 'border-slate-300 bg-slate-100 text-slate-700 dark:border-slate-400/40 dark:bg-slate-400/10 dark:text-slate-100'
  if (rank === 3) return 'border-orange-300 bg-orange-100 text-orange-700 dark:border-orange-400/40 dark:bg-orange-400/10 dark:text-orange-200'
  return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-100'
}

const podiumPlaceClass = (rank?: number | null) => {
  if (rank === 1) {
    return 'podium-place-champion border-amber-300 bg-gradient-to-b from-amber-50 via-white to-white shadow-[0_18px_44px_rgba(245,158,11,0.18)] dark:border-amber-400/40 dark:from-amber-500/15 dark:via-dark-900 dark:to-dark-900 md:min-h-[21rem]'
  }
  if (rank === 2) {
    return 'podium-place-runner-up border-slate-300 bg-white dark:border-slate-400/30 dark:bg-dark-900 md:min-h-[18.5rem]'
  }
  if (rank === 3) {
    return 'podium-place-third border-orange-300 bg-white dark:border-orange-400/30 dark:bg-dark-900 md:min-h-[17.25rem]'
  }
  return 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900'
}

const podiumStepClass = (rank?: number | null) => {
  if (rank === 1) return 'min-h-28 border-amber-200 bg-amber-100 text-amber-800 dark:border-amber-400/30 dark:bg-amber-400/10 dark:text-amber-100'
  if (rank === 2) return 'min-h-24 border-slate-200 bg-slate-100 text-slate-800 dark:border-slate-400/30 dark:bg-slate-400/10 dark:text-slate-100'
  if (rank === 3) return 'min-h-20 border-orange-200 bg-orange-100 text-orange-800 dark:border-orange-400/30 dark:bg-orange-400/10 dark:text-orange-100'
  return 'min-h-20 border-gray-200 bg-gray-50 text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100'
}

const rankBarClass = (rank?: number | null) => {
  if (rank === 1) return 'bg-amber-500'
  if (rank === 2) return 'bg-slate-500'
  if (rank === 3) return 'bg-orange-500'
  return 'bg-primary-500'
}

const loadLeaderboard = async () => {
  if (loading.value) {
    return
  }
  loading.value = true
  error.value = false
  try {
    leaderboard.value = await usageAPI.getDailyLeaderboard()
  } catch (err) {
    if (import.meta.env.DEV) {
      console.warn('Failed to load daily leaderboard:', err)
    }
    error.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadLeaderboard()
})
</script>
