<template>
  <div class="container py-4">
    <section class="network-card" aria-labelledby="network-title">
      <div class="network-header">
        <h1 id="network-title" class="network-title">{{ isMine ? 'Explore network' : `${nameOf(centreId)}'s network` }}</h1>
        <p v-if="isMine" class="network-intro">
          Everyone you've done a gig or a sale with, the rating you gave each other, and who they've worked with in turn.
          Tap someone to see what others say about them.
        </p>
        <p v-if="branches.length" class="network-summary">
          {{ connectionCount }} {{ connectionCount === 1 ? 'person' : 'people' }}
          {{ isMine ? "you've reviewed or who reviewed you" : 'they reviewed or who reviewed them' }},
          and who they're connected to{{ trimmed ? `. Showing the ${MAX_PEOPLE} most recent.` : '' }}
        </p>
        <router-link v-if="!isMine" :to="{ name: 'reviews' }" class="back-to-mine">Back to your network</router-link>
      </div>

      <p v-if="loading" class="network-note">Loading...</p>
      <p v-else-if="failed" class="network-note">
        Couldn't load the reviews.
        <button type="button" class="link-button" @click="load">Try again</button>
      </p>
      <p v-else-if="!branches.length" class="network-note">
        {{ isMine
          ? "No reviews yet. When a gig or a sale is done, you and the other person can review each other, and they'll show up here."
          : 'No reviews yet.' }}
      </p>

      <div v-else class="network-stage">
        <svg
          class="network"
          :viewBox="`0 0 ${WIDTH} ${layout.height}`"
          role="group"
          :aria-label="`${isMine ? 'Your' : `${nameOf(centreId)}'s`} network: ${branches.length} people`"
        >
          <!-- The branches: a smooth curve down from each person and round to
               whom they're connected to -->
          <g class="branches" aria-hidden="true">
            <path
              v-for="node in placed"
              :key="`branch-${keyOf(node)}`"
              :d="branchPath(node)"
              :class="['branch', { active: selectedKey === keyOf(node) }]"
            />
          </g>

          <!-- The average rating between the two, both ways, on each branch -->
          <g class="ratings" aria-hidden="true">
            <g v-for="node in placed" :key="`rating-${keyOf(node)}`">
              <rect :x="node.label.x - 6" :y="node.label.y - 3.1" width="12" height="6.2" rx="3.1" class="rating-pill" />
              <text :x="node.label.x" :y="node.label.y + 1.3" text-anchor="middle" class="rating-text">
                ★{{ formatRating(node.connection.averageRating) }}
              </text>
            </g>
          </g>

          <g class="you" aria-hidden="true">
            <circle :cx="YOU.x" :cy="YOU.y" :r="NODE_RADIUS" class="you-dot" />
            <text :x="YOU.x" :y="YOU.y + 1.4" text-anchor="middle" class="you-label">{{ isMine ? 'You' : initialOf(centreId) }}</text>
            <text v-if="!isMine" :x="YOU.x + NODE_RADIUS + 2" :y="YOU.y + 1.4" class="node-name">{{ shortName(centreId) }}</text>
          </g>

          <g
            v-for="node in placed"
            :key="keyOf(node)"
            :class="['node', `level-${node.level}`, { active: selectedKey === keyOf(node) }]"
            role="button"
            tabindex="0"
            :aria-label="nodeLabel(node)"
            :aria-expanded="selectedKey === keyOf(node)"
            @click.stop="toggle(node)"
            @keydown.enter.prevent="toggle(node)"
            @keydown.space.prevent="toggle(node)"
          >
            <title>{{ nameOf(node.connection.userId) }}</title>
            <!-- A larger, invisible circle makes the node easier to tap -->
            <circle :cx="node.x" :cy="node.y" :r="node.r + 3" class="node-hit" />
            <circle
              :cx="node.x"
              :cy="node.y"
              :r="node.r"
              :fill="VIA_STYLE[node.connection.via].fill"
              :stroke="VIA_STYLE[node.connection.via].stroke"
              stroke-width="0.5"
            />
            <text :x="node.x" :y="node.y + 1.4" text-anchor="middle" class="node-initial" :fill="VIA_STYLE[node.connection.via].text">
              {{ initialOf(node.connection.userId) }}
            </text>
            <text :x="node.x + node.r + 2" :y="node.y + 1.4" class="node-name">{{ shortName(node.connection.userId) }}</text>
          </g>
        </svg>

        <!-- A small card about the tapped person, under their node; it goes
             away by itself after a few seconds -->
        <div
          v-if="selected"
          ref="popup"
          class="node-popup"
          :style="{ left: `min(${selected.x}%, calc(100% - 13rem))`, top: `${((selected.y + selected.r + 2) / layout.height) * 100}%` }"
          role="dialog"
          :aria-label="`About ${nameOf(selected.connection.userId)}`"
          @click.stop
        >
          <router-link :to="{ name: 'user-profile', params: { id: selected.connection.userId } }" class="popup-name">
            {{ nameOf(selected.connection.userId) }}
          </router-link>
          <button type="button" class="popup-close" aria-label="Close" @click="close">×</button>
          <p class="popup-meta">
            ★ {{ formatRating(selected.connection.averageRating) }} ·
            {{ selected.connection.deals }} {{ selected.connection.deals === 1 ? 'deal' : 'deals' }}
          </p>
          <!-- What others said about them lately (public on their profile) -->
          <ul v-if="comments.length" class="popup-comments">
            <li v-for="comment in comments" :key="comment.id">
              <span class="comment-rating">★ {{ comment.rating }}</span>
              “{{ comment.text }}”
              <span class="comment-by">— {{ comment.by }}</span>
            </li>
          </ul>
          <p v-else-if="commentsLoading" class="popup-note">Loading comments…</p>
          <p v-else class="popup-note">No comments yet.</p>
        </div>
      </div>

      <div v-if="branches.length && !loading" class="network-side">
        <ul class="legend" aria-label="Key">
          <li v-for="via in VIAS" :key="via">
            <span class="legend-dot" :style="{ background: VIA_STYLE[via].fill, borderColor: VIA_STYLE[via].stroke }" aria-hidden="true"></span>
            {{ VIA_STYLE[via].label }}
          </li>
        </ul>
        <p class="network-note">
          Each branch shows the average rating between the two people, both ways. Under each person, the people they're connected to.
        </p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useReviewsStore } from '@/stores/reviews'
