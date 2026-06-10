<template>
  <div class="space-y-4">
    <div>
      <label class="input-label">{{ t('admin.accounts.dataImportDuplicateAction') }}</label>
      <div
        class="grid grid-cols-3 gap-1 rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-800"
      >
        <button
          v-for="option in duplicateActionOptions"
          :key="option.value"
          type="button"
          class="min-h-9 rounded-md px-2 text-sm font-medium transition-colors"
          :class="
            duplicateAction === option.value
              ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
              : 'text-gray-600 hover:bg-white/70 dark:text-dark-300 dark:hover:bg-dark-700/70'
          "
          @click="emit('update:duplicateAction', option.value)"
        >
          {{ t(option.labelKey) }}
        </button>
      </div>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.dataImportBatchProxy') }}</label>
      <select class="input" :value="proxySelectValue" @change="handleProxyChange">
        <option value="">{{ t('admin.accounts.dataImportBatchProxyFile') }}</option>
        <option value="0">{{ t('admin.accounts.dataImportBatchProxyNone') }}</option>
        <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
          {{ proxy.name }} - {{ proxy.host }}:{{ proxy.port }}
        </option>
      </select>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.dataImportPlatformGroups') }}</label>
      <div class="grid gap-2 sm:grid-cols-2">
        <div v-for="platform in platforms" :key="platform" class="min-w-0">
          <div class="mb-1 text-xs font-medium text-gray-500 dark:text-dark-400">
            {{ t(`admin.groups.platforms.${platform}`) }}
          </div>
          <select class="input" :value="groupSelectValue(platform)" @change="handleGroupChange(platform, $event)">
            <option value="">{{ t('admin.accounts.dataImportPlatformGroupNone') }}</option>
            <option v-for="group in groupsByPlatform[platform]" :key="group.id" :value="String(group.id)">
              {{ group.name }}
            </option>
          </select>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type {
  AccountPlatform,
  AdminGroup,
  DataImportPlatformGroupIDs,
  DuplicateAccountAction,
  Proxy
} from '@/types'

const platforms: AccountPlatform[] = ['anthropic', 'openai', 'gemini', 'antigravity', 'kiro']

const duplicateActionOptions: Array<{
  value: DuplicateAccountAction
  labelKey: string
}> = [
  { value: 'overwrite', labelKey: 'admin.accounts.dataImportDuplicateOverwrite' },
  { value: 'copy', labelKey: 'admin.accounts.dataImportDuplicateCopy' },
  { value: 'ignore', labelKey: 'admin.accounts.dataImportDuplicateIgnore' }
]

const props = defineProps<{
  show: boolean
  duplicateAction: DuplicateAccountAction
  platformGroupIds: DataImportPlatformGroupIDs
  proxyId: number | null
}>()

const emit = defineEmits<{
  (e: 'update:duplicateAction', value: DuplicateAccountAction): void
  (e: 'update:platformGroupIds', value: DataImportPlatformGroupIDs): void
  (e: 'update:proxyId', value: number | null): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const groups = defineModel<AdminGroup[]>('groups', { default: [] })
const proxies = defineModel<Proxy[]>('proxies', { default: [] })

const groupsByPlatform = computed<Record<AccountPlatform, AdminGroup[]>>(() => {
  const out: Record<AccountPlatform, AdminGroup[]> = {
    anthropic: [],
    openai: [],
    gemini: [],
    antigravity: [],
    kiro: []
  }
  for (const group of groups.value) {
    if (platforms.includes(group.platform as AccountPlatform)) {
      out[group.platform as AccountPlatform].push(group)
    }
  }
  return out
})

const proxySelectValue = computed(() => (props.proxyId === null ? '' : String(props.proxyId)))

watch(
  () => props.show,
  async open => {
    if (!open) return
    try {
      const [loadedGroups, loadedProxies] = await Promise.all([
        adminAPI.groups.getAll(),
        adminAPI.proxies.getAll()
      ])
      groups.value = loadedGroups
      proxies.value = loadedProxies
    } catch (error: any) {
      appStore.showError(error?.message || t('admin.accounts.dataImportOptionsLoadFailed'))
    }
  },
  { immediate: true }
)

const groupSelectValue = (platform: AccountPlatform): string => {
  const ids = props.platformGroupIds[platform]
  return ids && ids.length > 0 ? String(ids[0]) : ''
}

const handleProxyChange = (event: Event) => {
  const value = (event.target as HTMLSelectElement).value
  emit('update:proxyId', value === '' ? null : Number(value))
}

const handleGroupChange = (platform: AccountPlatform, event: Event) => {
  const value = (event.target as HTMLSelectElement).value
  const next: DataImportPlatformGroupIDs = { ...props.platformGroupIds }
  if (value === '') {
    delete next[platform]
  } else {
    next[platform] = [Number(value)]
  }
  emit('update:platformGroupIds', next)
}
</script>
