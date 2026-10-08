<template>
  <div class="mine">
    <p v-if="isLoading" class="mine-status" role="status">Loading...</p>

    <div v-else-if="error" class="mine-error" role="alert">
      <p>{{ error }}</p>
      <button v-if="error.includes('log in')" @click="redirectToLogin" class="btn btn-primary">Log in</button>
      <button @click="fetchMyGigs" class="btn btn-outline">Retry</button>
    </div>

    <!-- My stuff: the gigs you posted, live or ended -->
    <template v-else-if="activeTab === 'created'">
      <div class="mine-tabs" role="tablist" aria-label="Which gigs">
        <button role="tab" :aria-selected="stuffTab === 'live'" :class="['mine-tab', { active: stuffTab === 'live' }]" @click="stuffTab = 'live'">
          Live {{ liveTasks.length }}
        </button>
        <button role="tab" :aria-selected="stuffTab === 'ended'" :class="['mine-tab', { active: stuffTab === 'ended' }]" @click="stuffTab = 'ended'">
          Ended {{ endedTasks.length }}
        </button>
      </div>

      <EmptyState
        v-if="createdTasks.length === 0"
        emoji="📝"
        title="No gigs yet"
        text="Gigs you post show up here, with who applied."
        :action="{ label: 'Post a gig', to: { name: 'create-task' } }"
      />
      <p v-else-if="shownTasks.length === 0" class="mine-status">
        {{ stuffTab === 'live' ? 'Nothing live right now.' : 'Nothing has ended.' }}
      </p>
      <ul v-else class="mine-list" aria-label="Gigs you posted">
        <li v-for="task in shownTasks" :key="task.id">
          <StickyNote tone="gig" :seed="task.id">
            <template #header>
              <span class="card-top">
                <span class="kind-chip kind-gig">🛠 Gig</span>
                <span class="card-status">{{ statusText(task.status) }}</span>
              </span>
              <h3 class="card-title">
                <router-link :to="{ name: 'task-detail', params: { id: task.id } }" class="card-link">{{ task.title }}</router-link>
              </h3>
              <p class="card-text">{{ task.description }}</p>
            </template>
            <template #footer>
              <span class="card-meta">{{ createdMeta(task) }}</span>
              <span v-if="confirmingId === task.id" class="card-actions">
                <button class="btn btn-danger btn-sm" :disabled="busyId === task.id" @click="remove(task.id)">
                  {{ busyId === task.id ? 'Removing...' : 'Yes, remove' }}
                </button>
                <button class="btn btn-outline btn-sm" :disabled="busyId === task.id" @click="confirmingId = null">Keep it</button>
              </span>
              <span v-else class="card-actions">
                <button v-if="canReview(task)" class="btn btn-primary btn-sm" @click="review(task)">Review</button>
                <button v-if="task.status === 'expired' || task.status === 'cancelled'" class="btn btn-primary btn-sm" :disabled="busyId === task.id" @click="repost(task.id)">
                  {{ busyId === task.id ? 'Reposting...' : 'Repost' }}
                </button>
                <button v-if="canRemove(task)" class="link-danger" @click="confirmingId = task.id">Remove</button>
              </span>
              <span v-if="cardError[task.id]" class="card-error" role="alert">{{ cardError[task.id] }}</span>
            </template>
          </StickyNote>
        </li>
      </ul>
    </template>

    <!-- Assignments: gigs you're doing -->
    <template v-else-if="activeTab === 'assigned'">
      <EmptyState
        v-if="assignedTasks.length === 0"
        emoji="🤝"
        title="No assignments yet"
        text="Gigs you're picked for show up here."
        :action="{ label: 'Browse gigs', to: { name: 'tasks' } }"
      />
      <ul v-else class="mine-list" aria-label="Gigs you're doing">
        <li v-for="task in assignedTasks" :key="task.id">
          <StickyNote tone="gig" :seed="task.id">
            <template #header>
              <span class="card-top">
                <span class="kind-chip kind-gig">🛠 Gig</span>
                <span class="card-status">{{ statusText(task.status) }}</span>
              </span>
              <h3 class="card-title">
                <router-link :to="{ name: 'task-detail', params: { id: task.id } }" class="card-link">{{ task.title }}</router-link>
              </h3>
              <p class="card-text">{{ task.description }}</p>
            </template>
            <template #footer>
              <span class="card-meta">Posted by @{{ task.creator?.username || 'someone' }}</span>
              <span class="card-actions">
                <button class="btn btn-primary btn-sm" @click="chatWith(task.id, task.created_by, task.creator?.username)">Open chat</button>
              </span>
            </template>
          </StickyNote>
        </li>
      </ul>
    </template>

    <!-- Applications: gigs you applied for, and how it went -->
    <template v-else>
      <EmptyState
        v-if="activeApplications.length === 0"
        title="No applications yet"
        text="Gigs you apply for will show up here, along with whether you got them."
        :action="{ label: 'Browse gigs', to: { name: 'tasks' } }"
        :secondary="{ label: 'Post your own gig', to: { name: 'create-task' } }"
      />
      <ul v-else class="mine-list" aria-label="Gigs you applied for">
        <li v-for="application in activeApplications" :key="application.id">
          <StickyNote
            tone="gig"
            :seed="application.task_id"
            :class="{ 'card-closed': application.status === 'declined' }"
          >
            <template #header>
              <span class="card-top">
                <span class="kind-chip kind-gig">🛠 Gig</span>
                <span :class="['card-status', 'with-dot', `dot-${application.status}`]">{{ applicationStatusLabel[application.status] }}</span>
              </span>
              <h3 class="card-title">
                <router-link :to="{ name: 'task-detail', params: { id: application.task_id } }" class="card-link">{{ application.task.title }}</router-link>
              </h3>
              <p class="card-text">{{ APPLICATION_NOTES[application.status] }}</p>
            </template>
            <template #footer>
              <span class="card-meta">
                {{ application.status === 'pending' ? `Applied ${timeAgo(application.created_at)}` : `Posted by @${application.task.creator?.username || 'someone'}` }}
              </span>
              <span v-if="application.status === 'accepted' && application.task.creator" class="card-actions">
                <button class="btn btn-primary btn-sm" @click="chatWith(application.task_id, application.task.creator.id, application.task.creator.username)">
                  Open chat
                </button>
              </span>
            </template>
          </StickyNote>
        </li>
      </ul>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'