import { useUserStore } from '@/stores/user'
import { KIND_STYLE } from '@/utils/radar'
import {
  connectionsFrom, recentNetwork, layoutNetwork, MAX_PEOPLE, branchPath, formatRating, WIDTH, YOU, NODE_RADIUS,
  type Branch, type PlacedConnection, type Via
} from '@/utils/network'

const reviewsStore = useReviewsStore()
const authStore = useAuthStore()
const route = useRoute()
const userStore = useUserStore()

// Gigs and marketplace items keep the colours they have on the home radar
const VIA_STYLE: Record<Via | 'both', { fill: string; stroke: string; text: string; label: string }> = {
  gig: { fill: KIND_STYLE.task.fill, stroke: KIND_STYLE.task.stroke, text: '#fff', label: 'Gigs' },
  item: { fill: KIND_STYLE.item.fill, stroke: KIND_STYLE.item.stroke, text: '#422006', label: 'Marketplace' },
  both: { fill: '#16a34a', stroke: '#14532d', text: '#fff', label: 'Gigs and marketplace' }
}
const VIAS = ['gig', 'item', 'both'] as const

// The card closes by itself after this long
const POPUP_MS = 10_000

const branches = ref<Branch[]>([])
// Everyone the centre person is connected to, and whether the tree leaves
// some people out to stay within MAX_PEOPLE
const connectionCount = ref(0)
const trimmed = ref(false)
const loading = ref(true)
const failed = ref(false)
const selected = ref<PlacedConnection | null>(null)

// Whose network this is: yours, or someone's you followed from a profile
// (/reviews/:userId)
const me = computed(() => authStore.user?.id)
const centreId = computed(() => Number(route.params.userId) || me.value || 0)
const isMine = computed(() => centreId.value === me.value)

const layout = computed(() => layoutNetwork(branches.value, centreId.value))
const placed = computed(() => layout.value.placed)

// A person can appear under more than one branch: a node is the person in
// one place
const keyOf = (node: PlacedConnection) => `${node.parentId}-${node.connection.userId}`
const selectedKey = computed(() => (selected.value ? keyOf(selected.value) : null))

function nameOf(userId: number): string {
  const user = userStore.getUserById(userId)
  return user?.name || user?.username || `User ${userId}`
}

// Names next to nodes stay short enough to fit the width
function shortName(userId: number): string {
  const name = nameOf(userId)
  return name.length > 12 ? `${name.slice(0, 11)}…` : name
}

function initialOf(userId: number): string {
  return nameOf(userId).charAt(0).toUpperCase()
}

function nodeLabel(node: PlacedConnection): string {
  const { userId, deals, averageRating } = node.connection
  const parent = node.parentId === me.value ? 'you' : nameOf(node.parentId)
  return `${nameOf(userId)}: ${deals} ${deals === 1 ? 'deal' : 'deals'} with ${parent}, average rating ${formatRating(averageRating)}`
}

const popup = ref<HTMLElement | null>(null)
let closeTimer: ReturnType<typeof setTimeout> | undefined

// The two latest comments others left about the tapped person
interface Comment { id: number | string; rating: number; text: string; by: string }
const comments = ref<Comment[]>([])
const commentsLoading = ref(false)
const commentsCache = new Map<number, Comment[]>()

async function loadComments(userId: number) {
  const cached = commentsCache.get(userId)
  if (cached) {
    comments.value = cached
    return
  }
  comments.value = []
  commentsLoading.value = true
  try {
    const reviews = await reviewsStore.fetchAllRatings(userId)
    const latest = reviews
      .filter(r => r.comment?.trim())
      .slice(0, 2)
      .map(r => ({ id: r.id, rating: r.rating, text: r.comment.trim(), by: r.reviewer?.username || 'someone' }))
    commentsCache.set(userId, latest)
    if (selected.value?.connection.userId === userId) comments.value = latest
  } catch {
    // The card works without them
  } finally {
    commentsLoading.value = false
  }
}

