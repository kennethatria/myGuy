<template>
  <div class="board">
    <div class="board-tabs" role="tablist" aria-label="Marketplace views">
      <button
        v-for="tab in tabs"
        :key="tab.view"
        role="tab"
        :aria-selected="view === tab.view"
        :class="['board-tab', { active: view === tab.view }]"
        @click="showView(tab.view)"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="board-content">
      <NearbyBanner v-if="view !== 'mine'" :state="viewer.state.value" @request="viewer.request" />

      <p v-if="loading" class="board-status" role="status">Loading...</p>

      <div v-else-if="error" class="alert-danger" role="alert">
        {{ error }}
        <button @click="load" class="btn btn-sm btn-outline">Retry</button>
      </div>

      <template v-else-if="view === 'board'">
        <ul v-if="items.length > 0" class="note-board" aria-label="Listings">
          <li v-for="item in items" :key="item.id">
            <StickyNote
              tone="sell"
              :seed="item.id"
              :tape="!!photoUrl(item)"
              :photo="photoUrl(item)"
              :photo-alt="item.title"
              :to="{ name: 'store-item', params: { id: item.id } }"
            >
              <template #header>
                <span class="note-top">
                  <h3 class="note-headline">{{ item.title }}</h3>
                </span>
                <p class="note-text">{{ item.description }}</p>
              </template>
              <template #footer>
                <span class="note-meta">{{ noteMeta(item, item.seller?.username || 'someone', showUnknownItems, now) }}</span>
                <span v-if="item.status === 'reserved'" class="reserved-tag">Reserved</span>
              </template>
            </StickyNote>
          </li>
        </ul>

        <EmptyState
          v-else
          emoji="📦"
          title="Nothing for sale yet"
          text="Be the first to sell something."
          :action="{ label: 'Sell something', to: { name: 'create-listing' } }"
        />
      </template>

      <template v-else-if="view === 'wanted'">
        <ul v-if="requests.length > 0" class="note-board" aria-label="Requests">
          <li v-for="request in requests" :key="request.id">
            <StickyNote
              tone="want"
              tape
              :seed="request.id"
              :to="{ name: 'store-request', params: { id: request.id } }"
            >
              <template #header>
                <span class="note-top">
                  <h3 class="note-headline">{{ request.title }}</h3>
                </span>
                <p class="note-text">{{ request.description }}</p>
              </template>
              <template #footer>
                <span class="note-meta">{{ noteMeta(request, request.requester?.username || 'someone', showUnknownRequests, now) }}</span>
                <span v-if="request.offer_count" class="note-meta offers">{{ offersLabel(request.offer_count) }}</span>
              </template>
            </StickyNote>
          </li>
        </ul>

        <EmptyState
          v-else
          emoji="🙋"
          title="Nobody is looking for anything yet"
          text="Ask for something you need, and sellers can list it for you."
          :action="{ label: 'Ask for something', to: { name: 'create-request' } }"
        />
      </template>

      <template v-else>
        <h2 class="section-title">Your listings</h2>
        <ul v-if="myItems.length > 0" class="note-board" aria-label="Your listings">
          <li v-for="item in myItems" :key="item.id">
            <StickyNote tone="sell" :seed="item.id" :photo="photoUrl(item)" :photo-alt="item.title">
              <template #header>
                <span class="note-top">
                  <h3 class="note-headline">
                    <router-link :to="{ name: 'store-item', params: { id: item.id } }" class="note-title-link">
                      {{ item.title }}
                    </router-link>
                  </h3>
                </span>
                <p class="note-text">{{ item.description }}</p>
              </template>
              <template #footer>
                <span class="note-meta">{{ statusLabel(item) }}</span>
                <span v-if="item.status === 'expired'" class="note-actions">
                  <button class="btn btn-sm btn-primary" :disabled="busyId === `items-${item.id}`" @click="repost('items', item)">
                    {{ busyId === `items-${item.id}` ? 'Reposting...' : 'Repost' }}
                  </button>
                  <button class="btn btn-sm btn-outline" :disabled="busyId === `items-${item.id}`" @click="remove('items', item)">
                    Remove
                  </button>
                </span>
              </template>
            </StickyNote>
          </li>
        </ul>
        <EmptyState
          v-else
          emoji="📦"
          title="Nothing listed"
          text="Things you sell show up here, with who asked to book them."
          :action="{ label: 'Sell something', to: { name: 'create-listing' } }"
        />

        <h2 class="section-title">Your requests</h2>
        <ul v-if="myRequests.length > 0" class="note-board" aria-label="Your requests">
          <li v-for="request in myRequests" :key="request.id">
            <StickyNote tone="want" :seed="request.id">
              <template #header>
                <span class="note-top">
                  <h3 class="note-headline">
                    <router-link :to="{ name: 'store-request', params: { id: request.id } }" class="note-title-link">
                      {{ request.title }}
                    </router-link>
                  </h3>
                </span>
                <p class="note-text">{{ request.description }}</p>
              </template>
              <template #footer>
                <span class="note-meta">{{ requestStatusLabel(request) }}</span>
                <span v-if="request.status === 'expired'" class="note-actions">
                  <button class="btn btn-sm btn-primary" :disabled="busyId === `requests-${request.id}`" @click="repost('requests', request)">
                    {{ busyId === `requests-${request.id}` ? 'Reposting...' : 'Repost' }}
                  </button>
                  <button class="btn btn-sm btn-outline" :disabled="busyId === `requests-${request.id}`" @click="remove('requests', request)">
                    Remove
                  </button>
                </span>
              </template>
            </StickyNote>
          </li>
        </ul>
        <EmptyState
          v-else
          emoji="🙋"
          title="No requests"
          text="Things you ask for show up here, with what sellers list for you."
          :action="{ label: 'Ask for something', to: { name: 'create-request' } }"
        />
      </template>

      <nav v-if="view !== 'mine' && !loading && totalPages > 1" aria-label="Board pages">
        <ul class="pagination">
          <li class="page-item" :class="{ disabled: currentPage === 1 }">
            <button class="page-link" @click="goToPage(currentPage - 1)" :disabled="currentPage === 1">
              Previous
            </button>
          </li>
          <li class="page-item active">
            <span class="page-link" aria-current="page">{{ currentPage }} of {{ totalPages }}</span>
          </li>
          <li class="page-item" :class="{ disabled: currentPage === totalPages }">
            <button class="page-link" @click="goToPage(currentPage + 1)" :disabled="currentPage === totalPages">
              Next
            </button>
          </li>
        </ul>
      </nav>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import NearbyBanner from '@/components/NearbyBanner.vue'
