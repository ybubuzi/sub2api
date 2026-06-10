<template>
  <div class="card p-4">
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <div class="min-w-0 flex-1">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('usage.uptimeChart') }}
        </h3>
        <div class="mt-2 flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span>{{ t('usage.uptimeTotal') }}: {{ formatCount(data?.summary.total || 0) }}</span>
          <span>{{ t('usage.uptimeSuccess') }}: {{ formatCount(data?.summary.success || 0) }}</span>
          <span>{{ t('usage.uptimeFailure') }}: {{ formatCount(data?.summary.failure || 0) }}</span>
          <span>{{ t('usage.uptimeAvailability') }}: {{ formatAvailability(data?.summary.availability) }}</span>
        </div>
      </div>
      <div class="grid w-full grid-cols-2 gap-2 sm:w-auto sm:min-w-[280px]">
        <Select
          :model-value="windowValue"
          :options="windowOptions"
          :disabled="loading"
          @update:model-value="updateWindow"
        />
        <Select
          :model-value="dimension"
          :options="dimensionOptions"
          :disabled="loading"
          @update:model-value="updateDimension"
        />
      </div>
      <button
        type="button"
        class="btn btn-secondary px-2"
        :disabled="loading"
        :title="t('common.refresh')"
        @click="$emit('refresh')"
      >
        <Icon name="refresh" size="sm" />
      </button>
    </div>

    <div v-if="loading" class="flex h-64 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div v-else-if="hasChartData && chartData" class="h-64">
      <Line :data="chartData" :options="lineOptions" />
    </div>
    <div
      v-else
      class="flex h-64 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('usage.uptimeNoData') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip
} from 'chart.js'
import { Line } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import type { UptimeChartResponse, UptimeWindow } from '@/api/uptime'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

type DimensionValue = string | number | boolean | null
interface UptimeSelectOption extends Record<string, unknown> {
  value: string | number | boolean | null
  label: string
  disabled?: boolean
}

const props = defineProps<{
  data: UptimeChartResponse | null
  loading?: boolean
  windowValue: UptimeWindow
  dimension: string
  windowOptions: UptimeSelectOption[]
  dimensionOptions: UptimeSelectOption[]
}>()

const emit = defineEmits<{
  (event: 'update:windowValue', value: UptimeWindow): void
  (event: 'update:dimension', value: string): void
  (event: 'refresh'): void
}>()

const { t } = useI18n()

const colors = [
  '#2563eb',
  '#059669',
  '#d97706',
  '#dc2626',
  '#7c3aed',
  '#0891b2',
  '#4f46e5',
  '#be123c',
  '#65a30d',
  '#9333ea'
]

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const chartTheme = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

const hasChartData = computed(() => {
  return !!props.data?.series.some((series) => series.summary.total > 0)
})

const chartData = computed(() => {
  const first = props.data?.series[0]
  if (!first) return null
  return {
    labels: first.buckets.map((bucket) => formatBucketLabel(bucket.start)),
    datasets: props.data.series.map((series, index) => {
      const color = colors[index % colors.length]
      return {
        label: series.label,
        data: series.buckets.map((bucket) =>
          bucket.availability == null ? null : Number((bucket.availability * 100).toFixed(2))
        ),
        borderColor: color,
        backgroundColor: `${color}1f`,
        fill: false,
        tension: 0.25,
        spanGaps: true,
        pointRadius: 2,
        pointHoverRadius: 4
      }
    })
  }
})

const lineOptions = computed(() => ({
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
        color: chartTheme.value.text,
        usePointStyle: true,
        boxWidth: 8,
        font: { size: 11 }
      }
    },
    tooltip: {
      callbacks: {
        label: (ctx: any) => `${ctx.dataset.label}: ${formatAvailability(ctx.raw / 100)}`
      }
    }
  },
  scales: {
    x: {
      grid: { color: chartTheme.value.grid },
      ticks: { color: chartTheme.value.text, maxRotation: 0, font: { size: 10 } }
    },
    y: {
      min: 0,
      max: 100,
      grid: { color: chartTheme.value.grid },
      ticks: {
        color: chartTheme.value.text,
        callback: (value: string | number) => `${value}%`,
        font: { size: 10 }
      }
    }
  }
}))

const updateWindow = (value: DimensionValue) => {
  if (value === '1h' || value === '6h') {
    emit('update:windowValue', value)
  }
}

const updateDimension = (value: DimensionValue) => {
  if (typeof value === 'string' && value !== props.dimension) {
    emit('update:dimension', value)
  }
}

const formatBucketLabel = (value: string): string => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatAvailability = (value: number | null | undefined): string => {
  if (value == null || Number.isNaN(value)) return '-'
  return `${(value * 100).toFixed(2)}%`
}

const formatCount = (value: number): string => value.toLocaleString()
</script>
