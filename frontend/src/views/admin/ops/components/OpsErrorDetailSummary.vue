<template>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
    <SummaryItem :label="t('admin.ops.errorDetail.requestId')" mono :value="requestId || '—'" />
    <SummaryItem :label="t('admin.ops.errorDetail.time')" :value="formatDateTime(detail.created_at)" />
    <SummaryItem :label="actorLabel" :value="actorValue" />
    <SummaryItem :label="t('admin.ops.errorDetail.platform')" :value="detail.platform || '—'" />
    <SummaryItem :label="t('admin.ops.errorDetail.group')" :value="detail.group_name || formatNullableID(detail.group_id)" />
    <SummaryItem :label="t('admin.ops.errorDetail.model')" :value="modelValue" />
    <SummaryItem :label="t('admin.ops.errorDetail.inboundEndpoint')" mono :value="detail.inbound_endpoint || '—'" />
    <SummaryItem :label="t('admin.ops.errorDetail.upstreamEndpoint')" mono :value="detail.upstream_endpoint || '—'" />
    <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
      <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.errorDetail.status') }}</div>
      <div class="mt-1">
        <span :class="['inline-flex items-center rounded-lg px-2 py-1 text-xs font-black ring-1 ring-inset shadow-sm', statusClass]">
          {{ detail.status_code }}
        </span>
      </div>
    </div>
    <SummaryItem :label="t('admin.ops.errorDetail.requestType')" :value="formatRequestTypeLabel(detail.request_type)" />
    <SummaryItem :label="t('admin.ops.errorDetail.message')" truncate :value="detail.message || '—'" />
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsErrorDetail } from '@/api/admin/ops'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  detail: OpsErrorDetail
  requestId: string
}>()

const { t } = useI18n()

const actorLabel = computed(() => {
  return isUpstreamError(props.detail) ? t('admin.ops.errorDetail.account') : t('admin.ops.errorDetail.user')
})

const actorValue = computed(() => {
  if (isUpstreamError(props.detail)) {
    return props.detail.account_name || formatNullableID(props.detail.account_id)
  }
  return props.detail.user_email || formatNullableID(props.detail.user_id)
})

const modelValue = computed(() => {
  if (!hasModelMapping(props.detail)) return displayModel(props.detail) || '—'
  return `${props.detail.requested_model} -> ${props.detail.upstream_model}`
})

const statusClass = computed(() => {
  const code = props.detail.status_code ?? 0
  if (code >= 500) return 'bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-900/30 dark:text-red-400 dark:ring-red-500/30'
  if (code === 429) return 'bg-purple-50 text-purple-700 ring-purple-600/20 dark:bg-purple-900/30 dark:text-purple-400 dark:ring-purple-500/30'
  if (code >= 400) return 'bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-900/30 dark:text-amber-400 dark:ring-amber-500/30'
  return 'bg-gray-50 text-gray-700 ring-gray-600/20 dark:bg-gray-900/30 dark:text-gray-400 dark:ring-gray-500/30'
})

function isUpstreamError(d: OpsErrorDetail): boolean {
  const phase = String(d.phase || '').toLowerCase()
  const owner = String(d.error_owner || '').toLowerCase()
  return phase === 'upstream' && owner === 'provider'
}

function hasModelMapping(d: OpsErrorDetail): boolean {
  const requested = String(d.requested_model || '').trim()
  const upstream = String(d.upstream_model || '').trim()
  return !!requested && !!upstream && requested !== upstream
}

function displayModel(d: OpsErrorDetail): string {
  const upstream = String(d.upstream_model || '').trim()
  if (upstream) return upstream
  const requested = String(d.requested_model || '').trim()
  if (requested) return requested
  return String(d.model || '').trim()
}

function formatNullableID(id: number | null | undefined): string {
  return id != null ? String(id) : '—'
}

function formatRequestTypeLabel(type: number | null | undefined): string {
  switch (type) {
    case 1: return t('admin.ops.errorDetail.requestTypeSync')
    case 2: return t('admin.ops.errorDetail.requestTypeStream')
    case 3: return t('admin.ops.errorDetail.requestTypeWs')
    default: return t('admin.ops.errorDetail.requestTypeUnknown')
  }
}

const SummaryItem = defineComponent({
  name: 'SummaryItem',
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    mono: { type: Boolean, default: false },
    truncate: { type: Boolean, default: false }
  },
  setup(itemProps) {
    return () => h('div', { class: 'rounded-xl bg-gray-50 p-4 dark:bg-dark-900' }, [
      h('div', { class: 'text-xs font-bold uppercase tracking-wider text-gray-400' }, itemProps.label),
      h('div', {
        class: [
          'mt-1 text-sm font-medium text-gray-900 dark:text-white',
          itemProps.mono ? 'break-all font-mono' : '',
          itemProps.truncate ? 'truncate' : ''
        ],
        title: itemProps.truncate ? itemProps.value : undefined
      }, itemProps.value)
    ])
  }
})
</script>