import EmptyState from '@/components/EmptyState.vue'
import { hasDistances } from '@/utils/distance'
import { useViewerLocation, nearParam } from '@/composables/useViewerLocation'
import { expiryLabel, noteMeta } from '@/utils/gigNote'
import { offersLabel } from '@/utils/listingNote'
import { trackEvent } from '@/utils/analytics'

interface StoreItem {
  id: number
  title: string
  description: string
  status: string
  deadline?: string
  distance?: string
  seller?: { id: number; username: string }
  images?: { id: number; url: string }[]
}

interface ItemRequest {
  id: number
  title: string
  description: string
  status: string
  deadline?: string
  offer_count: number
  distance?: string
  requester?: { id: number; username: string }
}

type View = 'board' | 'wanted' | 'mine'
type Kind = 'items' | 'requests'

const tabs: { view: View; label: string }[] = [
  { view: 'board', label: 'For sale' },
  { view: 'wanted', label: 'Wanted' },
  { view: 'mine', label: 'Yours' }
]

const authStore = useAuthStore()
const route = useRoute()
const router = useRouter()

// The tab is kept in the URL so back and shared links land on it
const tabFromQuery = (tab: unknown): View => tabs.find((t) => t.view === tab)?.view ?? 'board'
const view = ref<View>(tabFromQuery(route.query.tab))
const loading = ref(false)
const error = ref('')
const items = ref<StoreItem[]>([])
const requests = ref<ItemRequest[]>([])
const myItems = ref<StoreItem[]>([])
const myRequests = ref<ItemRequest[]>([])
const total = ref(0)
const currentPage = ref(1)
const perPage = 24
// Nearest first once the viewer's rough location is known
const viewer = useViewerLocation(() => {
  currentPage.value = 1
  if (view.value !== 'mine') load()
})
// No sort picker: nearest first when the viewer's area is known, else newest
const sortBy = computed(() => (viewer.location.value ? 'distance' : 'created_at'))
const busyId = ref<string | null>(null)
const totalPages = computed(() => Math.ceil(total.value / perPage))
const showUnknownItems = computed(() => hasDistances(items.value))
const showUnknownRequests = computed(() => hasDistances(requests.value))

// Countdowns move without refetching
const now = ref(new Date())
let clock: ReturnType<typeof setInterval> | undefined

const authHeaders = () => ({ Authorization: `Bearer ${authStore.token}` })

const photoUrl = (item: StoreItem) =>
  item.images?.length ? config.STORE_API_BASE_URL + item.images[0].url : ''

const statusLabel = (item: StoreItem) => {
  if (item.status === 'active') {
    return (item.deadline && expiryLabel(item.deadline, now.value)) || 'On the board'
  }
  if (item.status === 'expired') return 'No bookings within 24 hours'
  return item.status.charAt(0).toUpperCase() + item.status.slice(1)
}

