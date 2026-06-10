import type { AdminGroup, GroupPlatform } from '@/types'

export type MirrorGroupPlatform = Extract<GroupPlatform, 'openai' | 'anthropic'>

export interface MirrorCandidateContext {
  sourceGroupID: number
  sourcePlatform: MirrorGroupPlatform
  targetPlatform: MirrorGroupPlatform
}

export function buildMirrorCandidateContext(group: AdminGroup | null): MirrorCandidateContext | null {
  if (!group || !isMirrorPlatform(group.platform)) {
    return null
  }
  if (group.is_mirror || group.mirror_source_group_id) {
    const sourcePlatform = group.mirror_source_platform
    if (!group.mirror_source_group_id || !isMirrorPlatform(sourcePlatform)) {
      return null
    }
    return {
      sourceGroupID: group.mirror_source_group_id,
      sourcePlatform,
      targetPlatform: group.platform
    }
  }
  return {
    sourceGroupID: group.id,
    sourcePlatform: group.platform,
    targetPlatform: oppositeMirrorPlatform(group.platform)
  }
}

export function mergeMirrorModelCandidates(input: {
  primary: string[]
  secondary?: string[]
  existing?: string[]
}): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const model of [...input.primary, ...(input.secondary ?? []), ...(input.existing ?? [])]) {
    const normalized = model.trim()
    if (!normalized || seen.has(normalized)) continue
    seen.add(normalized)
    out.push(normalized)
  }
  return out
}

export function normalizeMirrorModelCandidates(models: string[]): string[] {
  return mergeMirrorModelCandidates({ primary: models })
}

function isMirrorPlatform(platform: GroupPlatform | '' | undefined): platform is MirrorGroupPlatform {
  return platform === 'openai' || platform === 'anthropic'
}

function oppositeMirrorPlatform(platform: MirrorGroupPlatform): MirrorGroupPlatform {
  return platform === 'openai' ? 'anthropic' : 'openai'
}
