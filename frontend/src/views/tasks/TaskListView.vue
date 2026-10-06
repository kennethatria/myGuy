<template>
  <div class="container py-4">
    <div class="board-header">
      <div>
        <h1 class="text-2xl font-semibold">Gig Board</h1>
        <p class="text-muted mt-1">Short notes from people who need a hand. Each stays up for 24 hours.</p>
      </div>
      <router-link :to="{ name: 'create-task' }" class="btn btn-primary">
        Post a Gig
      </router-link>
    </div>

    <NearbyBanner :state="viewer.state.value" @request="viewer.request" />

    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else-if="error" class="alert alert-danger" role="alert">
      {{ error }}
      <button @click="fetchTasks" class="btn btn-sm btn-outline ms-3">Retry</button>
    </div>

    <template v-else-if="paginatedResult">
      <ul v-if="paginatedResult.tasks.length > 0" class="note-board" aria-label="Open gigs">
        <li v-for="task in paginatedResult.tasks" :key="task.id">
          <StickyNote
            :title="task.title"
            :body="task.description"
            :seed="task.id"
            :to="{ name: 'task-detail', params: { id: task.id } }"
          >
            <template #footer>
              <DistanceTag :distance="task.distance" :show-unknown="showUnknownDistance" />
              <span>@{{ task.creator?.username || 'someone' }}</span>
              <span v-if="expiryLabel(task.deadline, now)">{{ expiryLabel(task.deadline, now) }}</span>
            </template>
          </StickyNote>
        </li>
      </ul>

      <div v-else class="empty-board">
        <h2 class="h5">No notes on the board</h2>
        <p class="text-muted">
          Be the first to ask for a hand.
        </p>
        <router-link :to="{ name: 'create-task' }" class="btn btn-primary mt-2">Post a Gig</router-link>
      </div>
    </template>

    <nav v-if="paginatedResult && paginatedResult.total_pages > 1" class="mt-4" aria-label="Board pages">
      <ul class="pagination justify-content-center">
        <li class="page-item" :class="{ disabled: currentPage === 1 }">
          <button class="page-link" @click="goToPage(currentPage - 1)" :disabled="currentPage === 1">
            Previous
          </button>
        </li>
        <li
          v-for="page in visiblePages"
          :key="page"
          class="page-item"
          :class="{ active: page === currentPage }"
        >
          <button
            class="page-link"
            :aria-current="page === currentPage ? 'page' : undefined"
            @click="typeof page === 'number' && goToPage(page)"
          >
            {{ page }}
          </button>
        </li>
        <li class="page-item" :class="{ disabled: currentPage === paginatedResult.total_pages }">
          <button
            class="page-link"
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === paginatedResult.total_pages"
          >
            Next
          </button>
        </li>
      </ul>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import NearbyBanner from '@/components/NearbyBanner.vue'
import DistanceTag from '@/components/DistanceTag.vue'
import { hasDistances } from '@/utils/distance'
import { useViewerLocation, nearParam } from '@/composables/useViewerLocation'
import { expiryLabel } from '@/utils/gigNote'

interface Task {
  id: number
  title: string
  description: string
  status: string
  deadline: string
  // Rough distance from the viewer ("~2 km"), when both have a location
  distance?: string
  creator?: {
    id: number
    username: string
  }
}

interface PaginatedResult {
  tasks: Task[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

const authStore = useAuthStore()

const loading = ref(false)
const error = ref('')
const paginatedResult = ref<PaginatedResult | null>(null)
const currentPage = ref(1)
const perPage = 24
const showUnknownDistance = computed(() => hasDistances(paginatedResult.value?.tasks ?? []))
// Nearest first once the viewer's rough location is known
const viewer = useViewerLocation(() => {
  currentPage.value = 1
  fetchTasks()
})
// No sort picker: nearest first when the viewer's area is known, else newest
const sortBy = computed(() => (viewer.location.value ? 'distance' : 'created_at'))

// Countdowns move without refetching
const now = ref(new Date())
let clock: ReturnType<typeof setInterval> | undefined

const visiblePages = computed(() => {
  if (!paginatedResult.value) return []

  const total = paginatedResult.value.total_pages
  const current = currentPage.value
  const delta = 2
  const range: number[] = []
  const rangeWithDots: (number | string)[] = []
  let l: number | undefined

  for (let i = 1; i <= total; i++) {
    if (i === 1 || i === total || (i >= current - delta && i <= current + delta)) {
      range.push(i)
    }
  }

  range.forEach((i) => {
    if (l !== undefined) {
      if (i - l === 2) {
        rangeWithDots.push(l + 1)
      } else if (i - l !== 1) {
        rangeWithDots.push('...')
      }
    }
    rangeWithDots.push(i)
    l = i
  })

  return rangeWithDots
})

const buildQueryParams = () => {
  const params = new URLSearchParams()

  // The board shows live notes from other people
  params.append('status', 'open')
  if (authStore.user?.id) {
    params.append('exclude_created_by', String(authStore.user.id))
  }
  // The viewer's rough location sorts (or just tags) the notes by distance
  if (viewer.location.value) {
    params.append('near', nearParam(viewer.location.value))
  }
  params.append('sort_by', sortBy.value)
  params.append('sort_order', 'desc')
  params.append('page', String(currentPage.value))
  params.append('per_page', String(perPage))

  return params
}

const fetchTasks = async () => {
  loading.value = true
  error.value = ''

  try {
    const response = await fetch(`${config.API_URL}/tasks?${buildQueryParams()}`, {
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      }
    })

    if (!response.ok) {
      throw new Error('Failed to load gigs')
    }

    paginatedResult.value = await response.json()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load gigs'
    console.error('Error fetching tasks:', err)
  } finally {
    loading.value = false
  }
}

const goToPage = (page: number) => {
  if (page >= 1 && page <= (paginatedResult.value?.total_pages || 1)) {
    currentPage.value = page
    fetchTasks()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

onMounted(() => {
  fetchTasks()
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
  margin-bottom: 1.25rem;
}
.note-board {
  list-style: none;
  margin: 0;
  padding: 0.5rem 0.25rem;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 1.75rem;
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

.page-link:hover {
  color: var(--color-primary-dark);
  background-color: #e9ecef;
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
  .note-board {
    grid-template-columns: 1fr;
  }
}
</style>
