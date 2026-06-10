<template>
  <BaseDialog :show="show" :title="dialogTitle" width="wide" @close="close">
    <div v-if="group" class="space-y-4">
      <div class="flex flex-wrap items-center gap-3 rounded-lg bg-gray-50 px-4 py-2.5 text-sm dark:bg-dark-700">
        <span class="inline-flex items-center gap-1.5 text-gray-700 dark:text-gray-300">
          <PlatformIcon :platform="group.platform" size="sm" />
          {{ t('admin.groups.platforms.' + group.platform) }}
        </span>
        <span class="text-gray-400">|</span>
        <span class="font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
        <span v-if="isMirror" class="badge badge-primary">{{ t('admin.groups.mirror.badge') }}</span>
      </div>

      <div v-if="!isSupported" class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.groups.mirror.unsupported') }}
      </div>

      <template v-else>
        <div class="grid gap-4 md:grid-cols-[220px_minmax(0,1fr)]">
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
            <div class="mb-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.mirror.targetPlatform') }}
            </div>
            <div class="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
              <PlatformIcon :platform="targetPlatform" size="sm" />
              {{ t('admin.groups.platforms.' + targetPlatform) }}
            </div>
            <div v-if="isMirror" class="mt-3 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.mirror.sourceGroup', { id: group.mirror_source_group_id }) }}
            </div>
            <button
              v-else
              type="button"
              class="mt-4 inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-colors"
              :class="mirrorEnabled ? enabledClass : disabledClass"
              :disabled="saving || mirrorIndexLoading || Boolean(mirrorIndexError)"
              @click="mirrorEnabled = !mirrorEnabled"
            >
              <span
                class="h-2.5 w-2.5 rounded-full"
                :class="mirrorEnabled ? 'bg-emerald-500' : 'bg-gray-400'"
              />
              {{ mirrorEnabled ? t('admin.groups.mirror.enabled') : t('admin.groups.mirror.disabled') }}
            </button>
            <div v-if="mirrorIndexLoading" class="mt-3 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.mirror.loading') }}
            </div>
            <div v-if="mirrorIndexError" class="mt-3 text-xs text-red-600 dark:text-red-400">
              {{ mirrorIndexError }}
            </div>
          </div>

          <div class="rounded-lg border border-gray-200 dark:border-dark-600">
            <div class="flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-600">
              <div>
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.groups.mirror.mappingTitle') }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.mirror.mappingHint') }}
                </div>
                <div v-if="modelCandidates.loading.value" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.mirror.candidatesLoading') }}
                </div>
                <div v-if="modelCandidates.error.value" class="mt-1 text-xs text-red-600 dark:text-red-400">
                  {{ modelCandidates.error.value }}
                </div>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" @click="addRow">
                <Icon name="plus" size="sm" class="mr-1" />
                {{ t('admin.groups.mirror.addMapping') }}
              </button>
            </div>

            <div class="max-h-[420px] space-y-3 overflow-y-auto p-4">
              <datalist :id="clientModelListID">
                <option v-for="model in clientModelOptions" :key="model" :value="model" />
              </datalist>
              <datalist :id="sourceModelListID">
                <option v-for="model in sourceModelOptions" :key="model" :value="model" />
              </datalist>
              <div v-if="mappingRows.length === 0" class="py-6 text-center text-sm text-gray-400 dark:text-gray-500">
                {{ t('admin.groups.mirror.noMappings') }}
              </div>
              <div
                v-for="row in mappingRows"
                :key="row.id"
                class="grid gap-3 rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto]"
              >
                <input
                  v-model="row.from"
                  type="text"
                  class="input"
                  :list="clientModelListID"
                  :placeholder="t('admin.groups.mirror.clientModel')"
                />
                <div class="hidden items-center text-gray-400 md:flex">
                  <Icon name="arrowRight" size="sm" />
                </div>
                <input
                  v-model="row.to"
                  type="text"
                  class="input"
                  :list="sourceModelListID"
                  :placeholder="t('admin.groups.mirror.sourceModel')"
                />
                <button
                  type="button"
                  class="rounded-lg p-2 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                  @click="removeRow(row.id)"
                >
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="close">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="saveDisabled" @click="save">
          <Icon v-if="saving" name="refresh" size="sm" class="mr-1 animate-spin" />
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, GroupPlatform } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import {
  mergeMirrorModelCandidates
} from './groupMirrorModels'
import { useGroupMirrorModelCandidates } from './useGroupMirrorModelCandidates'

interface MappingRow {
  id: number
  from: string
  to: string
}

