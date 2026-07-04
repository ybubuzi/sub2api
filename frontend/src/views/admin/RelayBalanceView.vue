<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.relayBalance.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('admin.relayBalance.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button class="btn btn-primary" @click="openCreate">
            <Icon name="plus" size="md" class="mr-1" />
            {{ t('admin.relayBalance.newStation') }}
          </button>
        </div>
      </div>

      <!-- Total Balance Card -->
      <section class="card p-4">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.relayBalance.totalBalance') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
              {{ t('admin.relayBalance.totalBalanceHint', { count: totalBalance.station_count }) }}
            </p>
          </div>
          <div class="text-right">
            <div class="text-3xl font-bold text-primary-600">
              {{ totalBalance.currency }} {{ totalBalance.total_balance.toFixed(2) }}
            </div>
          </div>
        </div>
      </section>

      <!-- Trend Chart -->
      <section class="card p-4">
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.relayBalance.balanceTrend') }}
          </h3>
          <div class="flex flex-wrap items-center gap-2">
            <Select
              v-model="trendGranularity"
              :options="trendGranularityOptions"
              @change="loadTrendData"
            />
            <Select
              v-model="trendRange"
              :options="trendRangeOptions"
              @change="handleTrendRangeChange"
            />
            <template v-if="trendRange === 'custom'">
              <input
                v-model="trendFrom"
                type="datetime-local"
                class="input w-48"
                @change="loadTrendData"
              />
              <input
                v-model="trendTo"
                type="datetime-local"
                class="input w-48"
                @change="loadTrendData"
              />
            </template>
          </div>
        </div>
        <div class="h-64">
          <div v-if="trendLoading" class="flex h-full items-center justify-center">
            <LoadingSpinner size="md" />
          </div>
          <Line
            v-else-if="trendChartData"
            :data="trendChartData"
            :options="trendChartOptions"
          />
          <div
            v-else
            class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
          >
            {{ t('admin.relayBalance.noTrendData') }}
          </div>
        </div>
      </section>

      <section class="card p-4">
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
          <input v-model="search" class="input sm:max-w-xs" :placeholder="t('admin.relayBalance.searchPlaceholder')" @input="loadStations" />
          <select v-model="enabledFilter" class="input sm:w-40" @change="loadStations">
            <option value="">{{ t('admin.relayBalance.allStatus') }}</option>
            <option value="true">{{ t('common.enabled') }}</option>
            <option value="false">{{ t('common.disabled') }}</option>
          </select>
        </div>

        <DataTable :columns="stationColumns" :data="stations" :loading="loading">
          <template #cell-name="{ row }">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
              <div class="max-w-md truncate text-xs text-gray-500">{{ row.base_url }}</div>
            </div>
          </template>
          <template #cell-enabled="{ value }">
            <span :class="['badge', value ? 'badge-success' : 'badge-gray']">{{ value ? t('common.enabled') : t('common.disabled') }}</span>
          </template>
          <template #cell-last_balance="{ row }">
            <span v-if="row.last_balance !== null" class="font-mono text-sm">{{ row.last_balance }} {{ row.last_currency }}</span>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
          <template #cell-last_status="{ row }">
            <span :class="['badge', row.last_status === 'success' ? 'badge-success' : row.last_status ? 'badge-danger' : 'badge-gray']">
              {{ row.last_status || '-' }}
            </span>
            <div v-if="row.last_error" class="mt-1 max-w-xs truncate text-xs text-red-500">{{ row.last_error }}</div>
          </template>
          <template #cell-last_run_at="{ value }">
            <span class="text-sm text-gray-500">{{ value ? formatDateTime(value) : '-' }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-blue-50 hover:text-blue-600" :title="t('admin.relayBalance.runNow')" @click="handleRun(row)">
                <Icon name="play" size="sm" />
              </button>
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700" :title="t('common.edit')" @click="openEdit(row)">
                <Icon name="edit" size="sm" />
              </button>
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-red-50 hover:text-red-600" :title="t('common.delete')" @click="handleDelete(row)">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>
        </DataTable>
      </section>

      <section class="card p-4">
        <div class="mb-4 flex flex-col gap-4">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.relayBalance.history') }}</h2>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-secondary" @click="exportRunsCsv">{{ t('admin.relayBalance.exportCsv') }}</button>
              <button class="btn btn-secondary" @click="loadRuns">{{ t('common.refresh') }}</button>
            </div>
          </div>
          <div class="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
            <select v-model="runFilters.station_id" class="input" @change="resetRunsPageAndLoad">
              <option value="">{{ t('admin.relayBalance.allStations') }}</option>
              <option v-for="station in stations" :key="station.id" :value="station.id">{{ station.name }}</option>
            </select>
            <select v-model="runFilters.status" class="input" @change="resetRunsPageAndLoad">
              <option value="">{{ t('admin.relayBalance.allStatus') }}</option>
              <option value="success">success</option>
              <option value="failed">failed</option>
            </select>
            <input v-model="runFilters.started_from" type="datetime-local" class="input" @change="resetRunsPageAndLoad" />
            <input v-model="runFilters.started_to" type="datetime-local" class="input" @change="resetRunsPageAndLoad" />
            <select v-model="runFilters.sort_order" class="input" @change="resetRunsPageAndLoad">
              <option value="desc">{{ t('admin.relayBalance.timeDesc') }}</option>
              <option value="asc">{{ t('admin.relayBalance.timeAsc') }}</option>
            </select>
            <select v-model="runFilters.granularity" class="input" @change="resetRunsPageAndLoad">
              <option value="">{{ t('admin.relayBalance.rawRecords') }}</option>
              <option value="hour">{{ t('admin.relayBalance.byHour') }}</option>
              <option value="day">{{ t('admin.relayBalance.byDay') }}</option>
            </select>
          </div>
          <div class="flex flex-wrap items-center gap-3 text-sm text-gray-500 dark:text-dark-300">
            <span>{{ t('admin.relayBalance.pageSize') }}</span>
            <select v-model.number="runsPagination.page_size" class="input w-24" @change="resetRunsPageAndLoad">
              <option :value="10">10</option>
              <option :value="20">20</option>
              <option :value="50">50</option>
              <option :value="100">100</option>
            </select>
            <span>{{ t('common.total') }}: {{ runsPagination.total }}</span>
            <button class="btn btn-secondary" :disabled="runsPagination.page <= 1" @click="changeRunsPage(runsPagination.page - 1)">{{ t('pagination.previous') }}</button>
            <span>{{ runsPagination.page }} / {{ runsTotalPages }}</span>
            <button class="btn btn-secondary" :disabled="runsPagination.page >= runsTotalPages" @click="changeRunsPage(runsPagination.page + 1)">{{ t('pagination.next') }}</button>
          </div>
        </div>
        <DataTable :columns="runColumns" :data="runs" :loading="runsLoading">
          <template #cell-balance="{ row }">
            <span v-if="row.balance !== null" class="font-mono text-sm">{{ row.balance }} {{ row.currency }}</span>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
          <template #cell-status="{ row }">
            <div class="cursor-pointer" @click="toggleError(row.id)">
              <span :class="['badge', row.status === 'success' ? 'badge-success' : 'badge-danger']">{{ row.status }}</span>
              <div v-if="row.error && expandedErrorId === row.id" class="mt-1 max-w-md truncate text-xs text-red-500">{{ row.error }}</div>
              <div v-if="row.stderr && expandedErrorId === row.id" class="mt-1 max-w-lg whitespace-pre-wrap break-all rounded bg-red-50 p-1.5 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">{{ row.stderr }}</div>
              <span v-else-if="(row.error || row.stderr) && expandedErrorId !== row.id" class="text-xs text-gray-400">({{ t('admin.relayBalance.clickToViewError') }})</span>
            </div>
          </template>
          <template #cell-started_at="{ value }">
            <span class="text-sm text-gray-500">{{ formatDateTime(value) }}</span>
          </template>
        </DataTable>
      </section>
    </div>

    <div v-if="showDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <form class="max-h-[90vh] w-full max-w-5xl overflow-y-auto rounded-2xl bg-white p-6 shadow-xl dark:bg-dark-800" @submit.prevent="saveStation">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ editing ? t('admin.relayBalance.editStation') : t('admin.relayBalance.newStation') }}</h2>
          <button type="button" class="text-gray-400 hover:text-gray-600" @click="showDialog = false">×</button>
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.relayBalance.name') }}</label>
            <input v-model="form.name" required class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.relayBalance.baseUrl') }}</label>
            <input v-model="form.base_url" required class="input" placeholder="https://example.com" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.relayBalance.cron') }}</label>
            <input v-model="form.cron_expression" required class="input font-mono" placeholder="0 * * * *" />
          </div>
          <label class="flex items-center gap-2 pt-7 text-sm text-gray-700 dark:text-dark-200">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
            {{ t('common.enabled') }}
          </label>
        </div>
        <div class="mt-4">
          <label class="input-label">package.json</label>
          <textarea v-model="form.package_json" rows="5" class="input font-mono text-xs" />
        </div>
        <div class="mt-4">
          <label class="input-label">{{ t('admin.relayBalance.script') }}</label>
          <textarea v-model="form.script" required rows="16" class="input font-mono text-xs" spellcheck="false" />
          <p class="mt-2 text-xs text-gray-500">{{ t('admin.relayBalance.scriptHint') }}</p>
        </div>
        <div class="mt-6 flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="showDialog = false">{{ t('common.cancel') }}</button>
          <button type="submit" class="btn btn-primary" :disabled="saving">{{ t('common.save') }}</button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import relayBalanceAPI, { type RelayBalanceRun, type RelayBalanceStation, type RelayBalanceStationRequest, type RelayBalanceTrendResponse } from '@/api/admin/relayBalance'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores'
import { formatDateTime } from '@/utils/format'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const runsLoading = ref(false)
const saving = ref(false)
const search = ref('')
const enabledFilter = ref('')
const stations = ref<RelayBalanceStation[]>([])
const runs = ref<RelayBalanceRun[]>([])
const showDialog = ref(false)
const editing = ref<RelayBalanceStation | null>(null)
const runsPagination = reactive({ page: 1, page_size: 20, total: 0 })
const runFilters = reactive({
  station_id: '',
  status: '',
  started_from: '',
  started_to: '',
  sort_order: 'desc' as 'asc' | 'desc',
  granularity: '' as '' | 'hour' | 'day'
})
const expandedErrorId = ref<number | null>(null)

function toggleError(id: number) {
  expandedErrorId.value = expandedErrorId.value === id ? null : id
}

// Total Balance
const totalBalance = ref<{ total_balance: number; currency: string; station_count: number }>({
  total_balance: 0,
  currency: 'USD',
  station_count: 0
})

// Trend Chart
const trendLoading = ref(false)
const trendGranularity = ref<'hour' | 'day'>('hour')
const trendRange = ref('7')
const trendFrom = ref('')
const trendTo = ref('')
const trendData = ref<RelayBalanceTrendResponse | null>(null)

const trendGranularityOptions = computed(() => [
  { value: 'hour', label: t('admin.relayBalance.byHour') },
  { value: 'day', label: t('admin.relayBalance.byDay') }
])

const trendRangeOptions = computed(() => [
  { value: '1', label: t('admin.relayBalance.last1Day') },
  { value: '7', label: t('admin.relayBalance.last7Days') },
  { value: '30', label: t('admin.relayBalance.last30Days') },
  { value: '90', label: t('admin.relayBalance.last90Days') },
  { value: 'custom', label: t('admin.relayBalance.customRange') }
])

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

const trendPalette = [
  '#3b82f6', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899',
  '#14b8a6', '#f97316', '#6366f1', '#84cc16', '#06b6d4'
]

const trendChartData = computed(() => {
  if (!trendData.value || !trendData.value.buckets.length) return null

  const labels = trendData.value.buckets.map((bucket: string) => {
    const date = new Date(bucket)
    return trendGranularity.value === 'hour'
      ? `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`
      : `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`
  })

  const datasets: any[] = trendData.value.series.map((series, idx) => {
    const color = trendPalette[idx % trendPalette.length]
    return {
      label: series.station_name,
      data: series.balances,
      borderColor: color,
      backgroundColor: `${color}20`,
      fill: false,
      tension: 0.3,
      pointRadius: 2,
      pointHoverRadius: 5
    }
  })

  if (trendData.value.total.length > 0) {
    datasets.push({
      label: t('admin.relayBalance.totalBalance'),
      data: trendData.value.total,
      borderColor: '#ef4444',
      backgroundColor: '#ef444420',
      borderWidth: 3,
      fill: false,
      tension: 0.3,
      pointRadius: 3,
      pointHoverRadius: 6
    })
  }

  return { labels, datasets }
})

const trendChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: { size: 11 }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => `${context.dataset.label}: ${Number(context.raw).toFixed(4)}`
      }
    }
  },
  scales: {
    x: {
      grid: { color: chartColors.value.grid },
      ticks: { color: chartColors.value.text, font: { size: 10 } }
    },
    y: {
      grid: { color: chartColors.value.grid },
      ticks: {
        color: chartColors.value.text,
        font: { size: 10 },
        callback: (value: string | number) => Number(value).toFixed(2)
      }
    }
  }
}))