import { timeLeft } from '@/utils/gigNote'
import { timeAgo } from '@/utils/conversationStatus'
import { useChatStore } from '@/stores/chat'
import StickyNote from '@/components/StickyNote.vue'
import EmptyState from '@/components/EmptyState.vue'

const tasksStore = useTasksStore()
const router = useRouter()
const route = useRoute()
const isLoading = ref(false)
const error = ref('')

// Which list: chosen from the side navigation (/my-gigs/created etc.)
type Tab = 'created' | 'assigned' | 'applications'
const TABS: Tab[] = ['created', 'assigned', 'applications']
const activeTab = computed<Tab>(() => {
  const tab = route.params.tab as Tab
  return TABS.includes(tab) ? tab : 'created'
})

const applicationStatusLabel = {
  pending: 'Pending',
  accepted: 'Approved',
  declined: 'Not selected'
} as const

// Declined applicants get no reason, only that it didn't work out
const APPLICATION_NOTES = {
  pending: 'Waiting for the poster to answer.',
  accepted: 'The poster approved you.',
  declined: "This one didn't work out."
} as const

const STATUS_TEXT: Record<string, string> = {
  open: '🟢 Open',
  in_progress: 'In progress',
  pending_approval: 'Awaiting approval',
  expired: 'Expired',
  cancelled: 'Cancelled'
}
const statusText = (status: string) => STATUS_TEXT[status] ?? status

// Applications on the user's own gig still waiting for their decision
const pendingCount = (task: { applications?: { status?: string }[] }) =>
  (task.applications || []).filter(app => app.status === 'pending').length

const redirectToLogin = () => {
  const authStore = useAuthStore()
  authStore.logout() // Clear any existing auth state
  router.push({ name: 'login' })
}

// Completed gigs leave these lists; their history stays in the conversation
// (marked expired) and their pages still open from links
const notCompleted = <T extends { status?: string }>(tasks: T[]) => tasks.filter(task => task.status !== 'completed')

const createdTasks = computed(() => notCompleted(tasksStore.userTasks || []))

// Live: on the board or under way. Ended: expired or cancelled, to repost
// or remove (completed ones have left the list).
type CreatedTask = (typeof createdTasks.value)[number]
const stuffTab = ref<'live' | 'ended'>('live')
const isEnded = (task: CreatedTask) => task.status === 'expired' || task.status === 'cancelled'
const liveTasks = computed(() => createdTasks.value.filter(task => !isEnded(task)))
const endedTasks = computed(() => createdTasks.value.filter(isEnded))
const shownTasks = computed(() => (stuffTab.value === 'live' ? liveTasks.value : endedTasks.value))

const createdMeta = (task: CreatedTask) => {
  const applied = task.applications?.length ?? 0
  const parts = [`${applied} ${applied === 1 ? 'applicant' : 'applicants'}`]
  const left = task.status === 'open' ? timeLeft(task.deadline) : ''
  if (left) parts.push(`${left} left`)
  return parts.join(' · ')
}

// Server-side filtering already excludes self-assigned tasks
const assignedTasks = computed(() => notCompleted(tasksStore.assignedTasks || []))

const activeApplications = computed(() =>
  (tasksStore.myApplications || []).filter(application => application.task?.status !== 'completed'))

const chatStore = useChatStore()
const chatWith = (taskId: number, otherUserId: number, otherUserName?: string) =>
  chatStore.openChat({ taskId, otherUserId, otherUserName })

// Review: the conversation where the next step is. The person doing it, or
// the one applicant; with several applicants, the list of conversations.
const canReview = (task: CreatedTask) => !!task.assigned_to || pendingCount(task) > 0
const review = (task: CreatedTask) => {
  if (task.assigned_to) return chatWith(task.id, task.assigned_to, task.assignee?.username)
  const waiting = (task.applications || []).filter(app => app.status === 'pending')
  if (waiting.length === 1 && waiting[0].applicant) return chatWith(task.id, waiting[0].applicant.id, waiting[0].applicant.username)
  return chatStore.openChat()
}

