import { describe, expect, it, vi, beforeEach } from 'vitest'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import { useGroupMirrorModelCandidates } from '../useGroupMirrorModelCandidates'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      getModelsListCandidates: vi.fn()
    }
  }
}))

const sourceGroup: AdminGroup = {
  id: 10,
  name: 'OpenAI A',
  description: null,
  platform: 'openai',
  rate_multiplier: 1,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  kiro_cache_emulation_enabled: false,
  kiro_cache_emulation_ratio: 1,
  created_at: '',
  updated_at: '',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: false,
  sort_order: 0
}

describe('useGroupMirrorModelCandidates', () => {
  beforeEach(() => {
    vi.mocked(adminAPI.groups.getModelsListCandidates).mockReset()
  })

  it('loads target platform and source platform candidates from the source group', async () => {
    vi.mocked(adminAPI.groups.getModelsListCandidates)
      .mockResolvedValueOnce(['claude-sonnet-4-6'])
      .mockResolvedValueOnce(['gpt-5.5', 'gpt-5.4'])
    const candidates = useGroupMirrorModelCandidates({
      errorMessage: () => 'failed',
      onError: vi.fn()
    })

    await candidates.load(sourceGroup, true)

    expect(adminAPI.groups.getModelsListCandidates).toHaveBeenNthCalledWith(1, 10, 'anthropic')
    expect(adminAPI.groups.getModelsListCandidates).toHaveBeenNthCalledWith(2, 10, 'openai')
    expect(candidates.clientModels.value).toEqual(['claude-sonnet-4-6'])
    expect(candidates.sourceModels.value).toEqual(['gpt-5.5', 'gpt-5.4'])
  })

  it('does not load candidates when the modal is hidden', async () => {
    const candidates = useGroupMirrorModelCandidates({
      errorMessage: () => 'failed',
      onError: vi.fn()
    })

    await candidates.load(sourceGroup, false)

    expect(adminAPI.groups.getModelsListCandidates).not.toHaveBeenCalled()
    expect(candidates.clientModels.value).toEqual([])
    expect(candidates.sourceModels.value).toEqual([])
  })
})
