/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId
  })
  return data
}

export interface BatchTransferApiKeyGroupResult {
  source_group_id: number
  target_group_id: number
  dry_run: boolean
  matched_count: number
  updated_count: number
  warnings?: string[]
}

export async function batchTransferApiKeyGroup(input: {
  source_group_id: number
  target_group_id: number
  dry_run?: boolean
}): Promise<BatchTransferApiKeyGroupResult> {
  const { data } = await apiClient.post<BatchTransferApiKeyGroupResult>(
    '/admin/api-keys/batch-transfer-group',
    input
  )
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  batchTransferApiKeyGroup
}

export default apiKeysAPI
