<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="metric in summaryMetrics"
              :key="metric.key"
              class="balance-metric-card"
              :class="metric.cardClass"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="text-sm font-medium text-gray-500 dark:text-dark-300">
                    {{ metric.label }}
                  </p>
                  <p class="mt-2 truncate text-2xl font-semibold text-gray-950 dark:text-white">
                    {{ metric.value }}
                  </p>
                </div>
                <div class="rounded-md p-2" :class="metric.iconClass">
                  <Icon :name="metric.icon" size="md" :stroke-width="2" />
                </div>
              </div>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <div class="relative w-full md:w-72">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.balances.searchPlaceholder')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>

            <div class="w-full sm:w-40">
              <Select
                v-model="statusFilter"
                :options="statusOptions"
                @change="applyFilters"
              />
            </div>

            <div class="w-full sm:w-44">
              <Select
                v-model="balanceStateFilter"
                data-testid="balance-state-filter"
                :options="balanceStateOptions"
                @change="applyFilters"
              />
            </div>

            <button
              type="button"
              class="btn btn-secondary px-3"
              :title="t('common.refresh')"
              :disabled="loading || summaryLoading"
              @click="refreshAll"
            >
              <Icon
                name="refresh"
                size="md"
                :class="loading || summaryLoading ? 'animate-spin' : ''"
              />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="users"
          :loading="loading"
          :actions-count="2"
          :server-side-sort="true"
          default-sort-key="balance"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-user="{ row }">
            <div class="flex items-center gap-3">
              <div
                class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-100 text-sm font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200"
              >
                {{ userInitial(row) }}
              </div>
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-950 dark:text-white">
                  {{ row.email }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ row.username || '-' }}
                </div>
              </div>
            </div>
          </template>

          <template #cell-balance="{ value }">
            <span class="font-semibold" :class="balanceClass(value)">
              {{ formatMoney(value) }}
            </span>
          </template>

          <template #cell-status="{ value }">
            <span class="inline-flex items-center gap-1.5 text-sm">
              <span
                class="h-2 w-2 rounded-full"
                :class="value === 'active' ? 'bg-emerald-500' : 'bg-gray-400'"
              ></span>
              <span class="text-gray-700 dark:text-gray-300">
                {{ value === 'active' ? t('common.active') : t('admin.users.disabled') }}
              </span>
            </span>
          </template>

          <template #cell-last_active_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ value ? formatDateTime(value) : '-' }}
            </span>
          </template>

          <template #cell-last_used_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ value ? formatDateTime(value) : '-' }}
            </span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ value ? formatDateTime(value) : '-' }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-emerald-700 transition-colors hover:bg-emerald-50 dark:text-emerald-300 dark:hover:bg-emerald-900/20"
                :title="t('admin.balances.actions.deposit')"
                @click="handleDeposit(row)"
              >
                <Icon name="plus" size="xs" :stroke-width="2" />
                <span>{{ t('admin.balances.actions.deposit') }}</span>
              </button>
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:text-dark-200 dark:hover:bg-dark-700"
                :title="t('admin.balances.actions.history')"
                @click="handleBalanceHistory(row)"
              >
                <Icon name="clock" size="xs" :stroke-width="2" />
                <span>{{ t('admin.balances.actions.history') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <div class="flex flex-col items-center">
              <Icon name="creditCard" size="xl" class="mb-3 text-gray-400 dark:text-dark-500" />
              <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
                {{ t('admin.balances.empty.title') }}
              </p>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <UserBalanceModal
      :show="showBalanceModal"
      :user="balanceUser"
      :operation="balanceOperation"
      @close="closeBalanceModal"
      @success="refreshAll"
    />
    <UserBalanceHistoryModal
      :show="showBalanceHistoryModal"
      :user="balanceHistoryUser"
      @close="closeBalanceHistoryModal"
      @deposit="handleDepositFromHistory"
      @withdraw="handleWithdrawFromHistory"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminBalanceSummary } from '@/api/admin'
import type { AdminUser } from '@/types'
import type { Column } from '@/components/common/types'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatDateTime } from '@/utils/format'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import UserBalanceModal from '@/components/admin/user/UserBalanceModal.vue'
import UserBalanceHistoryModal from '@/components/admin/user/UserBalanceHistoryModal.vue'

const { t } = useI18n()
const appStore = useAppStore()

const summary = ref<AdminBalanceSummary | null>(null)
const users = ref<AdminUser[]>([])
const loading = ref(false)
const summaryLoading = ref(false)
const searchQuery = ref('')
const statusFilter = ref('')
const balanceStateFilter = ref('')
let searchTimer: ReturnType<typeof setTimeout> | undefined

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})

const sortState = reactive<{ sort_by: string; sort_order: 'asc' | 'desc' }>({
  sort_by: 'balance',
  sort_order: 'desc',
})

const columns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.balances.columns.user'), sortable: true },
  { key: 'balance', label: t('admin.balances.columns.balance'), sortable: true },
  { key: 'status', label: t('admin.balances.columns.status'), sortable: true },
  { key: 'last_active_at', label: t('admin.balances.columns.lastActive'), sortable: true },
  { key: 'last_used_at', label: t('admin.balances.columns.lastUsed'), sortable: true },
  { key: 'created_at', label: t('admin.balances.columns.created'), sortable: true },
  { key: 'actions', label: t('admin.balances.columns.actions'), sortable: false },
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.users.allStatus') },
  { value: 'active', label: t('common.active') },
  { value: 'disabled', label: t('admin.users.disabled') },
])

