<template>
  <UptimeChart
    :data="data"
    :loading="loading"
    :window-value="windowValue"
    :dimension="dimension"
    :window-options="windowOptions"
    :dimension-options="dimensionOptions"
    @update:window-value="handleWindowChange"
    @update:dimension="handleDimensionChange"
    @refresh="loadData"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import UptimeChart from '@/components/charts/UptimeChart.vue'
import {
  getAdminUptime,
  getUserUptime,
  type AdminUptimeDimension,
  type UptimeChartResponse,
  type UptimeWindow,
  type UserUptimeDimension
} from '@/api/uptime'

type UptimeScope = 'user' | 'admin'
type UptimeDimension = UserUptimeDimension | AdminUptimeDimension

interface UptimeSelectOption extends Record<string, unknown> {
  value: string
  label: string
}

const props = defineProps<{
  scope: UptimeScope
}>()

const { t } = useI18n()
const appStore = useAppStore()
const data = ref<UptimeChartResponse | null>(null)
const loading = ref(false)
const windowValue = ref<UptimeWindow>('1h')
const dimension = ref<UptimeDimension>('all')
let abortController: AbortController | null = null

const windowOptions = computed<UptimeSelectOption[]>(() => [
  { value: '1h', label: t('usage.uptimeWindows.oneHour') },
  { value: '6h', label: t('usage.uptimeWindows.sixHours') }
])

const dimensionOptions = computed<UptimeSelectOption[]>(() => {
  if (props.scope === 'admin') {
    return [
      { value: 'all', label: t('usage.uptimeDimensions.allSite') },
      { value: 'model', label: t('usage.uptimeDimensions.model') },
      { value: 'group', label: t('usage.uptimeDimensions.group') }
    ]
  }
  return [
    { value: 'all', label: t('usage.uptimeDimensions.all') },
    { value: 'api_key', label: t('usage.uptimeDimensions.apiKey') },
    { value: 'model', label: t('usage.uptimeDimensions.model') }
  ]
})

const loadData = async () => {
  abortController?.abort()
  const currentAbortController = new AbortController()
  abortController = currentAbortController
  loading.value = true
  try {
    data.value = await requestUptime(currentAbortController.signal)
  } catch (error: any) {
    if (isAbortError(error)) return
    appStore.showError(t('usage.uptimeFailedToLoad'))
  } finally {
    if (abortController === currentAbortController) {
      loading.value = false
    }
  }
}

const requestUptime = (signal: AbortSignal) => {
  if (props.scope === 'admin') {
    return getAdminUptime(
      { window: windowValue.value, dimension: dimension.value as AdminUptimeDimension },
      { signal }
    )
  }
  return getUserUptime(
    { window: windowValue.value, dimension: dimension.value as UserUptimeDimension },
    { signal }
  )
}

const handleWindowChange = (value: UptimeWindow) => {
  if (windowValue.value === value) return
  windowValue.value = value
  void loadData()
}

const handleDimensionChange = (value: string) => {
  if (dimension.value === value) return
  dimension.value = value as UptimeDimension
  void loadData()
}

const isAbortError = (error: any): boolean => {
  return error?.name === 'AbortError' || error?.code === 'ERR_CANCELED'
}

onMounted(() => {
  void loadData()
})

onUnmounted(() => {
  abortController?.abort()
})
</script>
