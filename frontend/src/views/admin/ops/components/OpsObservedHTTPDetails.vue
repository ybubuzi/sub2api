<template>
  <div class="space-y-4">
    <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <PayloadBlock :title="t('admin.ops.errorDetail.requestHeaders')" :content="requestHeaders" />
      <PayloadBlock :title="t('admin.ops.errorDetail.responseHeaders')" :content="responseHeaders" />
      <PayloadBlock
        :title="t('admin.ops.errorDetail.requestBody')"
        :content="requestBody"
        :bytes="detail?.request_body_bytes ?? undefined"
        :truncated="!!detail?.request_body_truncated"
      />
      <PayloadBlock
        :title="t('admin.ops.errorDetail.responseBody')"
        :content="responseBody"
        :bytes="detail?.response_body_bytes ?? undefined"
        :truncated="!!detail?.response_body_truncated"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsErrorDetail } from '@/api/admin/ops'
import { resolvePrimaryResponseBody } from '../utils/errorDetailResponse'

const props = defineProps<{
  detail: OpsErrorDetail | null
  errorType?: 'request' | 'upstream'
}>()

const { t } = useI18n()

const requestHeaders = computed(() => props.detail?.request_headers || '')
const requestBody = computed(() => props.detail?.request_body || '')
const responseHeaders = computed(() => props.detail?.response_headers || '')
const responseBody = computed(() => {
  const observed = String(props.detail?.response_body || '').trim()
  return observed || resolvePrimaryResponseBody(props.detail, props.errorType)
})

function prettyJSON(raw?: string): string {
  if (!raw) return 'N/A'
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

const PayloadBlock = defineComponent({
  name: 'PayloadBlock',
  props: {
    title: { type: String, required: true },
    content: { type: String, default: '' },
    bytes: { type: Number, default: null },
    truncated: { type: Boolean, default: false }
  },
  setup(blockProps) {
    return () => h('div', { class: 'rounded-xl bg-gray-50 p-6 dark:bg-dark-900' }, [
      h('div', { class: 'flex flex-wrap items-center justify-between gap-2' }, [
        h('h3', { class: 'text-sm font-black uppercase tracking-wider text-gray-900 dark:text-white' }, blockProps.title),
        h('div', { class: 'flex items-center gap-2 text-xs font-medium text-gray-500 dark:text-gray-400' }, [
          blockProps.bytes != null ? h('span', `${blockProps.bytes} B`) : null,
          blockProps.truncated ? h('span', { class: 'rounded-md bg-amber-100 px-2 py-0.5 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300' }, t('admin.ops.errorDetail.truncated')) : null
        ])
      ]),
      h('pre', { class: 'mt-4 max-h-[420px] overflow-auto rounded-xl border border-gray-200 bg-white p-4 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100' }, [
        h('code', prettyJSON(blockProps.content))
      ])
    ])
  }
})
</script>
