<template>
  <div class="network-page">
    <section class="network-card" :aria-label="isMine ? 'Your network' : `${nameOf(centreId)}'s network`">
      <div class="network-header">
        <h2 v-if="!isMine" class="network-title">{{ nameOf(centreId) }}'s network</h2>
        <!-- What this is, on a small taped note -->
        <StickyNote v-if="isMine" tone="sell" :seed="1" tape fold class="network-intro">
          <template #header>
            <p class="network-intro-text">
              Everyone you've done a gig or a sale with, and the rating you gave each other.
              <strong>Tap someone</strong> to see what others say.
            </p>
          </template>
        </StickyNote>
        <p v-if="branches.length" class="network-summary">
          {{ peopleCount }} {{ peopleCount === 1 ? 'person' : 'people' }} in {{ isMine ? 'your' : 'their' }} network{{ trimmed ? `. Showing the ${MAX_PEOPLE} most recent.` : '' }}
        </p>
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
          :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
          role="group"
          :aria-label="`${isMine ? 'Your' : `${nameOf(centreId)}'s`} network: ${peopleCount} ${peopleCount === 1 ? 'person' : 'people'}`"
        >
          <!-- Shaded bands as on the home radar, darker nearer the centre
               (drawn outermost first); people sit on the band edges, your
               connections on the inner one and theirs on the outer -->
          <g class="rings" aria-hidden="true">
            <circle
              v-for="(radius, i) in bandRadii"
              :key="`band-${i}`"
              :cx="layout.centre.x"
              :cy="layout.centre.y"
              :r="radius"
              :class="['band', `band-${bandRadii.length - 1 - i}`]"
            />
          </g>

          <!-- The links, coloured by the average rating between the two;
               the tapped one stands out -->
          <g class="links" aria-hidden="true">
            <line
              v-for="node in placed"
              :key="`link-${keyOf(node)}`"
              :x1="node.from.x"
              :y1="node.from.y"
              :x2="node.x"
              :y2="node.y"
              :class="['link', tierOf(node.connection.averageRating), { active: selectedKey === keyOf(node) }]"
            />
          </g>

          <!-- The average rating between the two on every link, in its colour -->
          <g class="ratings" aria-hidden="true">
            <g v-for="node in placed" :key="`rating-${keyOf(node)}`" :class="tierOf(node.connection.averageRating)">
              <rect :x="node.label.x - 5" :y="node.label.y - 2.5" width="10" height="5" rx="2.5" class="rating-pill" />
              <text :x="node.label.x" :y="node.label.y + 1.1" text-anchor="middle" class="rating-text small">
                ★{{ formatRating(node.connection.averageRating) }}
              </text>
            </g>
          </g>

          <g class="you" aria-hidden="true">
            <circle :cx="layout.centre.x" :cy="layout.centre.y" :r="CENTRE_RADIUS" class="you-dot" />
            <text :x="layout.centre.x" :y="layout.centre.y + 1.1" text-anchor="middle" class="you-label">{{ isMine ? 'You' : initialOf(centreId) }}</text>
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
            <circle :cx="node.x" :cy="node.y" :r="node.r" :class="['node-dot', tierOf(node.connection.averageRating)]" />
            <text :x="node.x" :y="node.y + 1.2" text-anchor="middle" class="node-initial">
              {{ initialOf(node.connection.userId) }}
            </text>
          </g>

          <!-- The tapped person's name, and their link's rating drawn larger on top -->
          <g v-if="selected" class="selection" aria-hidden="true">
            <text
              :x="selected.x < layout.centre.x ? selected.x - selected.r - 1.5 : selected.x + selected.r + 1.5"
              :y="selected.y + 1.4"
              :text-anchor="selected.x < layout.centre.x ? 'end' : 'start'"
              class="node-name"
            >{{ shortName(selected.connection.userId) }}</text>
            <rect :x="selected.label.x - 6" :y="selected.label.y - 3.1" width="12" height="6.2" rx="3.1" :class="['rating-pill', tierOf(selected.connection.averageRating)]" />
            <text :x="selected.label.x" :y="selected.label.y + 1.3" text-anchor="middle" :class="['rating-text', tierOf(selected.connection.averageRating)]">
              ★ {{ formatRating(selected.connection.averageRating) }}
            </text>
          </g>
        </svg>

        <!-- A small card about the tapped person, under their node; it goes
             away by itself after a few seconds -->
        <div
          v-if="selected"
          ref="popup"
          class="node-popup"
          :style="{ left: `min(${selected.x}%, calc(100% - 13rem))`, top: `${((selected.y + selected.r + 2) / HEIGHT) * 100}%` }"
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
            {{ selected.connection.deals }} {{ selected.connection.deals === 1 ? 'deal' : 'deals' }} ·
            {{ VIA_LABEL[selected.connection.via] }}
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
          <li v-for="tier in TIERS" :key="tier.tier">
            <span :class="['legend-dot', tier.tier]" aria-hidden="true"></span>
            {{ tier.label }}
          </li>
        </ul>
        <p class="network-note">
          Each line's colour is the average rating between the two people, both ways. On the outer ring, the people each connection has worked with.
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
import StickyNote from '@/components/StickyNote.vue'
import {
  connectionsFrom, recentNetwork, layoutNetwork, networkSize, tierOf, MAX_PEOPLE, formatRating, WIDTH, HEIGHT, EDGE_RADIUS, CENTRE_RADIUS,
  type Branch, type PlacedConnection, type Via, type Tier
} from '@/utils/network'