const props = defineProps<{
  show: boolean
  group: AdminGroup | null
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const mirrorIndexLoading = ref(false)
const mirrorIndexError = ref('')
const mirrorEnabled = ref(false)
const mappingRows = ref<MappingRow[]>([])
const mirrorIndex = ref<AdminGroup[]>([])
let rowID = 0
const modelCandidates = useGroupMirrorModelCandidates({
  errorMessage: () => t('admin.groups.mirror.candidatesLoadFailed'),
  onError: appStore.showError
})

const isMirror = computed(() => Boolean(props.group?.is_mirror || props.group?.mirror_source_group_id))
const isSupported = computed(() => props.group?.platform === 'openai' || props.group?.platform === 'anthropic')
const saveDisabled = computed(() => {
  return saving.value || !isSupported.value || mirrorIndexLoading.value || Boolean(mirrorIndexError.value)
})
const dialogTitle = computed(() => {
  return isMirror.value || sourceMirror.value
    ? t('admin.groups.mirror.editTitle')
    : t('admin.groups.mirror.title')
})
const targetPlatform = computed<GroupPlatform>(() => {
  if (!props.group) return 'anthropic'
  if (isMirror.value) return props.group.platform
  return props.group.platform === 'openai' ? 'anthropic' : 'openai'
})
const sourceMirror = computed(() => {
  if (!props.group || isMirror.value) return null
  return mirrorCandidates.value.find((candidate) => {
    return candidate.mirror_source_group_id === props.group?.id && candidate.platform === targetPlatform.value
  }) ?? null
})
const mirrorCandidates = computed(() => {
  return mirrorIndex.value.length > 0 ? mirrorIndex.value : props.groups
})
const clientModelOptions = computed(() => mergeMirrorModelCandidates({
  primary: modelCandidates.clientModels.value,
  secondary: modelCandidates.sourceModels.value,
  existing: mappingRows.value.map((row) => row.from)
}))
const sourceModelOptions = computed(() => mergeMirrorModelCandidates({
  primary: modelCandidates.sourceModels.value,
  existing: mappingRows.value.map((row) => row.to)
}))
const clientModelListID = computed(() => `mirror-client-models-${props.group?.id ?? 'new'}`)
const sourceModelListID = computed(() => `mirror-source-models-${props.group?.id ?? 'new'}`)

const enabledClass = 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300'
const disabledClass = 'border-gray-200 bg-white text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300'

watch(
  () => [props.show, props.group] as const,
  () => void hydrateState(),
  { immediate: true }
)

async function hydrateState(): Promise<void> {
  mirrorIndexError.value = ''
  mirrorIndex.value = []
  modelCandidates.clear()
  await loadMirrorIndex()
  resetState()
  await modelCandidates.load(props.group, props.show)
}

async function loadMirrorIndex(): Promise<void> {
  if (!props.show || !props.group || isMirror.value || !isSupported.value) return
  mirrorIndexLoading.value = true
  try {
    mirrorIndex.value = await loadAllGroups()
  } catch (error: any) {
    mirrorIndexError.value = error.response?.data?.detail || error.message || t('admin.groups.mirror.loadFailed')
    appStore.showError(mirrorIndexError.value)
  } finally {
    mirrorIndexLoading.value = false
  }
}

async function loadAllGroups(): Promise<AdminGroup[]> {
  const pageSize = 500
  const first = await adminAPI.groups.list(1, pageSize)
  const groups = [...first.items]
  for (let page = 2; groups.length < first.total; page += 1) {
    const next = await adminAPI.groups.list(page, pageSize)
    groups.push(...next.items)
  }
  return groups
}

function resetState(): void {
  if (!props.show || !props.group) return
  const mapping = sourceMirror.value?.mirror_model_mapping ?? props.group.mirror_model_mapping ?? {}
  mirrorEnabled.value = isMirror.value || Boolean(sourceMirror.value)
  mappingRows.value = Object.entries(mapping).map(([from, to]) => ({
    id: ++rowID,
    from,
    to
  }))
}

function addRow(): void {
  mappingRows.value.push({ id: ++rowID, from: '', to: '' })
}

function removeRow(id: number): void {
  mappingRows.value = mappingRows.value.filter((row) => row.id !== id)
}

function buildMapping(): Record<string, string> {
  const out: Record<string, string> = {}
  mappingRows.value.forEach((row) => {
    const from = row.from.trim()
    const to = row.to.trim()
    if (!from && !to) return
    if (!from || !to) throw new Error(t('admin.groups.mirror.mappingIncomplete'))
    if (out[from]) throw new Error(t('admin.groups.mirror.mappingDuplicate'))
    out[from] = to
  })
  return out
}

async function save(): Promise<void> {
  if (!props.group || !isSupported.value) return
  saving.value = true
  try {
    if (isMirror.value) {
      await adminAPI.groups.update(props.group.id, { mirror_model_mapping: buildMapping() })
    } else {
      await adminAPI.groups.setMirror(props.group.id, {
        target_platform: targetPlatform.value as 'anthropic' | 'openai',
        enabled: mirrorEnabled.value,
        mirror_model_mapping: buildMapping()
      })
    }
    appStore.showSuccess(t('admin.groups.mirror.saved'))
    emit('success')
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error.message || t('admin.groups.mirror.failed'))
  } finally {
    saving.value = false
  }
}

function close(): void {
  emit('close')
}
</script>
