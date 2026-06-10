import { describe, expect, it } from 'vitest'
import type { AdminGroup } from '@/types'
import {
  buildMirrorCandidateContext,
  mergeMirrorModelCandidates,
  normalizeMirrorModelCandidates
} from '../groupMirrorModels'

const baseGroup: AdminGroup = {
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

describe('groupMirrorModels', () => {
  it('builds source group candidate context from group platform', () => {
    expect(buildMirrorCandidateContext(baseGroup)).toEqual({
      sourceGroupID: 10,
      sourcePlatform: 'openai',
      targetPlatform: 'anthropic'
    })
  })

  it('builds mirror group candidate context from mirror source metadata', () => {
    expect(buildMirrorCandidateContext({
      ...baseGroup,
      id: 11,
      platform: 'anthropic',
      is_mirror: true,
      mirror_source_group_id: 10,
      mirror_source_platform: 'openai'
    })).toEqual({
      sourceGroupID: 10,
      sourcePlatform: 'openai',
      targetPlatform: 'anthropic'
    })
  })

  it('returns null for unsupported platforms', () => {
    expect(buildMirrorCandidateContext({ ...baseGroup, platform: 'gemini' })).toBeNull()
  })

  it('normalizes and deduplicates candidate models while preserving order', () => {
    expect(normalizeMirrorModelCandidates([' gpt-5.5 ', '', 'gpt-5.5', 'gpt-5.4'])).toEqual([
      'gpt-5.5',
      'gpt-5.4'
    ])
  })

  it('merges loaded candidates with existing manual values', () => {
    expect(mergeMirrorModelCandidates({
      primary: ['claude-sonnet-4-6'],
      secondary: ['qwen3.6-plus'],
      existing: ['manual-model', ' qwen3.6-plus ']
    })).toEqual(['claude-sonnet-4-6', 'qwen3.6-plus', 'manual-model'])
  })
})
