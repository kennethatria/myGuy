<template>
  <div class="container py-4 request-page">
    <router-link :to="{ name: 'store', query: { tab: 'wanted' } }" class="back-link">
      <span aria-hidden="true">←</span> Back to Wanted
    </router-link>

    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else-if="error" class="alert alert-danger" role="alert">{{ error }}</div>

    <template v-else-if="request">
      <StickyNote :seed="request.id" size="large" class="request-note">
        <template #header>
          <span v-if="request.status !== 'active'" class="note-status">{{ statusLabel }}</span>
          <span class="note-kicker">Wanted</span>
          <h1 class="note-detail-headline">{{ request.title }}</h1>
          <p class="note-detail-body">{{ request.description }}</p>
        </template>
        <template #footer>
          <span>@{{ request.requester?.username || 'someone' }}</span>
          <span v-if="request.status === 'active' && request.deadline && expiryLabel(request.deadline, now)">
            {{ expiryLabel(request.deadline, now) }}
          </span>
        </template>
      </StickyNote>

      <div v-if="isOwner" class="request-actions">
        <p v-if="request.status === 'expired'" class="text-muted">
          No seller listed anything within 24 hours. Repost it for another 24 hours, or remove it.
        </p>
        <p v-else-if="request.status === 'fulfilled'" class="text-muted">
          You booked one of the listings below, so this request is closed.
        </p>
        <p v-else class="text-muted">
          Sellers see this on the Wanted tab. When one lists something for it, you get a message.
        </p>
        <div class="action-row">
          <button
            v-if="request.status === 'expired'"
            class="btn btn-primary"
            :disabled="busy"
            @click="repost"
          >
            {{ busy ? 'Reposting...' : 'Repost for 24 hours' }}
          </button>
          <button v-if="request.status !== 'fulfilled'" class="btn btn-outline" :disabled="busy" @click="remove">
            Remove
          </button>
        </div>
      </div>

      <div v-else-if="request.status === 'active'" class="request-actions">
        <router-link
          :to="{ name: 'create-listing', query: { request: request.id } }"
          class="btn btn-primary btn-wide"
        >
          I have this: list it
        </router-link>
        <p class="text-muted">
          Your listing goes on the marketplace as usual, and @{{ request.requester?.username || 'the requester' }} gets a message about it.
        </p>
      </div>

      <p v-else class="text-muted request-actions">This request is closed.</p>

      <section class="offers" aria-labelledby="offers-title">
        <h2 id="offers-title" class="section-title">
          Listed for this request<span v-if="listings.length"> ({{ listings.length }})</span>
        </h2>
        <ul v-if="listings.length" class="note-board">
          <li v-for="item in listings" :key="item.id">
            <StickyNote
              :title="item.title"
              :body="item.description"
              :seed="item.id"
              :photo="photoUrl(item)"
              :photo-alt="item.title"
              :to="{ name: 'store-item', params: { id: item.id } }"
            >
              <template #footer>
                <DistanceTag :distance="item.distance" :show-unknown="showUnknownListings" />
                <span>@{{ item.seller?.username || 'someone' }}</span>
                <span v-if="listingPriceLabel(item)">{{ listingPriceLabel(item) }}</span>
                <span v-if="item.deadline && expiryLabel(item.deadline, now)">{{ expiryLabel(item.deadline, now) }}</span>
              </template>
            </StickyNote>
          </li>
        </ul>
        <p v-else class="text-muted">Nothing listed yet.</p>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import DistanceTag from '@/components/DistanceTag.vue'
import { hasDistances } from '@/utils/distance'
import { expiryLabel } from '@/utils/gigNote'
import { listingPriceLabel } from '@/utils/listingNote'
import { setPageTitle } from '@/utils/pageTitle'

interface ItemRequest {
  id: number
  title: string
  description: string
  status: string
  deadline?: string
  requester_id: number
  requester?: { id: number; username: string }
}