const requestStatusLabel = (request: ItemRequest) => {
  if (request.status === 'fulfilled') return 'Fulfilled'
  if (request.status === 'expired') return 'No listings within 24 hours'
  if (request.offer_count) return offersLabel(request.offer_count)
  return (request.deadline && expiryLabel(request.deadline, now.value)) || 'On the board'
}

const getJSON = async (path: string, failure: string) => {
  const response = await fetch(`${config.STORE_API_URL}${path}`, { headers: authHeaders() })
  if (!response.ok) throw new Error(failure)
  return response.json()
}

// The live notes of others, newest or soonest to come down first
const boardParams = (excludeKey: string) => {
  const params = new URLSearchParams({
    sort_by: sortBy.value,
    sort_order: 'desc',
    page: String(currentPage.value),
    per_page: String(perPage)
  })
  if (authStore.user?.id) params.append(excludeKey, String(authStore.user.id))
  // The viewer's rough location sorts (or just tags) the notes by distance
  if (viewer.location.value) params.append('near', nearParam(viewer.location.value))
  return params
}

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    if (view.value === 'board') {
      // No status: the board's default, live items for sale
      const params = boardParams('exclude_seller_id')
      const data = await getJSON(`/items?${params}`, 'Failed to load listings')
      items.value = data.items ?? []
      total.value = data.total ?? 0
    } else if (view.value === 'wanted') {
      const data = await getJSON(`/requests?${boardParams('exclude_requester_id')}`, 'Failed to load requests')
      requests.value = data.requests ?? []
      total.value = data.total ?? 0
    } else {
      const [listings, asks] = await Promise.all([
        getJSON('/user/listings', 'Failed to load your listings'),
        getJSON('/user/requests', 'Failed to load your requests')
      ])
      // Sold listings and fulfilled requests leave your lists; their history
      // stays in the conversation and their pages still open from links
      myItems.value = listings.filter((item: StoreItem) => item.status !== 'sold')
      myRequests.value = asks.filter((request: ItemRequest) => request.status !== 'fulfilled')
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load the marketplace'
  } finally {
    loading.value = false
  }
}

const showView = (next: View) => {
  view.value = next
  currentPage.value = 1
  router.replace({ query: next === 'board' ? {} : { tab: next } })
  load()
}

// Links to another tab (the side navigation's My Requests) while already here
watch(() => route.query.tab, (tab) => {
  const next = tabFromQuery(tab)
  if (next === view.value) return
  view.value = next
  currentPage.value = 1
  load()
})

// Put an expired note back on the board for a fresh 24 hours
const repost = async (kind: Kind, note: { id: number }) => {
  busyId.value = `${kind}-${note.id}`
  try {
    const response = await fetch(`${config.STORE_API_URL}/${kind}/${note.id}/repost`, {
      method: 'POST',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not repost it. Please try again.')
    trackEvent('post-reposted', { kind: kind === 'items' ? 'listing' : 'request' })
    await load()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not repost it. Please try again.')
  } finally {
    busyId.value = null
  }
}

const remove = async (kind: Kind, note: { id: number; title: string }) => {
  if (!confirm(`Remove "${note.title}" for good?`)) return
  busyId.value = `${kind}-${note.id}`
  try {
    const response = await fetch(`${config.STORE_API_URL}/${kind}/${note.id}`, {
      method: 'DELETE',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not remove it. Please try again.')
    if (kind === 'items') {
      myItems.value = myItems.value.filter((mine) => mine.id !== note.id)
    } else {
      myRequests.value = myRequests.value.filter((mine) => mine.id !== note.id)
    }
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not remove it. Please try again.')
  } finally {
    busyId.value = null
  }
}

const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    load()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

onMounted(() => {
  load()
  clock = setInterval(() => { now.value = new Date() }, 60_000)
})

onUnmounted(() => {
  if (clock) clearInterval(clock)
})
</script>

<style scoped src="@/assets/board.css"></style>

<style scoped>
.section-title {
  margin: 8px 0 12px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-muted);
}

.section-title ~ .section-title {
  margin-top: 28px;
}

/* The title link covers the whole note (see .note-title-link::after) */
.note-board > li {
  position: relative;
}

.note-title-link {
  color: inherit;
  font-weight: inherit;
}

.note-title-link::after {
  content: '';
  position: absolute;
  inset: 0;
}

.note-title-link:hover,
.note-title-link:focus-visible {
  color: inherit;
  text-decoration: underline;
}

/* Above the note-wide link, and big enough for a thumb */
.note-actions {
  position: relative;
  z-index: 1;
  display: flex;
  gap: 0.5rem;
}

.note-actions .btn {
  min-height: 44px;
}

.offers {
  flex: none;
  font-weight: 600;
}
</style>