const reviewsStore = useReviewsStore()
const authStore = useAuthStore()
const route = useRoute()
const userStore = useUserStore()

const VIA_LABEL: Record<Via | 'both', string> = { gig: 'gigs', item: 'marketplace', both: 'gigs and marketplace' }
const TIERS: { tier: Tier; label: string }[] = [
  { tier: 'strong', label: 'Strong 4.5+' },
  { tier: 'fair', label: 'Fair 3–4.4' },
  { tier: 'weak', label: 'Weak below 3' }
]

// The card closes by itself after this long
const POPUP_MS = 10_000

const branches = ref<Branch[]>([])
// Everyone in the centre person's network, and whether the picture leaves
// some people out to stay within MAX_PEOPLE
const peopleCount = ref(0)
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
// Each band's outer edge, outermost first: the faint edge band, then the rings
const bandRadii = computed(() => (layout.value.rings.length ? [EDGE_RADIUS, ...[...layout.value.rings].reverse()] : []))

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
    const newest = [...all].sort((a, b) => b.latest.localeCompare(a.latest))
    const firsts = newest.slice(0, MAX_PEOPLE)
    const children = await Promise.all(firsts.map(first =>
      reviewsStore.fetchInteractions(first.userId)
        .then(list => connectionsFrom(list).filter(child => child.userId !== centre))
        .catch(() => [])
    ))
    const full = firsts.map((connection, i) => ({ connection, children: children[i] }))
    branches.value = recentNetwork(full)
    // Everyone, including connections too old to have made the picture
    peopleCount.value = networkSize([...full, ...newest.slice(MAX_PEOPLE).map(connection => ({ connection, children: [] }))], centre)
    trimmed.value = networkSize(branches.value, centre) < peopleCount.value
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
.network-page {
  max-width: 600px;
  margin: 0 auto;
  padding: 6px 22px 1.5rem;
}

.network-header {
  margin-bottom: 0.75rem;
}

.network-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
}

/* Doubled to win over StickyNote's own size */
.network-intro.network-intro {
  margin-bottom: 12px;
  padding: 16px 18px;
  transform: rotate(-0.5deg);
}

.network-intro-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.4;
  color: var(--text-body);
}

.network-intro-text strong {
  font-weight: 600;
  color: var(--text);
}

.network-summary,
.network-note {
  margin: 0.25rem 0 0;
  font-size: 13px;
  color: var(--text-muted);
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

/* Tiers: strong, fair and weak links (lines, outlines, pills, legend) */
.strong { --tier: #3E7F46; --tier-soft: #E8F6EA; --tier-text: #2B5E32; }
.fair { --tier: #B07A2C; --tier-soft: #FFF4D6; --tier-text: #713F12; }
.weak { --tier: #B9402F; --tier-soft: #FDE8E4; --tier-text: #102A5E; }

/* Bands: darkest nearest the centre, as on the home radar; the white
   edges are the lines people sit on */
.band {
  stroke: #fff;
  stroke-width: 0.6;
}

.band-0 { fill: #E2ECFF; }
.band-1 { fill: #EDF3FF; }
.band-2 { fill: #F5F9FF; }

.link {
  stroke: var(--tier);
  stroke-width: 0.9;
  stroke-linecap: round;
  opacity: 0.85;
}

.link.active {
  stroke-width: 1.4;
  opacity: 1;
}

.rating-pill {
  fill: var(--tier-soft);
  stroke: var(--tier);
  stroke-width: 0.3;
}

.rating-text {
  font-size: 3.4px;
  font-weight: 700;
  fill: var(--tier-text);
}

.rating-text.small {
  font-size: 2.8px;
}

.you-dot {
  fill: var(--accent);
  stroke: #fff;
  stroke-width: 0.8;
}

.you-label {
  font-size: 3px;
  font-weight: 700;
  fill: var(--on-accent);
}

.node {
  cursor: pointer;
  outline: none;
}

.node-hit {
  fill: transparent;
}

.node-dot {
  fill: #475569;
  stroke: var(--tier);
  stroke-width: 1;
}

.level-2 .node-dot {
  fill: #6b7280;
}

.node-initial {
  font-size: 3.6px;
  font-weight: 700;
  fill: #fff;
  pointer-events: none;
}

.level-2 .node-initial {
  font-size: 3px;
}

.node-name {
  font-size: 3.6px;
  font-weight: 700;
  fill: #111827;
  pointer-events: none;
}

.node:hover .node-dot,
.node:focus-visible .node-dot,
.node.active .node-dot {
  stroke-width: 1.6;
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
  color: var(--color-primary);
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
  border-radius: 50%;
  background: var(--tier);
  vertical-align: -0.1rem;
}

.link-button {
  border: none;
  background: none;
  padding: 0;
  color: var(--color-primary);
  text-decoration: underline;
  cursor: pointer;
}
</style>
