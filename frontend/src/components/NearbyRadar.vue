<template>
  <section class="radar-card" aria-label="Near you">
    <NearbyBanner :state="viewer.state.value" @request="viewer.request" />

    <template v-if="viewer.state.value === 'ready'">
      <svg
        class="radar"
        viewBox="0 0 100 100"
        role="group"
        :aria-label="`Distance radar: ${countsSentence(totals)} posted by others`"
      >
        <!-- Bands: one shaded disc per distance bucket, darker nearer you
             (drawn outermost first), and a dotted edge ring for posts with
             no shared location -->
        <g class="rings" aria-hidden="true">
          <circle :cx="CENTRE" :cy="CENTRE" :r="ringRadius(UNKNOWN_RING)" class="ring unknown" />
          <circle
            v-for="ring in BANDS"
            :key="ring"
            :cx="CENTRE"
            :cy="CENTRE"
            :r="ringRadius(ring)"
            :class="['band', `band-${ring}`]"
          />
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
          <!-- Solid dots get a white edge to stand out on the bands -->
          <circle
            :cx="spot(post).x"
            :cy="spot(post).y"
            r="2.3"
            :fill="post.bucket < 0 ? '#fff' : KIND_STYLE[post.kind].fill"
            :stroke="post.bucket < 0 ? KIND_STYLE[post.kind].fill : '#fff'"
            :stroke-width="post.bucket < 0 ? 0.8 : 1"
          />
        </g>

        <!-- Labels sit in pills over the dots' gap (LABEL_CLEAR), so they
             read on any band -->
        <g class="labels" aria-hidden="true">
          <g v-for="ring in RINGS" :key="`label-${ring}`">
            <rect
              :x="labelPoint(ring).x - pillWidth(ring) / 2"
              :y="labelPoint(ring).y - 3.1"
              :width="pillWidth(ring)"
              height="4.3"
              rx="2.15"
              class="label-pill"
            />
            <text :x="labelPoint(ring).x" :y="labelPoint(ring).y" class="ring-label" text-anchor="middle">{{ RING_LABELS[ring] }}</text>
          </g>
        </g>

        <!-- Drawn last so dots on a busy inner ring never hide it -->
        <g class="you" aria-hidden="true">
          <circle :cx="CENTRE" :cy="CENTRE" r="3" class="you-dot" />
          <text :x="CENTRE" :y="CENTRE + 6.4" text-anchor="middle" class="you-label">You</text>
        </g>
      </svg>

      <!-- The key doubles as a filter: tap a kind to see only that -->
      <div class="legend" role="group" aria-label="Show only">
        <button
          v-for="kind in KINDS"
          :key="kind"
          type="button"
          :class="['legend-item', { dimmed: filter && filter !== kind }]"
          :aria-pressed="filter === kind"
          @click="toggleFilter(kind)"
        >
          <span class="legend-dot" :style="{ background: KIND_STYLE[kind].fill }" aria-hidden="true"></span>
          {{ LABELS[kind] }} {{ totals[kind] }}
        </button>
      </div>

      <p v-if="unplacedCount" class="radar-note">Hollow dots: no location shared.</p>
      <p v-if="moreThanShown" class="radar-note">The radar shows the 5 nearest of each kind; the list has more.</p>
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
        <h2 class="visually-hidden">Notes near you</h2>
        <ul class="note-list">
          <li v-for="post in shown" :key="`list-${post.kind}-${post.id}`">
            <StickyNote size="row" :tone="TONE[post.kind]" :seed="post.id" :to="routeFor(post)" :tape="post.kind !== 'item'" fold>
              <template #header>
                <span v-if="post.kind === 'item'" class="row-thumb" :class="{ 'tilt-left': post.id % 2 === 0 }">
                  <img v-if="post.photo" :src="post.photo" alt="" loading="lazy" />
                  <span v-else aria-hidden="true">📦</span>
                </span>
                <span class="row-main">
                  <span class="row-title">{{ post.title }}</span>
                  <span v-if="post.kind === 'item'" class="row-meta">{{ [distanceText(post), timeLeft(post.deadline)].filter(Boolean).join(' · ') }}</span>
                  <span v-else class="row-body">{{ post.description }}</span>
                </span>
                <span v-if="post.kind !== 'item'" class="row-side">
                  <span>{{ distanceText(post) }}</span>
                  <span>{{ timeLeft(post.deadline) }}</span>
                </span>
              </template>
            </StickyNote>
          </li>
        </ul>
        <p v-if="!shown.length" class="radar-note">Nothing like that near you right now.</p>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import NearbyBanner from '@/components/NearbyBanner.vue'
