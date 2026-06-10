<template>
  <BaseDialog :show="show" :title="t('admin.groups.relationship.title')" width="full" @close="close">
    <div class="space-y-4">
      <div class="flex flex-wrap items-end gap-3">
        <div class="w-44">
          <label class="input-label">{{ t('admin.groups.form.platform') }}</label>
          <select v-model="platformFilter" class="input">
            <option value="">{{ t('admin.groups.allPlatforms') }}</option>
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="gemini">Gemini</option>
            <option value="antigravity">Antigravity</option>
            <option value="kiro">Kiro</option>
          </select>
        </div>
        <div class="w-40">
          <label class="input-label">{{ t('admin.groups.relationship.maxKeys') }}</label>
          <input v-model.number="maxKeys" type="number" min="1" step="1" class="input" />
        </div>
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadGraph">
          <Icon name="refresh" size="sm" :class="['mr-1', loading && 'animate-spin']" />
          {{ t('common.refresh') }}
        </button>
        <div class="ml-auto text-sm text-gray-500 dark:text-gray-400">
          {{ summaryText }}
        </div>
      </div>

      <div v-if="loading" class="flex h-[560px] items-center justify-center rounded-lg border border-gray-200 dark:border-dark-600">
        <Icon name="refresh" size="lg" class="animate-spin text-primary-500" />
      </div>
      <GroupRelationshipGraphCanvas v-else :elements="elements" />

      <div class="grid gap-4 rounded-lg border border-gray-200 p-4 dark:border-dark-600 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto]">
        <div>
          <label class="input-label">{{ t('admin.groups.relationship.sourceGroup') }}</label>
          <select v-model.number="sourceGroupID" class="input">
            <option value="">{{ t('admin.groups.relationship.selectGroup') }}</option>
            <option v-for="group in transferGroups" :key="group.id" :value="group.id">
              {{ group.name }} · {{ group.platform }} · #{{ group.id }}
            </option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.relationship.targetGroup') }}</label>
          <select v-model.number="targetGroupID" class="input">
            <option value="">{{ t('admin.groups.relationship.selectGroup') }}</option>
            <option v-for="group in targetGroups" :key="group.id" :value="group.id">
              {{ group.name }} · {{ group.platform }} · #{{ group.id }}
            </option>
          </select>
        </div>
        <button type="button" class="btn btn-secondary self-end" :disabled="!canTransfer || transferLoading" @click="previewTransfer">
          {{ t('admin.groups.relationship.previewTransfer') }}
        </button>
        <button type="button" class="btn btn-primary self-end" :disabled="!canTransfer || transferLoading" @click="executeTransfer">
          <Icon v-if="transferLoading" name="refresh" size="sm" class="mr-1 animate-spin" />
          {{ t('admin.groups.relationship.executeTransfer') }}
        </button>
      </div>

      <div v-if="transferResult" class="rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="font-medium text-gray-900 dark:text-white">
          {{ transferResult.dry_run ? t('admin.groups.relationship.previewResult') : t('admin.groups.relationship.transferResult') }}
        </div>
        <div class="mt-2 text-gray-600 dark:text-gray-300">
          {{ t('admin.groups.relationship.matchedCount', { count: transferResult.matched_count }) }}
          <span class="mx-2 text-gray-300 dark:text-gray-600">|</span>
          {{ t('admin.groups.relationship.updatedCount', { count: transferResult.updated_count }) }}
        </div>
        <ul v-if="transferResult.warnings?.length" class="mt-2 list-disc pl-5 text-amber-700 dark:text-amber-300">
          <li v-for="warning in transferResult.warnings" :key="warning">{{ warning }}</li>
        </ul>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="close">
          {{ t('common.close') }}
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
import type {
  BatchTransferApiKeyGroupResult
} from '@/api/admin/apiKeys'
import type {
  RelationshipGraph,
  RelationshipGraphGroup
} from '@/api/admin/groups'
import type { GroupPlatform } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupRelationshipGraphCanvas from './GroupRelationshipGraphCanvas.vue'
import {
  buildRelationshipGraphElements,
  isTransferableGroup
} from './relationshipGraphLayout'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; success: [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const transferLoading = ref(false)
const platformFilter = ref<GroupPlatform | ''>('')
const maxKeys = ref(200)
const graph = ref<RelationshipGraph>(emptyGraph())
const sourceGroupID = ref<number | ''>('')
const targetGroupID = ref<number | ''>('')
const transferResult = ref<BatchTransferApiKeyGroupResult | null>(null)

const elements = computed(() => buildRelationshipGraphElements(graph.value))
const transferGroups = computed(() => graph.value.groups.filter((group) => isTransferableGroup(group.platform)))
const targetGroups = computed(() => {
  return transferGroups.value.filter((group) => group.id !== sourceGroupID.value)
})
const canTransfer = computed(() => {
  return Number(sourceGroupID.value) > 0 && Number(targetGroupID.value) > 0 && sourceGroupID.value !== targetGroupID.value
})
const summaryText = computed(() => {
  return t('admin.groups.relationship.summary', {
    groups: graph.value.groups.length,
    keys: graph.value.api_keys.length,
    accounts: graph.value.accounts.length
  })
})

watch(
  () => props.show,
  (visible) => {
    if (visible) loadGraph()
  }
)

watch(platformFilter, () => {
  if (props.show) loadGraph()
})

async function loadGraph(): Promise<void> {
  loading.value = true
  transferResult.value = null
  try {
    graph.value = await adminAPI.groups.getRelationshipGraph({
      platform: platformFilter.value || undefined,
      include_api_keys: true,
      include_accounts: true,
      max_api_keys_per_group: Number(maxKeys.value) || 200
    })
    sourceGroupID.value = ''
    targetGroupID.value = ''
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error.message || t('admin.groups.relationship.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function previewTransfer(): Promise<void> {
  await transfer(true)
}

async function executeTransfer(): Promise<void> {
  await transfer(false)
  await loadGraph()
  emit('success')
}

async function transfer(dryRun: boolean): Promise<void> {
  if (!canTransfer.value) return
  transferLoading.value = true
  try {
    transferResult.value = await adminAPI.apiKeys.batchTransferApiKeyGroup({
      source_group_id: Number(sourceGroupID.value),
      target_group_id: Number(targetGroupID.value),
      dry_run: dryRun
    })
    if (!dryRun) appStore.showSuccess(t('admin.groups.relationship.transferSaved'))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error.message || t('admin.groups.relationship.transferFailed'))
  } finally {
    transferLoading.value = false
  }
}

function close(): void {
  emit('close')
}

function emptyGraph(): RelationshipGraph {
  return {
    groups: [] as RelationshipGraphGroup[],
    api_keys: [],
    accounts: [],
    edges: []
  }
}
</script>