async function loadTrendData() {
  trendLoading.value = true
  try {
    const params: any = {
      granularity: trendGranularity.value
    }

    if (trendRange.value === 'custom') {
      if (trendFrom.value && trendTo.value) {
        params.started_from = new Date(trendFrom.value).toISOString()
        params.started_to = new Date(trendTo.value).toISOString()
      }
    } else {
      const days = parseInt(trendRange.value)
      const now = new Date()
      const from = new Date(now.getTime() - days * 24 * 60 * 60 * 1000)
      params.started_from = from.toISOString()
      params.started_to = now.toISOString()
    }

    const res = await relayBalanceAPI.getTrend(params)
    trendData.value = res
  } catch (err: any) {
    console.error('Failed to load trend data:', err)
  } finally {
    trendLoading.value = false
  }
}

const defaultScript = `export default async function run(ctx) {
  const res = await fetch(ctx.baseUrl)
  const data = await res.json()
  return {
    balance: Number(data.balance),
    currency: data.currency || 'USD',
    raw: data
  }
}`

const form = reactive<RelayBalanceStationRequest>({
  name: '',
  base_url: '',
  cron_expression: '0 * * * *',
  enabled: false,
  package_json: '{\n  "type": "module"\n}',
  script: defaultScript
})

const stationColumns: Column[] = [
  { key: 'name', label: t('admin.relayBalance.station') },
  { key: 'enabled', label: t('common.status') },
  { key: 'cron_expression', label: 'Cron' },
  { key: 'last_balance', label: t('admin.relayBalance.balance') },
  { key: 'last_status', label: t('admin.relayBalance.lastStatus') },
  { key: 'last_run_at', label: t('admin.relayBalance.lastRun') },
  { key: 'actions', label: t('common.actions') }
]

