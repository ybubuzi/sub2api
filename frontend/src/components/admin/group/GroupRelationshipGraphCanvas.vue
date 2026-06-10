<template>
  <div class="h-[560px] overflow-auto rounded-lg border border-gray-200 bg-slate-50 dark:border-dark-600 dark:bg-dark-900">
    <svg
      :width="elements.width"
      :height="elements.height"
      class="min-h-full min-w-full"
      @pointermove="dragMove"
      @pointerup="dragEnd"
      @pointerleave="dragEnd"
    >
      <g class="graph-lines">
        <path
          v-for="line in renderedLines"
          :key="line.id"
          :d="line.path"
          :class="['relationship-line', line.type]"
        />
      </g>

      <g
        v-for="node in nodes"
        :key="node.id"
        :transform="`translate(${node.x}, ${node.y})`"
        class="cursor-grab active:cursor-grabbing"
        @pointerdown="dragStart($event, node.id)"
      >
        <rect
          :width="node.width"
          :height="node.height"
          rx="8"
          :class="nodeClass(node)"
        />
        <circle cx="16" cy="18" r="4" :class="dotClass(node)" />
        <text x="28" y="22" class="node-title">
          {{ truncate(node.label, 24) }}
        </text>
        <text x="16" y="44" class="node-subtitle">
          {{ truncate(node.subtitle, 30) }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type {
  RelationshipGraphElements,
  RelationshipNode,
  RelationshipLine
} from './relationshipGraphLayout'

interface RenderedLine extends RelationshipLine {
  path: string
}

const props = defineProps<{
  elements: RelationshipGraphElements
}>()

const nodes = ref<RelationshipNode[]>([])
const dragging = ref<{
  id: string
  offsetX: number
  offsetY: number
} | null>(null)

watch(
  () => props.elements,
  (value) => {
    nodes.value = value.nodes.map((node) => ({ ...node }))
  },
  { immediate: true }
)

const nodeMap = computed(() => {
  return new Map(nodes.value.map((node) => [node.id, node]))
})

const renderedLines = computed<RenderedLine[]>(() => {
  return props.elements.lines
    .map((line) => {
      const from = nodeMap.value.get(line.from)
      const to = nodeMap.value.get(line.to)
      if (!from || !to) return null
      return { ...line, path: buildPath(from, to) }
    })
    .filter((line): line is RenderedLine => Boolean(line))
})

function dragStart(event: PointerEvent, id: string): void {
  const node = nodeMap.value.get(id)
  if (!node) return
  dragging.value = {
    id,
    offsetX: event.offsetX - node.x,
    offsetY: event.offsetY - node.y
  }
  ;(event.currentTarget as SVGElement).setPointerCapture(event.pointerId)
}

function dragMove(event: PointerEvent): void {
  if (!dragging.value) return
  const target = nodes.value.find((node) => node.id === dragging.value?.id)
  if (!target) return
  target.x = Math.max(8, event.offsetX - dragging.value.offsetX)
  target.y = Math.max(8, event.offsetY - dragging.value.offsetY)
}

function dragEnd(): void {
  dragging.value = null
}

function buildPath(from: RelationshipNode, to: RelationshipNode): string {
  const startX = from.x + from.width
  const startY = from.y + from.height / 2
  const endX = to.x
  const endY = to.y + to.height / 2
  const curve = Math.max(60, (endX - startX) / 2)
  return `M ${startX} ${startY} C ${startX + curve} ${startY}, ${endX - curve} ${endY}, ${endX} ${endY}`
}

function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) return value
  return `${value.slice(0, maxLength - 1)}...`
}

function nodeClass(node: RelationshipNode): string[] {
  return [
    'node-box',
    `node-${node.type}`,
    node.status === 'inactive' ? 'node-inactive' : ''
  ]
}

function dotClass(node: RelationshipNode): string[] {
  return ['node-dot', node.status === 'active' ? 'node-dot-active' : '']
}
</script>

<style scoped>
.relationship-line {
  fill: none;
  stroke: #94a3b8;
  stroke-width: 1.6;
  opacity: 0.55;
}

.relationship-line.fallback_group,
.relationship-line.invalid_request_fallback_group {
  stroke: #f59e0b;
  stroke-dasharray: 6 6;
}

.node-box {
  fill: #ffffff;
  stroke: #cbd5e1;
  stroke-width: 1;
  filter: drop-shadow(0 4px 10px rgb(15 23 42 / 0.08));
}

.node-user {
  stroke: #6366f1;
}

.node-api_key {
  stroke: #0ea5e9;
}

.node-group {
  stroke: #10b981;
}

.node-account {
  stroke: #f97316;
}

.node-inactive {
  opacity: 0.55;
}

.node-dot {
  fill: #94a3b8;
}

.node-dot-active {
  fill: #22c55e;
}

.node-title {
  fill: #111827;
  font-size: 13px;
  font-weight: 600;
  pointer-events: none;
}

.node-subtitle {
  fill: #64748b;
  font-size: 11px;
  pointer-events: none;
}

:global(.dark) .node-box {
  fill: #111827;
  stroke: #475569;
}

:global(.dark) .node-title {
  fill: #f8fafc;
}

:global(.dark) .node-subtitle {
  fill: #94a3b8;
}
</style>
