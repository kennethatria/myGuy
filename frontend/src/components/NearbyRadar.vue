<template>
  <section class="radar-card" aria-labelledby="radar-title">
    <div class="radar-header">
      <h2 id="radar-title" class="radar-title">Near you</h2>
      <p v-if="viewer.state.value === 'ready' && !loading" class="radar-summary">
        Within 2 km: {{ countsSentence(closeCounts) }}
      </p>
    </div>

    <NearbyBanner :state="viewer.state.value" @request="viewer.request" />

    <template v-if="viewer.state.value === 'ready'">
      <div class="radar-layout">
        <svg
          class="radar"
          viewBox="0 0 100 100"
          role="group"
          :aria-label="`Distance radar: ${countsSentence(totals)} posted by others`"
        >
          <!-- Rings: one circle per distance bucket, nearest in the middle,
               and a dotted edge ring for posts with no shared location -->
          <g class="rings" aria-hidden="true">
            <circle
              v-for="ring in RINGS"
              :key="ring"
              :cx="CENTRE"
              :cy="CENTRE"
              :r="ringRadius(ring)"
              :class="['ring', { unknown: ring === UNKNOWN_RING }]"
            />
            <text
              v-for="ring in RINGS"
              :key="`label-${ring}`"
              :x="labelPoint(ring).x"
              :y="labelPoint(ring).y"
              class="ring-label"
              text-anchor="middle"
            >{{ RING_LABELS[ring] }}</text>
          </g>


          <g
            v-for="post in drawn"
            :key="`${post.kind}-${post.id}`"
            class="dot"
            role="link"
            tabindex="0"
            :aria-label="`${KIND_STYLE[post.kind].one}: ${post.title}, ${post.distance || 'no location shared'}`"
            @click="open(post)"
            @keydown.enter.prevent="open(post)"
            @keydown.space.prevent="open(post)"
          >
            <title>{{ post.title }} · {{ post.distance || 'no location shared' }}</title>
            <!-- A larger, invisible circle makes the dot easier to tap -->
            <circle :cx="spot(post).x" :cy="spot(post).y" r="5" class="dot-hit" />
            <circle
              :cx="spot(post).x"
              :cy="spot(post).y"
              r="1.6"
              :fill="post.bucket < 0 ? '#fff' : KIND_STYLE[post.kind].fill"
              :stroke="post.bucket < 0 ? KIND_STYLE[post.kind].fill : KIND_STYLE[post.kind].stroke"
              :stroke-width="post.bucket < 0 ? 0.8 : 0.45"
            />
          </g>

          <!-- Drawn last so dots on a busy inner ring never hide it -->
          <g class="you" aria-hidden="true">
            <circle :cx="CENTRE" :cy="CENTRE" r="1.8" class="you-dot" />
            <text :x="CENTRE" :y="CENTRE + 3.6" text-anchor="middle" class="you-label">You</text>
          </g>
        </svg>

        <div class="radar-side">
          <ul class="legend" aria-label="Key">
            <li v-for="kind in KINDS" :key="kind">
              <span class="legend-dot" :style="{ background: KIND_STYLE[kind].fill, borderColor: KIND_STYLE[kind].stroke }" aria-hidden="true"></span>
              {{ label(kind) }} <span class="legend-count">{{ totals[kind] }}</span>
            </li>
          </ul>

          <p v-if="unplacedCount" class="radar-note">
            Hollow dots on the dotted edge ring had no location shared, so their distance is unknown.
          </p>
          <p v-if="moreThanShown" class="radar-note">
            The radar shows the 5 nearest of each kind. See the boards for everything.
          </p>
          <p v-if="hiddenSentence" class="radar-note">
            Busy around you: {{ hiddenSentence }} {{ hiddenTotal === 1 ? "isn't" : "aren't" }} drawn.
          </p>

          <p v-if="loading && !posts.length" class="radar-note">Finding what's near you...</p>
          <p v-else-if="failed" class="radar-note">
            Couldn't load nearby posts.
            <button type="button" class="link-button" @click="reload">Try again</button>
          </p>
          <p v-else-if="!posts.length" class="radar-note">Nothing posted by others yet.</p>

          <template v-else>
            <h3 class="closest-title">Closest to you</h3>
            <ul class="closest">
              <li v-for="post in posts.slice(0, 5)" :key="`list-${post.kind}-${post.id}`">
                <router-link :to="routeFor(post)" class="closest-link">
                  <span class="legend-dot" :style="{ background: KIND_STYLE[post.kind].fill, borderColor: KIND_STYLE[post.kind].stroke }" aria-hidden="true"></span>
                  <span class="closest-name">{{ post.title }}</span>
                  <span v-if="post.distance" class="closest-distance">{{ post.distance }}</span>
                  <span v-else class="closest-distance" role="img" aria-label="Distance unknown: no location was shared">🤷</span>
                </router-link>
              </li>
            </ul>
          </template>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import NearbyBanner from '@/components/NearbyBanner.vue'
import { useViewerLocation } from '@/composables/useViewerLocation'
import { useNearbyPosts, type NearbyPost } from '@/composables/useNearbyPosts'
import { CENTRE, KIND_STYLE, RING_LABELS, UNKNOWN_RING, countsSentence, labelPoint, layoutDots, ringRadius, type PostKind } from '@/utils/radar'

const KINDS: PostKind[] = ['task', 'item', 'request']
const LABELS: Record<PostKind, string> = { task: 'Gigs', item: 'Marketplace', request: 'Requests' }