const runColumns: Column[] = [
  { key: 'station_name', label: t('admin.relayBalance.station') },
  { key: 'balance', label: t('admin.relayBalance.balance') },
  { key: 'status', label: t('common.status') },
  { key: 'duration_ms', label: t('admin.relayBalance.duration') },
  { key: 'started_at', label: t('admin.relayBalance.startedAt') }
]

const runsTotalPages = computed(() => Math.max(1, Math.ceil(runsPagination.total / runsPagination.page_size)))

function resetForm() {
  Object.assign(form, {
    name: '',
    base_url: '',
    cron_expression: '0 * * * *',
    enabled: false,
    package_json: '{\n  "type": "module"\n}',
    script: defaultScript
  })
}

function openCreate() {
  editing.value = null
  resetForm()
  showDialog.value = true
}

function openEdit(row: RelayBalanceStation) {
  editing.value = row
  Object.assign(form, {
    name: row.name,
    base_url: row.base_url,
    cron_expression: row.cron_expression,
    enabled: row.enabled,
    package_json: row.package_json,
    script: row.script
  })
  showDialog.value = true
}

async function loadStations() {
  loading.value = true
  try {
    const res = await relayBalanceAPI.listStations(1, 50, { search: search.value, enabled: enabledFilter.value })
    stations.value = res.items
  } finally {
    loading.value = false
  }
}

