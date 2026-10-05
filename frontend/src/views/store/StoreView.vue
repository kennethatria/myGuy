<template>
  <div class="container py-4">
    <div class="board-header">
      <div>
        <h1 class="text-2xl font-semibold">Marketplace</h1>
        <p class="text-muted mt-1">Things people are selling. Each note stays up for 24 hours.</p>
      </div>
      <router-link :to="{ name: 'create-listing' }" class="btn btn-primary">
        Post a Listing
      </router-link>
    </div>

    <div class="board-tabs" role="tablist" aria-label="Marketplace views">
      <button
        role="tab"
        :aria-selected="view === 'board'"
        :class="['board-tab', { active: view === 'board' }]"
        @click="showView('board')"
      >
        Board
      </button>
      <button
        role="tab"
        :aria-selected="view === 'mine'"
        :class="['board-tab', { active: view === 'mine' }]"
        @click="showView('mine')"
      >
        Your listings
      </button>
    </div>

    <template v-if="view === 'board'">
      <div class="board-controls">
        <label class="visually-hidden" for="listing-search">Search listings</label>
        <input
          id="listing-search"
          v-model="searchQuery"
          type="search"
          class="board-input"
          placeholder="Search notes..."
          @input="debouncedSearch"
        />
        <label class="visually-hidden" for="listing-sort">Sort notes</label>
        <select id="listing-sort" v-model="sortBy" class="board-input board-sort" @change="applySort">
          <option value="created_at">Newest first</option>
          <option value="deadline">Expiring soon</option>
        </select>
      </div>

      <div v-if="loading" class="text-center py-5">
        <div class="spinner-border" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
      </div>

      <div v-else-if="error" class="alert alert-danger" role="alert">
        {{ error }}
        <button @click="fetchItems" class="btn btn-sm btn-outline ms-3">Retry</button>
      </div>

      <template v-else>
        <ul v-if="items.length > 0" class="note-board" aria-label="Listings">
          <li v-for="item in items" :key="item.id">
            <StickyNote
              :title="item.title"
              :body="item.description"
              :seed="item.id"
              :photo="photoUrl(item)"
              :photo-alt="item.title"
              :to="{ name: 'store-item', params: { id: item.id } }"
            >
              <template #footer>
                <span>@{{ item.seller?.username || 'someone' }}</span>
                <span v-if="listingPriceLabel(item)">{{ listingPriceLabel(item) }}</span>
                <span v-if="item.deadline && expiryLabel(item.deadline, now)">{{ expiryLabel(item.deadline, now) }}</span>
              </template>
            </StickyNote>
          </li>
        </ul>

        <div v-else class="empty-board">
          <h2 class="h5">No notes on the board</h2>
          <p class="text-muted">
            {{ searchQuery ? 'Nothing matches that search.' : 'Be the first to sell something.' }}
          </p>
          <router-link :to="{ name: 'create-listing' }" class="btn btn-primary mt-2">Post a Listing</router-link>
        </div>
      </template>

      <nav v-if="totalPages > 1" class="mt-4" aria-label="Board pages">
        <ul class="pagination justify-content-center">
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
    </template>

    <template v-else>
      <div v-if="loading" class="text-center py-5">
        <div class="spinner-border" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
      </div>

      <div v-else-if="error" class="alert alert-danger" role="alert">
        {{ error }}
        <button @click="fetchMine" class="btn btn-sm btn-outline ms-3">Retry</button>
      </div>

      <ul v-else-if="myItems.length > 0" class="note-board" aria-label="Your listings">
        <li v-for="item in myItems" :key="item.id">
          <StickyNote :seed="item.id" :photo="photoUrl(item)" :photo-alt="item.title">
            <template #header>
              <h3 class="note-headline">
                <router-link :to="{ name: 'store-item', params: { id: item.id } }" class="note-title-link">
                  {{ item.title }}
                </router-link>
              </h3>
              <p class="note-text">{{ item.description }}</p>
            </template>
            <template #footer>
              <span>{{ statusLabel(item) }}</span>
              <span v-if="item.status === 'expired'" class="note-actions">
                <button
                  class="btn btn-sm btn-primary"
                  :disabled="busyId === item.id"
                  @click="repost(item)"
                >
                  {{ busyId === item.id ? 'Reposting...' : 'Repost' }}
                </button>
                <button
                  class="btn btn-sm btn-outline"
                  :disabled="busyId === item.id"
                  @click="remove(item)"
                >
                  Remove
                </button>
              </span>
            </template>
          </StickyNote>
        </li>
      </ul>

      <div v-else class="empty-board">
        <h2 class="h5">You haven't listed anything</h2>
        <router-link :to="{ name: 'create-listing' }" class="btn btn-primary mt-2">Post a Listing</router-link>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { debounce } from 'lodash-es'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import { expiryLabel } from '@/utils/gigNote'
import { listingPriceLabel } from '@/utils/listingNote'

interface StoreItem {
  id: number
  title: string
  description: string
  status: string
  deadline?: string
  price_type?: string
  fixed_price?: number
  starting_bid?: number
  current_bid?: number
  seller?: { id: number; username: string }
  images?: { id: number; url: string }[]
}

const authStore = useAuthStore()

const view = ref<'board' | 'mine'>('board')
const loading = ref(false)
const error = ref('')
const items = ref<StoreItem[]>([])
const myItems = ref<StoreItem[]>([])
const total = ref(0)
const searchQuery = ref('')
const currentPage = ref(1)
const perPage = 24
const sortBy = ref<'created_at' | 'deadline'>('created_at')
const busyId = ref<number | null>(null)
const totalPages = computed(() => Math.ceil(total.value / perPage))

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

