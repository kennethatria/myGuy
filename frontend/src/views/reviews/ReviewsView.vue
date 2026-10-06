<template>
  <div class="container py-4">
    <section class="network-card" aria-labelledby="network-title">
      <div class="network-header">
        <h1 id="network-title" class="network-title">Your network</h1>
        <p v-if="connections.length" class="network-summary">
          {{ connections.length }} {{ connections.length === 1 ? 'person' : 'people' }} you've reviewed or who reviewed you
        </p>
      </div>

      <p v-if="loading" class="network-note">Loading your network...</p>
      <p v-else-if="failed" class="network-note">
        Couldn't load your reviews.
        <button type="button" class="link-button" @click="load">Try again</button>
      </p>
      <p v-else-if="!connections.length" class="network-note">
        No reviews yet. When a gig or a sale is done, you and the other person can review each other, and they'll show up here.
      </p>

      <div v-else class="network-layout">
        <div class="network-stage">
          <svg
            class="network"
            :viewBox="`0 0 ${WIDTH} ${height}`"
            role="group"
            :aria-label="`Your network: ${connections.length} people`"
          >
            <!-- Lines first, so nodes sit on top of their ends -->
            <g class="edges" aria-hidden="true">
              <line
                v-for="node in placed"
                :key="`edge-${node.connection.userId}`"
                :x1="node.from.x"
                :y1="node.from.y"
                :x2="node.x"
                :y2="node.y"
                :class="['edge', { active: selected?.userId === node.connection.userId }]"
              />
            </g>

            <g class="you" aria-hidden="true">
              <circle :cx="YOU.x" :cy="YOU.y" :r="NODE_RADIUS" class="you-dot" />
              <text :x="YOU.x" :y="YOU.y + 1.6" text-anchor="middle" class="you-label">You</text>
            </g>

            <!-- The average rating between you, on each line -->
            <g class="ratings" aria-hidden="true">
              <g v-for="node in placed" :key="`rating-${node.connection.userId}`">
                <rect :x="node.label.x - 7" :y="node.label.y - 3.4" width="14" height="6.8" rx="3.4" class="rating-pill" />
                <text :x="node.label.x" :y="node.label.y + 1.6" text-anchor="middle" class="rating-text">
                  ★{{ formatRating(node.connection.averageRating) }}
                </text>
              </g>
            </g>

            <g
              v-for="node in placed"
              :key="node.connection.userId"
              :class="['node', { active: selected?.userId === node.connection.userId }]"
              role="button"
              tabindex="0"
              :aria-label="nodeLabel(node.connection)"
              :aria-expanded="selected?.userId === node.connection.userId"
              @click.stop="toggle(node.connection)"
              @keydown.enter.prevent="toggle(node.connection)"
              @keydown.space.prevent="toggle(node.connection)"
            >
              <title>{{ nameOf(node.connection.userId) }}</title>
              <!-- A larger, invisible circle makes the node easier to tap -->
              <circle :cx="node.x" :cy="node.y" :r="NODE_RADIUS + 3" class="node-hit" />
              <circle
                :cx="node.x"
                :cy="node.y"
                :r="NODE_RADIUS"
                :fill="VIA_STYLE[node.connection.via].fill"
                :stroke="VIA_STYLE[node.connection.via].stroke"
                stroke-width="0.6"
              />
              <text
                :x="node.x"
                :y="node.y + 1.9"
                text-anchor="middle"
                class="node-initial"
                :fill="VIA_STYLE[node.connection.via].text"
              >{{ initialOf(node.connection.userId) }}</text>
              <text :x="node.x" :y="node.y + NODE_RADIUS + 5.5" text-anchor="middle" class="node-name">
                {{ shortName(node.connection.userId) }}
              </text>
            </g>
          </svg>

          <!-- Summary of the chosen person, next to their dot -->
          <div
            v-if="selected && selectedSpot"
            class="node-popup"
            :class="{ left: selectedSpot.x > YOU.x, above: selectedSpot.y > height / 2 }"
            :style="{ left: `${selectedSpot.x}%`, top: `${(selectedSpot.y / height) * 100}%` }"
            role="dialog"
            :aria-label="`About ${nameOf(selected.userId)}`"
            ref="popup"
            @click.stop
          >
            <div class="popup-header">
              <router-link :to="{ name: 'user-profile', params: { id: selected.userId } }" class="popup-name">
                {{ nameOf(selected.userId) }}
              </router-link>
              <button type="button" class="popup-close" aria-label="Close" @click="selected = null">×</button>
            </div>
            <p class="popup-meta">
              {{ selected.deals }} {{ selected.deals === 1 ? 'deal' : 'deals' }} together
              · {{ viaLabel(selected.via) }}
            </p>
            <dl class="popup-ratings">
              <div>
                <dt>Rated you</dt>
                <dd>{{ selected.ratingOfYou !== null ? `★ ${selected.ratingOfYou}` : 'Not yet' }}</dd>
              </div>
              <div>
                <dt>You rated</dt>
                <dd>{{ selected.yourRating !== null ? `★ ${selected.yourRating}` : 'Not yet' }}</dd>
              </div>
            </dl>
            <ul class="popup-list">
              <li v-for="item in selected.interactions.slice(0, 3)" :key="`${item.deal}-${item.direction}`">
                <router-link :to="linkTo(item)" class="popup-deal">{{ item.via === 'gig' ? 'Gig' : 'Item' }}: {{ item.title }}</router-link>
                <span class="popup-review">
                  {{ item.direction === 'given' ? 'You' : 'They' }} gave ★ {{ item.rating }}<template v-if="item.comment">: “{{ item.comment }}”</template>
                </span>
              </li>
            </ul>
            <p v-if="selected.interactions.length > 3" class="popup-more">
              and {{ selected.interactions.length - 3 }} more
            </p>
          </div>
        </div>

        <div class="network-side">
          <ul class="legend" aria-label="Key">
            <li v-for="via in VIAS" :key="via">
              <span class="legend-dot" :style="{ background: VIA_STYLE[via].fill, borderColor: VIA_STYLE[via].stroke }" aria-hidden="true"></span>
              {{ viaLabel(via) }}
            </li>
          </ul>
          <p class="network-note">Each line shows the average rating between you, both ways. Most deals first. Tap someone to see what passed between you.</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { useReviewsStore } from '@/stores/reviews'
