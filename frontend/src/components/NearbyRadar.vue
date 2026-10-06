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
          :aria-label="`Distance radar: ${countsSentence(allCounts)} near you`"
        >
          <!-- Rings: one square per distance bucket, nearest in the middle -->
          <g class="rings" aria-hidden="true">
            <rect
              v-for="(half, i) in RING_HALF"
              :key="half"
              :x="CENTRE - half"
              :y="CENTRE - half"
              :width="half * 2"
              :height="half * 2"
              :class="['ring', { outer: i === RING_HALF.length - 1 }]"
              rx="1.5"
            />
            <text
              v-for="(half, i) in RING_HALF"
              :key="`label-${half}`"
              :x="CENTRE + half - 1"
              :y="CENTRE - half + 3.2"
              class="ring-label"
              text-anchor="end"
            >{{ BUCKETS[i] }}</text>
          </g>

          <g class="you" aria-hidden="true">
            <circle :cx="CENTRE" :cy="CENTRE" r="2.2" class="you-dot" />
            <text :x="CENTRE" :y="CENTRE + 6" text-anchor="middle" class="you-label">You</text>
          </g>

          <g
            v-for="post in posts"
            :key="`${post.kind}-${post.id}`"
            class="dot"
            role="link"
            tabindex="0"
            :aria-label="`${KIND_STYLE[post.kind].one}: ${post.title}, ${post.distance}`"
            @click="open(post)"
            @keydown.enter.prevent="open(post)"
            @keydown.space.prevent="open(post)"
          >
            <title>{{ post.title }} · {{ post.distance }}</title>
            <!-- A larger, invisible circle makes the dot easier to tap -->
            <circle :cx="spot(post).x" :cy="spot(post).y" r="5" class="dot-hit" />
            <circle
              :cx="spot(post).x"
              :cy="spot(post).y"
              r="2.4"
              :fill="KIND_STYLE[post.kind].fill"
              :stroke="KIND_STYLE[post.kind].stroke"
              stroke-width="0.6"
            />
          </g>
        </svg>

        <div class="radar-side">
          <ul class="legend" aria-label="Key">
            <li v-for="kind in KINDS" :key="kind">
              <span class="legend-dot" :style="{ background: KIND_STYLE[kind].fill, borderColor: KIND_STYLE[kind].stroke }" aria-hidden="true"></span>
              {{ label(kind) }} <span class="legend-count">{{ allCounts[kind] }}</span>
            </li>
          </ul>

          <p v-if="loading && !posts.length" class="radar-note">Finding what's near you...</p>
          <p v-else-if="failed" class="radar-note">
            Couldn't load nearby posts.
            <button type="button" class="link-button" @click="reload">Try again</button>
          </p>
          <p v-else-if="!posts.length" class="radar-note">Nothing with a location near you yet.</p>

          <template v-else>
            <h3 class="closest-title">Closest to you</h3>
            <ul class="closest">
              <li v-for="post in posts.slice(0, 5)" :key="`list-${post.kind}-${post.id}`">
                <router-link :to="routeFor(post)" class="closest-link">
                  <span class="legend-dot" :style="{ background: KIND_STYLE[post.kind].fill, borderColor: KIND_STYLE[post.kind].stroke }" aria-hidden="true"></span>
                  <span class="closest-name">{{ post.title }}</span>
                  <span class="closest-distance">{{ post.distance }}</span>
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
import { BUCKETS, CENTRE, KIND_STYLE, RING_HALF, countsSentence, radarPoint, type PostKind } from '@/utils/radar'

const KINDS: PostKind[] = ['task', 'item', 'request']
const LABELS: Record<PostKind, string> = { task: 'Gigs', item: 'Listings', request: 'Requests' }

const router = useRouter()
const viewer = useViewerLocation()
const { posts, loading, failed, reload } = useNearbyPosts(viewer.location)

const countWhere = (keep: (post: NearbyPost) => boolean) => {
  const counts: Record<PostKind, number> = { task: 0, item: 0, request: 0 }
  for (const post of posts.value) if (keep(post)) counts[post.kind]++
  return counts
}
const allCounts = computed(() => countWhere(() => true))
// "Within 2 km" = the two innermost rings (<1 km and ~2 km)
const closeCounts = computed(() => countWhere((post) => post.bucket <= 1))

const spot = (post: NearbyPost) => radarPoint(post.kind, post.id, post.bucket)
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
  background: #f8fafc;
  border-radius: 0.5rem;
}

.ring {
  fill: none;
  stroke: #cbd5e1;
  stroke-width: 0.4;
  stroke-dasharray: 1.2 1;
}

.ring.outer {
  stroke-dasharray: none;
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
  stroke-width: 1;
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
  margin: 0;
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
  margin: 0 0 0.25rem;
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