function close() {
  selected.value = null
  if (closeTimer) clearTimeout(closeTimer)
}

async function toggle(node: PlacedConnection) {
  if (selectedKey.value === keyOf(node)) return close()
  selected.value = node
  if (closeTimer) clearTimeout(closeTimer)
  closeTimer = setTimeout(close, POPUP_MS)
  loadComments(node.connection.userId)
  await nextTick()
  popup.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
}

// The centre person's connections, and each of theirs (ratings only; the
// centre person is left out of their lists, being the root)
async function load() {
  loading.value = true
  failed.value = false
  close()
  try {
    const centre = centreId.value
    const all = connectionsFrom(await reviewsStore.fetchInteractions(centre))
    // Only the most recent can make the tree: no need to fetch the others'
    const firsts = [...all].sort((a, b) => b.latest.localeCompare(a.latest)).slice(0, MAX_PEOPLE)
    const children = await Promise.all(firsts.map(first =>
      reviewsStore.fetchInteractions(first.userId)
        .then(list => connectionsFrom(list).filter(child => child.userId !== centre))
        .catch(() => [])
    ))
    const full = firsts.map((connection, i) => ({ connection, children: children[i] }))
    branches.value = recentNetwork(full)
    connectionCount.value = all.length
    const shown = branches.value.reduce((sum, b) => sum + 1 + b.children.length, 0)
    trimmed.value = shown < all.length + children.reduce((sum, c) => sum + c.length, 0)
  } catch (err) {
    console.error('Failed to load network:', err)
    failed.value = true
  } finally {
    loading.value = false
  }
}

function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

// Following a link to another network reuses this page
watch(centreId, load)

onMounted(() => {
  load()
  document.addEventListener('click', close)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  close()
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
  margin-bottom: 0.75rem;
}

.network-title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.network-intro {
  margin: 0.375rem 0 0;
  font-size: 0.95rem;
  color: #374151;
}

.network-summary,
.network-note {
  margin: 0.25rem 0 0;
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.back-to-mine {
  display: inline-block;
  margin-top: 0.25rem;
  color: var(--color-primary, #4f46e5);
  font-weight: 600;
  font-size: 0.875rem;
}

.network-stage {
  position: relative;
  max-width: 560px;
}

.network {
  display: block;
  width: 100%;
  height: auto;
}

/* Branches: one smooth, solid curve each */
.branch {
  fill: none;
  stroke: #94a3b8;
  stroke-width: 0.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.branch.active {
  stroke: var(--color-primary, #4f46e5);
  opacity: 1;
}

.rating-pill {
  fill: #fff;
  stroke: #e5e7eb;
  stroke-width: 0.3;
}

.rating-text {
  font-size: 3.6px;
  font-weight: 700;
  fill: #b45309;
}

.you-dot {
  fill: var(--color-primary, #4f46e5);
}

.you-label {
  font-size: 3.4px;
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
  font-size: 4.2px;
  font-weight: 700;
  pointer-events: none;
}

.level-2 .node-initial {
  font-size: 3.6px;
}

.node-name {
  font-size: 3.8px;
  fill: #374151;
  pointer-events: none;
}

.node:hover circle:not(.node-hit),
.node:focus-visible circle:not(.node-hit),
.node.active circle:not(.node-hit) {
  stroke: var(--color-primary, #4f46e5);
  stroke-width: 0.9;
}

/* A small square card: name, rating and deals, then what others said */
.node-popup {
  position: absolute;
  z-index: 10;
  width: 13rem;
  padding: 0.5rem 0.625rem;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 0.5rem;
  background: #fff;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.15);
  font-size: 0.8rem;
}

.popup-name {
  display: block;
  padding-right: 1.75rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  color: var(--color-primary, #4f46e5);
}

.popup-close {
  position: absolute;
  top: 0.125rem;
  right: 0.125rem;
  min-width: 32px;
  min-height: 32px;
  border: none;
  background: none;
  font-size: 1.1rem;
  line-height: 1;
  cursor: pointer;
  color: var(--color-text-light, #6b7280);
}

.popup-meta,
.popup-note {
  margin: 0.125rem 0 0;
  color: var(--color-text-light, #6b7280);
}

.popup-comments {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin: 0.5rem 0 0;
  padding: 0.5rem 0 0;
  border-top: 1px solid #f3f4f6;
  list-style: none;
  overflow-wrap: anywhere;
}

.comment-rating {
  font-weight: 700;
  color: #b45309;
}

.comment-by {
  color: var(--color-text-light, #6b7280);
}

.network-side {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 1rem;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 0.85rem;
}

.legend-dot {
  display: inline-block;
  width: 0.75rem;
  height: 0.75rem;
  margin-right: 0.35rem;
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
</style>
