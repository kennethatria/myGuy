<template>
  <div class="board">
    <div class="board-content">
      <NearbyBanner :state="viewer.state.value" @request="viewer.request" />

      <p v-if="loading" class="board-status" role="status">Loading...</p>

      <div v-else-if="error" class="alert-danger" role="alert">
        {{ error }}
        <button @click="fetchTasks" class="btn btn-sm btn-outline">Retry</button>
      </div>

      <template v-else-if="paginatedResult">
        <ul v-if="paginatedResult.tasks.length > 0" class="note-board" aria-label="Open gigs">
          <li v-for="task in paginatedResult.tasks" :key="task.id">
            <StickyNote
              tone="gig"
              tape
              :seed="task.id"
              :to="{ name: 'task-detail', params: { id: task.id } }"
            >
              <template #header>
                <h3 class="note-headline">{{ task.title }}</h3>
                <p class="note-text">{{ task.description }}</p>
              </template>
              <template #footer>
                <span class="note-meta">{{ noteMeta(task, task.creator?.username || 'someone', showUnknownDistance, now) }}</span>
              </template>
            </StickyNote>
          </li>
        </ul>

        <EmptyState
          v-else
          emoji="🛠"
          title="No gigs on the board"
          text="Be the first to ask for a hand."
          :action="{ label: 'Post a gig', to: { name: 'create-task' } }"
        />
      </template>

      <nav v-if="paginatedResult && paginatedResult.total_pages > 1" aria-label="Board pages">
        <ul class="pagination">
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import NearbyBanner from '@/components/NearbyBanner.vue'
import EmptyState from '@/components/EmptyState.vue'
import { hasDistances } from '@/utils/distance'
import { useViewerLocation, nearParam } from '@/composables/useViewerLocation'
import { noteMeta } from '@/utils/gigNote'

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

<style scoped src="@/assets/board.css"></style>