const fetchItems = async () => {
  loading.value = true
  error.value = ''

  // The board shows live notes, newest or soonest to come down first
  const params = new URLSearchParams({
    status: 'active',
    sort_by: sortBy.value,
    sort_order: sortBy.value === 'deadline' ? 'asc' : 'desc',
    page: String(currentPage.value),
    per_page: String(perPage)
  })
  if (searchQuery.value) params.append('search', searchQuery.value)
  // Others' notes only; yours are under "Your listings"
  if (authStore.user?.id) params.append('exclude_seller_id', String(authStore.user.id))

  try {
    const response = await fetch(`${config.STORE_API_URL}/items?${params}`, { headers: authHeaders() })
    if (!response.ok) throw new Error('Failed to load listings')
    const data = await response.json()
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load listings'
  } finally {
    loading.value = false
  }
}

const fetchMine = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await fetch(`${config.STORE_API_URL}/user/listings`, { headers: authHeaders() })
    if (!response.ok) throw new Error('Failed to load your listings')
    myItems.value = await response.json()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load your listings'
  } finally {
    loading.value = false
  }
}

const showView = (next: 'board' | 'mine') => {
  view.value = next
  if (next === 'board') {
    fetchItems()
  } else {
    fetchMine()
  }
}

// Put an expired note back on the board for a fresh 24 hours
const repost = async (item: StoreItem) => {
  busyId.value = item.id
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${item.id}/repost`, {
      method: 'POST',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not repost the listing. Please try again.')
    await fetchMine()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not repost the listing. Please try again.')
  } finally {
    busyId.value = null
  }
}

const remove = async (item: StoreItem) => {
  if (!confirm(`Remove "${item.title}" for good?`)) return
  busyId.value = item.id
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${item.id}`, {
      method: 'DELETE',
      headers: authHeaders()
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'Could not remove the listing. Please try again.')
    myItems.value = myItems.value.filter((mine) => mine.id !== item.id)
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not remove the listing. Please try again.')
  } finally {
    busyId.value = null
  }
}

const debouncedSearch = debounce(() => {
  currentPage.value = 1
  fetchItems()
}, 300)

const applySort = () => {
  currentPage.value = 1
  fetchItems()
}

const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    fetchItems()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

onMounted(() => {
  fetchItems()
  clock = setInterval(() => { now.value = new Date() }, 60_000)
})

onUnmounted(() => {
  if (clock) clearInterval(clock)
})
</script>

<style scoped>
.container {
  max-width: 1200px;
  margin: 0 auto;
}

.board-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.board-tabs {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 1.25rem;
  border-bottom: 1px solid var(--color-border, #e5e7eb);
}

.board-tab {
  min-height: 44px;
  padding: 0.5rem 0.9rem;
  border: none;
  border-bottom: 2px solid transparent;
  background: none;
  font-size: 1rem;
  color: var(--color-text-light, #6b7280);
  cursor: pointer;
}

.board-tab.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
  font-weight: 600;
}

.board-tab:focus-visible {
  outline: 3px solid var(--color-primary, #4f46e5);
  outline-offset: 2px;
}

.board-controls {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.board-input {
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 0.375rem;
  font-size: 1rem;
  background: #fff;
}

.board-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.15);
}

.board-controls .board-input:first-of-type {
  flex: 1;
  min-width: 0;
}

.board-sort {
  flex: 0 0 auto;
}

.note-board {
  list-style: none;
  margin: 0;
  padding: 0.5rem 0.25rem;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 1.75rem;
}

/* The title link covers the whole note (see .note-title-link::after) */
.note-board > li {
  position: relative;
}

.note-headline {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 700;
  line-height: 1.25;
}

.note-title-link {
  color: inherit;
  font-weight: inherit;
  text-decoration: none;
}

/* Tap anywhere on your note to open it, like on the board */
.note-title-link::after {
  content: '';
  position: absolute;
  inset: 0;
}

.note-title-link:hover,
.note-title-link:focus-visible {
  text-decoration: underline;
}

.note-text {
  margin: 0;
  font-size: 0.95rem;
  line-height: 1.45;
  flex: 1;
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

.empty-board {
  text-align: center;
  padding: 3rem 1rem;
  border: 2px dashed var(--color-border, #e5e7eb);
  border-radius: 8px;
}

.alert {
  padding: 0.75rem 1.25rem;
  border-radius: 0.25rem;
}

.alert-danger {
  color: #842029;
  background-color: #f8d7da;
  border: 1px solid #f5c2c7;
}

.text-muted {
  color: var(--color-text-light, #6b7280);
}

.pagination {
  display: flex;
  justify-content: center;
  padding-left: 0;
  list-style: none;
}

.page-item:not(:first-child) .page-link {
  margin-left: -1px;
}

.page-link {
  display: block;
  padding: 0.375rem 0.75rem;
  color: var(--color-primary);
  background-color: #fff;
  border: 1px solid #dee2e6;
}

.page-item.active .page-link {
  color: #fff;
  background-color: var(--color-primary);
  border-color: var(--color-primary);
}

.page-item.disabled .page-link {
  color: #6c757d;
  pointer-events: none;
}

@media (max-width: 480px) {
  .board-controls {
    flex-direction: column;
  }

  .note-board {
    grid-template-columns: 1fr;
  }
}
</style>