const balanceStateOptions = computed(() => [
  { value: '', label: t('admin.balances.filters.allBalanceStates') },
  { value: 'positive', label: t('admin.balances.filters.positive') },
  { value: 'low', label: t('admin.balances.filters.low') },
  { value: 'abnormal', label: t('admin.balances.filters.abnormal') },
  { value: 'zero', label: t('admin.balances.filters.zero') },
])

const summaryMetrics = computed(() => [
  {
    key: 'total',
    label: t('admin.balances.summary.totalBalance'),
    value: formatMoney(summary.value?.total_balance ?? 0),
    icon: 'dollar' as const,
    cardClass: 'border-l-blue-500',
    iconClass: 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-300',
  },
  {
    key: 'positive',
    label: t('admin.balances.summary.positiveUsers'),
    value: String(summary.value?.positive_balance_users ?? 0),
    icon: 'users' as const,
    cardClass: 'border-l-emerald-500',
    iconClass: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-300',
  },
  {
    key: 'low',
    label: t('admin.balances.summary.lowBalanceUsers'),
    value: String(summary.value?.low_balance_users ?? 0),
    icon: 'exclamationTriangle' as const,
    cardClass: 'border-l-amber-500',
    iconClass: 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-300',
  },
  {
    key: 'abnormal',
    label: t('admin.balances.summary.abnormalUsers'),
    value: String(summary.value?.abnormal_balance_users ?? 0),
    icon: 'exclamationCircle' as const,
    cardClass: 'border-l-rose-500',
    iconClass: 'bg-rose-50 text-rose-600 dark:bg-rose-900/20 dark:text-rose-300',
  },
])

const showBalanceModal = ref(false)
const balanceUser = ref<AdminUser | null>(null)
const balanceOperation = ref<'add' | 'subtract'>('add')
const showBalanceHistoryModal = ref(false)
const balanceHistoryUser = ref<AdminUser | null>(null)

function formatMoney(value: number): string {
  const normalized = Number.isFinite(value) ? value : 0
  return `$${normalized.toFixed(2)}`
}

function balanceClass(value: number): string {
  const threshold = summary.value?.low_balance_threshold ?? 1
  if (value < 0) return 'text-rose-600 dark:text-rose-300'
  if (value > 0 && value <= threshold) return 'text-amber-600 dark:text-amber-300'
  if (value > 0) return 'text-emerald-700 dark:text-emerald-300'
  return 'text-gray-500 dark:text-dark-300'
}

function userInitial(user: AdminUser): string {
  const seed = user.username || user.email || String(user.id)
  return seed.trim().charAt(0).toUpperCase() || '#'
}

async function loadSummary() {
  summaryLoading.value = true
  try {
    summary.value = await adminAPI.balances.getSummary()
  } catch (error) {
    console.error('Failed to load balance summary:', error)
    appStore.showError?.(t('admin.balances.errors.summaryFailed'))
  } finally {
    summaryLoading.value = false
  }
}

async function loadUsers() {
  loading.value = true
  try {
    const response = await adminAPI.users.list(pagination.page, pagination.page_size, {
      search: searchQuery.value.trim() || undefined,
      status: statusFilter.value ? (statusFilter.value as 'active' | 'disabled') : undefined,
      balance_state: balanceStateFilter.value
        ? (balanceStateFilter.value as 'positive' | 'low' | 'abnormal' | 'zero')
        : undefined,
      include_subscriptions: false,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order,
    })
    users.value = response.items
    pagination.total = response.total
  } catch (error) {
    console.error('Failed to load balance users:', error)
    appStore.showError?.(t('admin.balances.errors.usersFailed'))
  } finally {
    loading.value = false
  }
}

async function refreshAll() {
  await Promise.all([loadSummary(), loadUsers()])
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1
    loadUsers()
  }, 250)
}

function applyFilters() {
  pagination.page = 1
  loadUsers()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key === 'user' ? 'email' : key
  sortState.sort_order = order
  pagination.page = 1
  loadUsers()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadUsers()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadUsers()
}

function handleDeposit(user: AdminUser) {
  balanceUser.value = user
  balanceOperation.value = 'add'
  showBalanceModal.value = true
}

function handleWithdraw(user: AdminUser) {
  balanceUser.value = user
  balanceOperation.value = 'subtract'
  showBalanceModal.value = true
}

function closeBalanceModal() {
  showBalanceModal.value = false
  balanceUser.value = null
}

function handleBalanceHistory(user: AdminUser) {
  balanceHistoryUser.value = user
  showBalanceHistoryModal.value = true
}

function closeBalanceHistoryModal() {
  showBalanceHistoryModal.value = false
  balanceHistoryUser.value = null
}

function handleDepositFromHistory() {
  if (balanceHistoryUser.value) {
    handleDeposit(balanceHistoryUser.value)
  }
}

function handleWithdrawFromHistory() {
  if (balanceHistoryUser.value) {
    handleWithdraw(balanceHistoryUser.value)
  }
}

onMounted(() => {
  refreshAll()
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style scoped>
.balance-metric-card {
  @apply rounded-lg border border-l-4 border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800;
}
</style>