async function loadRuns() {
  runsLoading.value = true
  try {
    const res = await relayBalanceAPI.listRuns(runsPagination.page, runsPagination.page_size, buildRunFilters())
    runs.value = res.items
    runsPagination.total = res.total
    runsPagination.page = res.page
    runsPagination.page_size = res.page_size
  } finally {
    runsLoading.value = false
  }
}

function buildRunFilters() {
  return {
    station_id: runFilters.station_id || undefined,
    status: runFilters.status || undefined,
    started_from: toRFC3339(runFilters.started_from),
    started_to: toRFC3339(runFilters.started_to),
    sort_order: runFilters.sort_order,
    granularity: runFilters.granularity
  }
}

function toRFC3339(value: string) {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString()
}

function resetRunsPageAndLoad() {
  runsPagination.page = 1
  loadRuns()
}

function changeRunsPage(page: number) {
  runsPagination.page = Math.min(Math.max(1, page), runsTotalPages.value)
  loadRuns()
}

async function exportRunsCsv() {
  try {
    const res = await relayBalanceAPI.listRuns(1, 10000, buildRunFilters())
    const rows = res.items.map(run => [
      run.station_name,
      run.balance ?? '',
      run.currency || '',
      run.status,
      run.duration_ms,
      formatDateTime(run.started_at),
      run.finished_at ? formatDateTime(run.finished_at) : '',
      run.error || ''
    ])
    const csv = [
      ['station', 'balance', 'currency', 'status', 'duration_ms', 'started_at', 'finished_at', 'message'],
      ...rows
    ].map(row => row.map(csvCell).join(',')).join('\n')
    const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `relay-balance-runs-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

function csvCell(value: unknown) {
  const text = String(value ?? '')
  return `"${text.replace(/"/g, '""')}"`
}