interface StoreItem {
  id: number
  title: string
  description: string
  deadline?: string
  distance?: string
  price_type?: string
  fixed_price?: number
  starting_bid?: number
  current_bid?: number
  seller?: { id: number; username: string }
  images?: { id: number; url: string }[]
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const request = ref<ItemRequest | null>(null)
const listings = ref<StoreItem[]>([])
const loading = ref(true)
const error = ref('')
const busy = ref(false)

// Countdowns move without refetching
const now = ref(new Date())
let clock: ReturnType<typeof setInterval> | undefined

const showUnknownListings = computed(() => hasDistances(listings.value))
const isOwner = computed(() => request.value?.requester_id === authStore.user?.id)
const statusLabel = computed(() => {
  const status = request.value?.status ?? ''
  return status.charAt(0).toUpperCase() + status.slice(1)
})

const authHeaders = () => ({ Authorization: `Bearer ${authStore.token}` })

const photoUrl = (item: StoreItem) =>
  item.images?.length ? config.STORE_API_BASE_URL + item.images[0].url : ''

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const id = Number(route.params.id)
    if (!Number.isInteger(id) || id <= 0) throw new Error('Request not found')
    const [found, offers] = await Promise.all([
      fetch(`${config.STORE_API_URL}/requests/${id}`, { headers: authHeaders() }),
      fetch(`${config.STORE_API_URL}/requests/${id}/listings`, { headers: authHeaders() })
    ])
    if (!found.ok) throw new Error(found.status === 404 ? 'Request not found' : 'Failed to load the request')
    request.value = await found.json()
    listings.value = offers.ok ? await offers.json() : []
    setPageTitle(request.value?.title)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load the request'
  } finally {
    loading.value = false
  }
}

const repost = async () => {
  if (!request.value) return
  busy.value = true
  try {
    const response = await fetch(`${config.STORE_API_URL}/requests/${request.value.id}/repost`, {
      method: 'POST',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not repost the request. Please try again.')
    await load()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not repost the request. Please try again.')
  } finally {
    busy.value = false
  }
}

const remove = async () => {
  if (!request.value || !confirm(`Remove "${request.value.title}" for good?`)) return
  busy.value = true
  try {
    const response = await fetch(`${config.STORE_API_URL}/requests/${request.value.id}`, {
      method: 'DELETE',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not remove the request. Please try again.')
    router.push({ name: 'store', query: { tab: 'mine' } })
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not remove the request. Please try again.')
  } finally {
    busy.value = false
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

<style scoped>
.request-page {
  max-width: 720px;
  margin: 0 auto;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 44px;
  margin-bottom: 1rem;
  color: var(--color-primary, #4f46e5);
  text-decoration: none;
  font-weight: 500;
}

.request-note {
  margin-bottom: 1.25rem;
}

.note-status {
  align-self: flex-start;
  padding: 0.15rem 0.6rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(31, 41, 55, 0.12);
}

.note-kicker {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--note-muted);
}

.note-detail-headline {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1.2;
}

.note-detail-body {
  margin: 0;
  font-size: 1.15rem;
  line-height: 1.5;
  flex: 1;
}

.request-actions {
  margin-bottom: 2rem;
}

.request-actions p {
  margin: 0.5rem 0;
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.action-row .btn,
.btn-wide {
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.section-title {
  margin: 0 0 1rem;
  font-size: 1.15rem;
  font-weight: 600;
}

.note-board {
  list-style: none;
  margin: 0;
  padding: 0.5rem 0.25rem;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 1.75rem;
}

.text-muted {
  color: var(--color-text-light, #6b7280);
}

.alert-danger {
  padding: 0.75rem 1.25rem;
  border-radius: 0.25rem;
  color: #842029;
  background-color: #f8d7da;
  border: 1px solid #f5c2c7;
}

@media (max-width: 480px) {
  .note-detail-headline {
    font-size: 1.4rem;
  }

  .btn-wide,
  .action-row .btn {
    width: 100%;
  }

  .note-board {
    grid-template-columns: 1fr;
  }
}
</style>