import { useUserStore } from '@/stores/user'
import { KIND_STYLE } from '@/utils/radar'
import { connectionsFrom, layoutNetwork, formatRating, WIDTH, YOU, NODE_RADIUS, type Connection, type Interaction, type Via } from '@/utils/network'

const reviewsStore = useReviewsStore()
const userStore = useUserStore()

// Gigs and marketplace items keep the colours they have on the home radar
const VIA_STYLE: Record<Via | 'both', { fill: string; stroke: string; text: string; label: string }> = {
  gig: { fill: KIND_STYLE.task.fill, stroke: KIND_STYLE.task.stroke, text: '#fff', label: 'Gigs' },
  item: { fill: KIND_STYLE.item.fill, stroke: KIND_STYLE.item.stroke, text: '#422006', label: 'Marketplace' },
  both: { fill: '#16a34a', stroke: '#14532d', text: '#fff', label: 'Gigs and marketplace' }
}
const VIAS = ['gig', 'item', 'both'] as const

const connections = ref<Connection[]>([])
const loading = ref(true)
const failed = ref(false)
const selected = ref<Connection | null>(null)

const layout = computed(() => layoutNetwork(connections.value))
const placed = computed(() => layout.value.placed)
const height = computed(() => layout.value.height)
const selectedSpot = computed(() => placed.value.find(n => n.connection.userId === selected.value?.userId))

function nameOf(userId: number): string {
  const user = userStore.getUserById(userId)
  return user?.name || user?.username || `User ${userId}`
}

// Names under nodes stay short enough not to run into the next column
function shortName(userId: number): string {
  const name = nameOf(userId)
  return name.length > 12 ? `${name.slice(0, 11)}…` : name
}

function initialOf(userId: number): string {
  return nameOf(userId).charAt(0).toUpperCase()
}

function viaLabel(via: Via | 'both'): string {
  return VIA_STYLE[via].label
}

function nodeLabel(connection: Connection): string {
  return `${nameOf(connection.userId)}, ${connection.deals} ${connection.deals === 1 ? 'deal' : 'deals'} together`
}

function linkTo(item: Interaction): RouteLocationRaw {
  return item.via === 'gig'
    ? { name: 'task-detail', params: { id: item.linkId } }
    : { name: 'store-item', params: { id: item.linkId } }
}

const popup = ref<HTMLElement | null>(null)

async function toggle(connection: Connection) {
  selected.value = selected.value?.userId === connection.userId ? null : connection
  // On phones the summary sits under the graph: bring it into view
  await nextTick()
  popup.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
}