const router = useRouter()
const viewer = useViewerLocation()
const { posts, totals, loading, failed, reload } = useNearbyPosts(viewer.location)
const RINGS = Array.from({ length: UNKNOWN_RING + 1 }, (_, i) => i)

const countWhere = (keep: (post: NearbyPost) => boolean) => {
  const counts: Record<PostKind, number> = { task: 0, item: 0, request: 0 }
  for (const post of posts.value) if (keep(post)) counts[post.kind]++
  return counts
}
const unplacedCount = computed(() => drawn.value.filter((post) => post.bucket < 0).length)
// "Within 2 km" = the two innermost rings (<1 km and ~2 km)
const closeCounts = computed(() => countWhere((post) => post.bucket >= 0 && post.bucket <= 1))

// The radar shows the nearest PER_KIND_ON_RADAR of each kind (15 dots at
// most); the key and summary still count everything
const PER_KIND_ON_RADAR = 5
const onRadar = computed(() => {
  const taken: Record<PostKind, number> = { task: 0, item: 0, request: 0 }
  return posts.value.filter((post) => taken[post.kind]++ < PER_KIND_ON_RADAR)
})
const moreThanShown = computed(() => posts.value.length > onRadar.value.length)

// Dots sharing a ring are spaced evenly round it; a busy ring draws as many
// as fit (nearest first) and counts the rest
const layout = computed(() => layoutDots(onRadar.value))
const drawn = computed(() => onRadar.value.filter((post) => layout.value.spots.has(`${post.kind}-${post.id}`)))
const spot = (post: NearbyPost) => layout.value.spots.get(`${post.kind}-${post.id}`) ?? { x: CENTRE, y: CENTRE }
const hiddenTotal = computed(() => [...layout.value.hidden.values()].reduce((a, b) => a + b, 0))
const hiddenSentence = computed(() =>
  [...layout.value.hidden.entries()]
    .sort(([a], [b]) => a - b)
    .map(([ring, n]) => (ring === UNKNOWN_RING ? `${n} more without a location` : `${n} more within ${RING_LABELS[ring]}`))
    .join(', ')
)
const label = (kind: PostKind) => LABELS[kind]

const routeFor = (post: NearbyPost) => {
  if (post.kind === 'task') return { name: 'task-detail', params: { id: post.id } }
  if (post.kind === 'item') return { name: 'store-item', params: { id: post.id } }
  return { name: 'store-request', params: { id: post.id } }
}
const open = (post: NearbyPost) => router.push(routeFor(post))
</script>

<style scoped>
.radar-card {
  margin-bottom: 1.5rem;
  padding: 1rem;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 0.75rem;
  background: #fff;
}

.radar-header {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.25rem 1rem;
  margin-bottom: 0.75rem;
}

.radar-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 600;
}

.radar-summary {
  margin: 0;
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.radar-layout {
  display: grid;
  grid-template-columns: minmax(0, 320px) minmax(0, 1fr);
  gap: 1.25rem;
  align-items: start;
}

.radar {
  width: 100%;
  aspect-ratio: 1;
  display: block;
  background: radial-gradient(circle, #f8fafc 0 69%, transparent 70%);
}

.ring {
  fill: none;
  stroke: #cbd5e1;
  stroke-width: 0.4;
  stroke-dasharray: 1.2 1;
}

.ring.unknown {
  stroke: #94a3b8;
  stroke-dasharray: 0.4 1.2;
  stroke-linecap: round;
  stroke-width: 0.6;
}

.ring-label {
  font-size: 2.6px;
  fill: #64748b;
}

.you-dot {
  fill: #111827;
}

.you-label {
  font-size: 3px;
  font-weight: 600;
  fill: #111827;
  stroke: #fff;
  stroke-width: 0.8px;
  paint-order: stroke;
}

.dot {
  cursor: pointer;
  outline: none;
}

.dot-hit {
  fill: transparent;
}

.dot:hover circle:not(.dot-hit),
.dot:focus-visible circle:not(.dot-hit) {
  stroke: #111827;
  stroke-width: 0.8;
}

.dot:focus-visible .dot-hit {
  fill: rgba(79, 70, 229, 0.15);
  stroke: var(--color-primary, #4f46e5);
  stroke-width: 0.5;
}

.legend,
.closest {
  list-style: none;
  margin: 0;
  padding: 0;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 1rem;
  margin-bottom: 0.75rem;
  font-size: 0.9rem;
}

.legend li {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.legend-count {
  font-weight: 600;
}

.legend-dot {
  flex: 0 0 auto;
  width: 0.75rem;
  height: 0.75rem;
  border: 1.5px solid;
  border-radius: 50%;
}

.radar-note {
  margin: 0 0 0.4rem;
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.link-button {
  min-height: 44px;
  padding: 0 0.25rem;
  border: none;
  background: none;
  color: var(--color-primary, #4f46e5);
  text-decoration: underline;
  cursor: pointer;
}

.closest-title {
  margin: 0.75rem 0 0.25rem;
  font-size: 0.95rem;
  font-weight: 600;
}

.closest-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 44px;
  color: inherit;
  text-decoration: none;
  border-bottom: 1px solid #f1f5f9;
}

.closest-link:hover .closest-name,
.closest-link:focus-visible .closest-name {
  text-decoration: underline;
}

.closest-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.closest-distance {
  flex: 0 0 auto;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-light, #6b7280);
}

/* Phones: the radar takes the full width, the key and list go under it */
@media (max-width: 640px) {
  .radar-layout {
    grid-template-columns: 1fr;
  }
}
</style>