async function loadAll() {
  await Promise.all([loadStations(), loadRuns(), loadTotalBalance(), loadTrendData()])
}

async function loadTotalBalance() {
  try {
    const res = await relayBalanceAPI.getTotalBalance()
    totalBalance.value = res
  } catch (err: any) {
    console.error('Failed to load total balance:', err)
  }
}

function handleTrendRangeChange() {
  if (trendRange.value !== 'custom') {
    loadTrendData()
  }
}

watch(trendGranularity, () => {
  loadTrendData()
})

async function saveStation() {
  saving.value = true
  try {
    if (editing.value) {
      await relayBalanceAPI.updateStation(editing.value.id, form)
    } else {
      await relayBalanceAPI.createStation(form)
    }
    appStore.showSuccess(t('common.saved'))
    showDialog.value = false
    await loadStations()
  } catch (err: any) {
    let detail = err?.message || t('common.unknownError')
    if (err?.reason) {
      detail += ` (${err.reason})`
    }
    if (err?.metadata) {
      detail += ` ${JSON.stringify(err.metadata)}`
    }
    console.error('saveStation failed:', err)
    appStore.showError(detail)
  } finally {
    saving.value = false
  }
}

async function handleRun(row: RelayBalanceStation) {
  try {
    await relayBalanceAPI.runStation(row.id)
    appStore.showSuccess(t('admin.relayBalance.runQueued'))
    await loadAll()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.relayBalance.runFailed'))
  }
}

async function handleDelete(row: RelayBalanceStation) {
  if (!confirm(t('admin.relayBalance.deleteConfirm'))) return
  await relayBalanceAPI.deleteStation(row.id)
  appStore.showSuccess(t('common.deleted'))
  await loadAll()
}

onMounted(loadAll)
</script>
