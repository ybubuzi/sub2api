import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import {
  buildMirrorCandidateContext,
  normalizeMirrorModelCandidates
} from './groupMirrorModels'

export function useGroupMirrorModelCandidates(options: {
  errorMessage: () => string
  onError: (message: string) => void
}) {
  const loading = ref(false)
  const error = ref('')
  const clientModels = ref<string[]>([])
  const sourceModels = ref<string[]>([])
  let requestID = 0

  function clear(): void {
    requestID += 1
    loading.value = false
    clientModels.value = []
    sourceModels.value = []
  }

  async function load(group: AdminGroup | null, show: boolean): Promise<void> {
    const context = buildMirrorCandidateContext(group)
    clear()
    error.value = ''
    if (!show || !context) return

    const currentID = ++requestID
    loading.value = true
    try {
      const [clientCandidates, sourceCandidates] = await Promise.all([
        adminAPI.groups.getModelsListCandidates(context.sourceGroupID, context.targetPlatform),
        adminAPI.groups.getModelsListCandidates(context.sourceGroupID, context.sourcePlatform)
      ])
      if (currentID !== requestID) return
      clientModels.value = normalizeMirrorModelCandidates(clientCandidates)
      sourceModels.value = normalizeMirrorModelCandidates(sourceCandidates)
    } catch (err: any) {
      if (currentID !== requestID) return
      error.value = err.response?.data?.detail || err.message || options.errorMessage()
      options.onError(error.value)
    } finally {
      if (currentID === requestID) loading.value = false
    }
  }

  return {
    loading,
    error,
    clientModels,
    sourceModels,
    clear,
    load
  }
}