// As on the gig's page: remove it while nobody is doing it
const canRemove = (task: CreatedTask) => !task.assigned_to && ['open', 'expired', 'cancelled'].includes(task.status)

const busyId = ref<number | null>(null)
const confirmingId = ref<number | null>(null)
const cardError = ref<Record<number, string>>({})

const runOn = async (taskId: number, step: () => Promise<unknown>, failure: string) => {
  busyId.value = taskId
  cardError.value = { ...cardError.value, [taskId]: '' }
  try {
    await step()
    await tasksStore.fetchUserTasks()
  } catch (err) {
    cardError.value = { ...cardError.value, [taskId]: err instanceof Error && err.message ? err.message : failure }
  } finally {
    busyId.value = null
    confirmingId.value = null
  }
}

// Put an expired note back on the board for a fresh 24 hours
const repost = (taskId: number) =>
  runOn(taskId, () => tasksStore.updateTaskStatus(taskId, 'open'), 'Could not repost the note. Please try again.')

// Anyone who applied is told in Messages (the backend does that)
const remove = (taskId: number) =>
  runOn(taskId, () => tasksStore.deleteTask(taskId), 'Could not remove the gig. Please try again.')

const fetchMyGigs = async () => {
  isLoading.value = true
  error.value = ''
  
  try {
    // Make sure we have a valid token before trying to fetch data
    const authStore = useAuthStore()
    if (!authStore.token) {
      error.value = 'Please log in to view your gigs.'
      isLoading.value = false
      return
    }
    
    // Check authentication status
    const isAuthenticated = await authStore.checkAuth()
    if (!isAuthenticated) {
      error.value = 'Your session has expired. Please log in again.'
      isLoading.value = false
      return
    }
    
    // Fetch real data from API
    await Promise.all([
      tasksStore.fetchUserTasks(),
      tasksStore.fetchAssignedTasks(),
      // A failure here shouldn't hide the other lists
      tasksStore.fetchMyApplications().catch(err => console.error('Failed to load applications:', err))
    ])
  } catch (err: unknown) {
    console.error('Failed to fetch gigs:', err)

    // Check if it's an authentication error
    const errMessage = err instanceof Error ? err.message : ''
    if (errMessage && errMessage.includes('log in again')) {
      error.value = errMessage
    } else {
      error.value = 'Failed to load your gigs. Please try again later.'
    }
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  await fetchMyGigs()
})
</script>

<style scoped>
.mine {
  max-width: 960px;
  margin: 0 auto;
  padding-bottom: 1.5rem;
}

.mine-status {
  padding: 2rem 20px;
  text-align: center;
  color: var(--text-muted);
}

.mine-error {
  margin: 16px 20px;
  padding: 12px 14px;
  border: 1px solid #f5c2c7;
  border-radius: 12px;
  background: #f8d7da;
  color: #842029;
}

.mine-error .btn {
  margin: 8px 8px 0 0;
}

/* Live / Ended */
.mine-tabs {
  display: flex;
  gap: 24px;
  padding: 0 20px;
  border-bottom: 1px solid var(--border);
}

.mine-tab {
  height: 44px;
  padding: 0;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--text-muted);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
}

.mine-tab.active {
  border-bottom-color: var(--accent);
  color: var(--text);
  font-weight: 600;
}

.mine-list {
  list-style: none;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 18px;
  margin: 0;
  padding: 20px 24px 0;
}

/* The title link covers the card; buttons sit above it */
.mine-list > li {
  position: relative;
}

.card-link {
  color: inherit;
  font-weight: inherit;
}

.card-link::after {
  content: '';
  position: absolute;
  inset: 0;
}

.card-link:hover,
.card-link:focus-visible {
  color: inherit;
  text-decoration: underline;
}

.card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.kind-chip {
  padding: 2px 8px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.65);
  font-size: 11px;
  font-weight: 700;
}

.kind-gig {
  color: #1E3A8A;
}

.card-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.7);
  color: #065F46;
  font-size: 12px;
  font-weight: 600;
}

.with-dot::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 4px;
}

.dot-pending { color: #92400E; }
.dot-pending::before { background: #F59E0B; }
.dot-accepted { color: #166534; }
.dot-accepted::before { background: #16A34A; }
.dot-declined { color: #4B5563; }
.dot-declined::before { background: #9CA3AF; }

.card-title {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  line-height: 1.3;
}

.card-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.45;
  color: #374151;
}

.card-meta {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--text-body);
}

.card-actions {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-actions .btn {
  min-height: 36px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
}

.link-danger {
  min-height: 36px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  color: #B91C1C;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.card-error {
  flex: 1 1 100%;
  color: #B91C1C;
  font-size: 13px;
}

/* Not selected: greyed out */
.card-closed.card-closed {
  --note-bg: #EDEDEA;
}

.card-closed .card-title,
.card-closed .card-text {
  color: var(--text-muted);
}
</style>