import StickyNote from '@/components/StickyNote.vue'
import { timeLeft } from '@/utils/gigNote'
import { TONE, postRoute } from '@/utils/postApi'
import { useViewerLocation } from '@/composables/useViewerLocation'
import { useNearbyPosts, type NearbyPost } from '@/composables/useNearbyPosts'
import { CENTRE, KIND_STYLE, RING_LABELS, UNKNOWN_RING, countsSentence, labelPoint, layoutDots, ringRadius, type PostKind } from '@/utils/radar'

const KINDS: PostKind[] = ['task', 'item', 'request']
const LABELS: Record<PostKind, string> = { task: 'Gigs', item: 'For sale', request: 'Wanted' }

const router = useRouter()
const viewer = useViewerLocation()
const { posts, totals, loading, failed, reload } = useNearbyPosts(viewer.location)
const RINGS = Array.from({ length: UNKNOWN_RING + 1 }, (_, i) => i)
// Distance bands, outermost first so nearer ones paint over them
const BANDS = RINGS.slice(0, UNKNOWN_RING).reverse()
// SVG can't size a box to its text: about 1.7 units per character at 3px
const pillWidth = (ring: number) => RING_LABELS[ring].length * 1.7 + 2.4

// One kind at a time, or everything
const filter = ref<PostKind | null>(null)
const toggleFilter = (kind: PostKind) => {
  filter.value = filter.value === kind ? null : kind
}
const shown = computed(() => (filter.value ? posts.value.filter((post) => post.kind === filter.value) : posts.value))

const unplacedCount = computed(() => drawn.value.filter((post) => post.bucket < 0).length)

// The radar shows the nearest PER_KIND_ON_RADAR of each kind (15 dots at
// most); the key and summary still count everything
const PER_KIND_ON_RADAR = 5
const onRadar = computed(() => {
  const taken: Record<PostKind, number> = { task: 0, item: 0, request: 0 }
  return shown.value.filter((post) => taken[post.kind]++ < PER_KIND_ON_RADAR)
})
const moreThanShown = computed(() => shown.value.length > onRadar.value.length)

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
const distanceText = (post: NearbyPost) => post.distance || 'No location'

const routeFor = (post: NearbyPost) => postRoute(post.kind, post.id)
const open = (post: NearbyPost) => router.push(routeFor(post))
</script>

<style scoped>
.radar-card {
  max-width: 440px;
  margin: 0 auto;
  padding: 0 0 1.5rem;
}

.radar {
  width: 100%;
  max-width: 300px;
  aspect-ratio: 1;
  display: block;
  margin: 0 auto;
}

/* Darkest nearest you, fading outwards; white edges keep bands distinct */
.band {
  stroke: #fff;
  stroke-width: 0.7;
}

.band-0 { fill: #FFD0C4; }
.band-1 { fill: #FFDDD3; }
.band-2 { fill: #FFE7E0; }
.band-3 { fill: #FFF1EC; }
.band-4 { fill: #FFF8F5; }

.ring.unknown {
  fill: none;
  stroke: #E8D6D0;
  stroke-width: 0.5;
  stroke-dasharray: 1.3 1.3;
}

.label-pill {
  fill: rgba(255, 255, 255, 0.85);
}

.ring-label {
  font-size: 3px;
  font-weight: 600;
  fill: #A5503F;
}

.you-dot {
  fill: var(--accent);
  stroke: #fff;
  stroke-width: 1;
}

.you-label {
  font-size: 4px;
  font-weight: 700;
  fill: var(--accent-text);
  stroke: #FFD0C4;
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
  fill: rgba(245, 138, 122, 0.15);
  stroke: var(--accent-text);
  stroke-width: 0.5;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0 20px;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  padding: 10px 4px;
  border: 0;
  background: transparent;
  color: #374151;
  font-size: 14px;
  cursor: pointer;
}

.legend-item[aria-pressed='true'] {
  font-weight: 600;
  color: var(--text);
  text-decoration: underline;
  text-underline-offset: 4px;
}

.legend-item.dimmed {
  opacity: 0.5;
}

.legend-dot {
  width: 9px;
  height: 9px;
  border-radius: 5px;
}

.radar-note {
  margin: 0 0 0.4rem;
  font-size: 13px;
  text-align: center;
  color: var(--text-muted);
}

.link-button {
  min-height: 44px;
  padding: 0 0.25rem;
  border: none;
  background: none;
  color: var(--accent-text);
  text-decoration: underline;
  cursor: pointer;
}

.note-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 0;
  padding: 18px 22px 0;
}

.row-thumb {
  flex: none;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3px;
  border-radius: 4px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(17, 24, 39, 0.15);
  font-size: 22px;
  transform: rotate(2.5deg);
}

.row-thumb.tilt-left {
  transform: rotate(-3deg);
}

.row-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 2px;
}

.row-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.row-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 1.3;
}

.row-body {
  font-size: 14px;
  color: var(--text-body);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-meta,
.row-side {
  font-size: 12px;
  color: var(--text-muted);
}

.row-side {
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

</style>
