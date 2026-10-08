<template>
  <div class="detail-page">
    <p v-if="loading" class="detail-status-text" role="status">Loading the request...</p>

    <div v-else-if="error" class="detail-error" role="alert">{{ error }}</div>

    <template v-else-if="request">
      <DetailNote
        tone="want"
        :seed="request.id"
        :status="request.status !== 'active' ? statusLabel : ''"
        :title="request.title"
        :body="request.description"
        :person="request.requester ?? null"
        :meta="postedMeta(request.created_at, request.deadline, request.status === 'active', now)"
      />

      <template v-if="isOwner">
        <p v-if="request.status === 'expired'" class="detail-line">
          No seller listed anything within 24 hours. Repost it for another 24 hours, or remove it.
        </p>
        <p v-else-if="request.status === 'fulfilled'" class="detail-line">
          You booked one of the listings below, so this request is closed.
        </p>
        <p v-else class="detail-line">
          Sellers see this on the Wanted tab. When one lists something for it, you get a message.
        </p>
      </template>
      <p v-else-if="request.status !== 'active'" class="detail-line">This request is closed.</p>

      <section aria-labelledby="offers-title">
        <h2 id="offers-title" class="detail-section-title">
          Listed for this request<span v-if="listings.length"> ({{ listings.length }})</span>
        </h2>
        <ul v-if="listings.length" class="note-board">
          <li v-for="item in listings" :key="item.id">
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
                <span class="note-meta">{{ noteMeta(item, item.seller?.username || 'someone', showUnknownListings, now) }}</span>
              </template>
            </StickyNote>
          </li>
        </ul>
        <p v-else class="detail-line">Nothing listed yet.</p>
      </section>

      <ActionBar v-if="isOwner ? request.status !== 'fulfilled' : request.status === 'active'">
        <template v-if="isOwner">
          <p v-if="confirmingRemove" class="bar-message">Remove "{{ request.title }}" for good?</p>
          <p v-if="actionError" class="bar-error" role="alert">{{ actionError }}</p>
          <div v-if="confirmingRemove" class="bar-row">
            <button class="btn btn-danger" :disabled="busy" @click="remove">
              {{ busy ? 'Removing...' : 'Yes, remove' }}
            </button>
            <button class="btn btn-outline" :disabled="busy" @click="confirmingRemove = false">Keep it</button>
          </div>
          <template v-else>
            <button v-if="request.status === 'expired'" class="btn btn-primary" :disabled="busy" @click="repost">
              {{ busy ? 'Reposting...' : 'Repost for 24 hours' }}
            </button>
            <button class="btn btn-outline-danger" :disabled="busy" @click="confirmingRemove = true">
              Remove request
            </button>
          </template>
        </template>
        <template v-else>
          <router-link :to="{ name: 'create-listing', query: { request: request.id } }" class="btn btn-primary">
            I have this: list it
          </router-link>
          <p class="bar-caption">
            Your listing goes on the marketplace as usual, and @{{ request.requester?.username || 'the requester' }} gets a message about it.
          </p>
        </template>
      </ActionBar>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import DetailNote from '@/components/DetailNote.vue'
import ActionBar from '@/components/ActionBar.vue'
import { hasDistances } from '@/utils/distance'
import { noteMeta, postedMeta } from '@/utils/gigNote'
import { setPageTitle } from '@/utils/pageTitle'
import { trackEvent } from '@/utils/analytics'

interface ItemRequest {
  id: number
  title: string
  description: string
  status: string
  deadline?: string
  created_at: string
  requester_id: number
  requester?: { id: number; username: string }
}

interface StoreItem {
  id: number
  title: string
  description: string
  deadline?: string
  distance?: string
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
const confirmingRemove = ref(false)
const actionError = ref('')

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
  actionError.value = ''
  try {
    const response = await fetch(`${config.STORE_API_URL}/requests/${request.value.id}/repost`, {
      method: 'POST',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not repost the request. Please try again.')
    trackEvent('post-reposted', { kind: 'request' })
    await load()
  } catch (err) {
    actionError.value = err instanceof Error ? err.message : 'Could not repost the request. Please try again.'
  } finally {
    busy.value = false
  }
}

const remove = async () => {
  if (!request.value) return
  busy.value = true
  actionError.value = ''
  try {
    const response = await fetch(`${config.STORE_API_URL}/requests/${request.value.id}`, {
      method: 'DELETE',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not remove the request. Please try again.')
    router.push({ name: 'store', query: { tab: 'mine' } })
  } catch (err) {
    actionError.value = err instanceof Error ? err.message : 'Could not remove the request. Please try again.'
    confirmingRemove.value = false
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

<style scoped src="@/assets/board.css"></style>
<style scoped src="@/assets/detail.css"></style>
