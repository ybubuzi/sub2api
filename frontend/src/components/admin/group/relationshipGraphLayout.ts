import type { RelationshipGraph } from '@/api/admin/groups'

export type RelationshipNodeType = 'user' | 'api_key' | 'group' | 'account'

export interface RelationshipNode {
  id: string
  label: string
  subtitle: string
  type: RelationshipNodeType
  status: string
  platform: string
  x: number
  y: number
  width: number
  height: number
}

export interface RelationshipLine {
  id: string
  from: string
  to: string
  type: string
}

export interface RelationshipGraphElements {
  nodes: RelationshipNode[]
  lines: RelationshipLine[]
  width: number
  height: number
}

const NODE_WIDTH = 170
const NODE_HEIGHT = 62
const COLUMN_GAP = 250
const ROW_GAP = 96
const PADDING_X = 48
const PADDING_Y = 36

const columnX = (index: number) => PADDING_X + index * COLUMN_GAP

const nodeY = (index: number) => PADDING_Y + index * ROW_GAP

const userNodeID = (userID: number) => `user:${userID}`

const graphNodeID = (prefix: string, id: number) => `${prefix}:${id}`

export const isTransferableGroup = (platform: string): boolean => {
  return platform === 'openai' || platform === 'anthropic'
}

export function buildRelationshipGraphElements(
  graph: RelationshipGraph
): RelationshipGraphElements {
  const nodes = new Map<string, RelationshipNode>()
  const lines: RelationshipLine[] = []
  const apiKeys = [...graph.api_keys].sort((a, b) => a.id - b.id)
  const groups = [...graph.groups].sort((a, b) => a.id - b.id)
  const accounts = [...graph.accounts].sort((a, b) => a.id - b.id)

  buildUserNodes(nodes, apiKeys)
  buildAPIKeyNodes(nodes, apiKeys)
  buildGroupNodes(nodes, groups)
  buildAccountNodes(nodes, accounts)
  buildUserKeyLines(lines, apiKeys)
  buildBackendLines(lines, graph)

  return {
    nodes: [...nodes.values()],
    lines,
    width: columnX(3) + NODE_WIDTH + PADDING_X,
    height: computeGraphHeight(nodes.size)
  }
}

function buildUserNodes(
  nodes: Map<string, RelationshipNode>,
  apiKeys: RelationshipGraph['api_keys']
): void {
  const userIDs = [...new Set(apiKeys.map((key) => key.user_id))].sort((a, b) => a - b)
  userIDs.forEach((userID, index) => {
    nodes.set(userNodeID(userID), {
      id: userNodeID(userID),
      label: `User #${userID}`,
      subtitle: 'API Key owner',
      type: 'user',
      status: '',
      platform: '',
      x: columnX(0),
      y: nodeY(index),
      width: NODE_WIDTH,
      height: NODE_HEIGHT
    })
  })
}

function buildAPIKeyNodes(
  nodes: Map<string, RelationshipNode>,
  apiKeys: RelationshipGraph['api_keys']
): void {
  apiKeys.forEach((key, index) => {
    nodes.set(graphNodeID('api_key', key.id), {
      id: graphNodeID('api_key', key.id),
      label: key.name || `API Key #${key.id}`,
      subtitle: `#${key.id} · user #${key.user_id}`,
      type: 'api_key',
      status: key.status,
      platform: '',
      x: columnX(1),
      y: nodeY(index),
      width: NODE_WIDTH,
      height: NODE_HEIGHT
    })
  })
}

function buildGroupNodes(
  nodes: Map<string, RelationshipNode>,
  groups: RelationshipGraph['groups']
): void {
  groups.forEach((group, index) => {
    nodes.set(graphNodeID('group', group.id), {
      id: graphNodeID('group', group.id),
      label: group.name || `Group #${group.id}`,
      subtitle: `${group.platform} · #${group.id}`,
      type: 'group',
      status: group.status,
      platform: group.platform,
      x: columnX(2),
      y: nodeY(index),
      width: NODE_WIDTH,
      height: NODE_HEIGHT
    })
  })
}

function buildAccountNodes(
  nodes: Map<string, RelationshipNode>,
  accounts: RelationshipGraph['accounts']
): void {
  accounts.forEach((account, index) => {
    nodes.set(graphNodeID('account', account.id), {
      id: graphNodeID('account', account.id),
      label: account.name || `Account #${account.id}`,
      subtitle: `${account.platform} · ${account.type}`,
      type: 'account',
      status: account.status,
      platform: account.platform,
      x: columnX(3),
      y: nodeY(index),
      width: NODE_WIDTH,
      height: NODE_HEIGHT
    })
  })
}

function buildUserKeyLines(
  lines: RelationshipLine[],
  apiKeys: RelationshipGraph['api_keys']
): void {
  apiKeys.forEach((key) => {
    lines.push({
      id: `user:${key.user_id}->api_key:${key.id}`,
      from: userNodeID(key.user_id),
      to: graphNodeID('api_key', key.id),
      type: 'owns_key'
    })
  })
}

function buildBackendLines(lines: RelationshipLine[], graph: RelationshipGraph): void {
  graph.edges.forEach((edge, index) => {
    lines.push({
      id: `${edge.from}->${edge.to}:${edge.type}:${index}`,
      from: edge.from,
      to: edge.to,
      type: edge.type
    })
  })
}

function computeGraphHeight(nodeCount: number): number {
  const minHeight = 520
  return Math.max(minHeight, PADDING_Y * 2 + Math.max(1, nodeCount) * ROW_GAP)
}