async function load() {
  loading.value = true
  failed.value = false
  try {
    connections.value = connectionsFrom(await reviewsStore.fetchMyInteractions())
  } catch (err) {
    console.error('Failed to load network:', err)
    failed.value = true
  } finally {
    loading.value = false
  }
}

// Clicking elsewhere or pressing Escape closes the summary
function close() {
  selected.value = null
}
function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

onMounted(() => {
  load()
  document.addEventListener('click', close)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', close)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.network-card {
  padding: 1rem;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 0.75rem;
  background: #fff;
}

.network-header {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.25rem 1rem;
  margin-bottom: 0.75rem;
}

.network-title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.network-summary,
.network-note {
  margin: 0;
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.network-layout {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem 1.5rem;
  align-items: flex-start;
}

.network-stage {
  position: relative;
  flex: 1 1 320px;
  max-width: 560px;
}

.network {
  display: block;
  width: 100%;
  height: auto;
}

.edge {
  stroke: #cbd5e1;
  stroke-width: 0.6;
}

.rating-pill {
  fill: #fff;
  stroke: #e5e7eb;
  stroke-width: 0.3;
}

.rating-text {
  font-size: 4.2px;
  font-weight: 700;
  fill: #b45309;
}

.node-name {
  font-size: 4.4px;
  fill: #374151;
  pointer-events: none;
}

.edge.active {
  stroke: var(--color-primary, #4f46e5);
}

.you-dot {
  fill: var(--color-primary, #4f46e5);
}

.you-label {
  font-size: 4.4px;
  font-weight: 700;
  fill: #fff;
}

.node {
  cursor: pointer;
  outline: none;
}

.node-hit {
  fill: transparent;
}

.node-initial {
  font-size: 5.4px;
  font-weight: 700;
  pointer-events: none;
}

.node:hover circle:not(.node-hit),
.node:focus-visible circle:not(.node-hit),
.node.active circle:not(.node-hit) {
  stroke: var(--color-primary, #4f46e5);
  stroke-width: 0.9;
}

.node-popup {
  position: absolute;
  z-index: 10;
  width: min(260px, 80vw);
  margin: 0.75rem 0 0 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 0.5rem;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.15);
  font-size: 0.85rem;
}

.node-popup.left {
  margin-left: 0;
  transform: translateX(calc(-100% - 0.75rem));
}

.node-popup.above {
  margin-top: 0;
  transform: translateY(calc(-100% - 0.75rem));
}

.node-popup.left.above {
  transform: translate(calc(-100% - 0.75rem), calc(-100% - 0.75rem));
}

.popup-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.popup-name {
  font-weight: 600;
  color: var(--color-primary, #4f46e5);
}

.popup-close {
  border: none;
  background: none;
  font-size: 1.2rem;
  line-height: 1;
  cursor: pointer;
  color: var(--color-text-light, #6b7280);
}

.popup-meta {
  margin: 0.15rem 0 0.5rem;
  color: var(--color-text-light, #6b7280);
}

.popup-ratings {
  display: flex;
  gap: 1rem;
  margin: 0 0 0.5rem;
}

.popup-ratings dt {
  font-size: 0.75rem;
  color: var(--color-text-light, #6b7280);
}

.popup-ratings dd {
  margin: 0;
  font-weight: 600;
}

.popup-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.popup-deal {
  display: block;
  font-weight: 500;
  color: inherit;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.popup-review {
  color: var(--color-text-light, #6b7280);
  overflow-wrap: anywhere;
}

.popup-more {
  margin: 0.4rem 0 0;
  color: var(--color-text-light, #6b7280);
}

.network-side {
  flex: 1 1 200px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.legend {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 0.9rem;
}

.legend-dot {
  display: inline-block;
  width: 0.8rem;
  height: 0.8rem;
  margin-right: 0.4rem;
  border: 1px solid;
  border-radius: 50%;
  vertical-align: -0.1rem;
}

.link-button {
  border: none;
  background: none;
  padding: 0;
  color: var(--color-primary, #4f46e5);
  text-decoration: underline;
  cursor: pointer;
}

/* Phones: the graph takes the full width, and the summary goes under it
   rather than beside a dot, where it could run off the screen */
@media (max-width: 640px) {
  .network-stage {
    flex-basis: 100%;
  }

  .node-popup,
  .node-popup.left,
  .node-popup.above,
  .node-popup.left.above {
    position: static;
    width: auto;
    margin: 0.75rem 0 0;
    transform: none;
    /* Scrolled into view clear of the floating chat button */
    scroll-margin-bottom: 6rem;
  }

  .popup-close {
    min-width: 44px;
    min-height: 44px;
  }
}
</style>
