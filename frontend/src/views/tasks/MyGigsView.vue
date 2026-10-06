<template>
  <div class="dashboard-container my-gigs">
    <!-- Page Header -->
    <div class="page-header">
      <h1 class="page-title">{{ pageTitle }}</h1>
    </div>

    <!-- Loading and error states -->
    <div v-if="isLoading" class="card p-4 mb-4 text-center">
      <div class="loading-spinner mb-2"></div>
      <p>Loading...</p>
    </div>

    <div v-else-if="error" class="card p-4 mb-4 bg-red-100 text-danger">
      <p>{{ error }}</p>
      <div class="mt-2">
        <button 
          v-if="error.includes('log in')" 
          @click="redirectToLogin" 
          class="btn btn-primary mr-2"
        >
          Log In
        </button>
        <button @click="fetchMyGigs" class="btn btn-outline">Retry</button>
      </div>
    </div>

    <div v-else>
      <div class="tabs-section">
        <!-- Tab Content -->
        <div class="tab-content">
          <!-- Created Gigs Tab -->
          <div v-if="activeTab === 'created'" class="tab-pane">
            <div v-if="createdTasks.length === 0" class="empty-state">
              <div class="empty-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none">
                  <path d="M14 2H6C4.9 2 4 2.9 4 4V20C4 21.1 4.89 22 5.99 22H18C19.1 22 20 21.1 20 20V8L14 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <polyline points="14,2 14,8 20,8" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <line x1="16" y1="13" x2="8" y2="13" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <line x1="16" y1="17" x2="8" y2="17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <polyline points="10,9 9,9 8,9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <h3>No gigs created yet</h3>
              <p>Start by posting your first gig and connect with talented freelancers.</p>
              <router-link :to="{ name: 'create-task' }" class="btn btn-primary">Post Your First Gig</router-link>
            </div>
            <div v-else class="task-list">
              <div 
                v-for="task in createdTasks" 
                :key="task.id"
                class="task-item"
                @click="navigateToTask(task.id)"
              >
                <div class="task-header">
                  <h3 class="task-title">{{ task.title }}</h3>
                  <span class="badge" :class="'badge-' + task.status">
                    {{ task.status.replace('_', ' ') }}
                  </span>
                </div>
                <p class="task-description">{{ task.description }}</p>
                <div class="task-footer">
                  <div class="task-meta">
                    <span v-if="task.status === 'expired'" class="task-deadline">No replies within 24 hours</span>
                    <span v-else-if="task.status === 'open' && expiryLabel(task.deadline)" class="task-deadline">
                      {{ expiryLabel(task.deadline) }}
                    </span>
                    <span v-else class="task-deadline">Posted {{ formatDate(task.created_at) }}</span>
                  </div>
                  <button
                    v-if="task.status === 'expired'"
                    class="btn btn-primary btn-sm"
                    :disabled="repostingId === task.id"
                    @click.stop="repost(task.id)"
                  >
                    {{ repostingId === task.id ? 'Reposting...' : 'Repost' }}
                  </button>
                  <div class="task-stats">
                    <span v-if="pendingCount(task) > 0" class="applications-count">
                      {{ pendingCount(task) }} awaiting your reply
                    </span>
                    <span v-else-if="task.applications?.length" class="text-sm text-gray">
                      {{ task.applications.length }} {{ task.applications.length === 1 ? 'application' : 'applications' }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Assignments Tab -->
          <div v-if="activeTab === 'assigned'" class="tab-pane">
            <div v-if="assignedTasks.length === 0" class="empty-state">
              <div class="empty-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none">
                  <path d="M16 21V19C16 17.9391 15.5786 16.9217 14.8284 16.1716C14.0783 15.4214 13.0609 15 12 15H5C3.93913 15 2.92172 15.4214 2.17157 16.1716C1.42143 16.9217 1 17.9391 1 19V21" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <circle cx="8.5" cy="7" r="4" stroke="currentColor" stroke-width="2"/>
                  <path d="M20 8V13" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <path d="M23 11L20 8L17 11" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <h3>No gigs assigned yet</h3>
              <p>Browse available gigs and apply to start working on exciting projects.</p>
              <router-link :to="{ name: 'tasks' }" class="btn btn-primary">Browse Available Gigs</router-link>
            </div>
            <div v-else class="task-list">
              <div 
                v-for="task in assignedTasks" 
                :key="task.id"
                class="task-item"
                @click="navigateToTask(task.id)"
              >
                <div class="task-header">
                  <h3 class="task-title">{{ task.title }}</h3>
                  <span class="badge" :class="'badge-' + task.status">
                    {{ task.status.replace('_', ' ') }}
                  </span>
                </div>
                <p class="task-description">{{ task.description }}</p>
                <div class="task-footer">
                  <div class="task-meta">
                    <span class="task-deadline">Posted {{ formatDate(task.created_at) }}</span>
                  </div>
                  <div class="task-creator">
                    <span>Created by: {{ task.creator?.username || 'Anonymous' }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Applications Tab -->
          <div v-if="activeTab === 'applications'" class="tab-pane">
            <div v-if="tasksStore.myApplications.length === 0" class="empty-state">
              <h3>No applications yet</h3>
              <p>Gigs you apply for show up here, with whether you got them.</p>
              <router-link :to="{ name: 'tasks' }" class="btn btn-primary">Browse Available Gigs</router-link>
            </div>
            <div v-else class="task-list">
              <div
                v-for="application in tasksStore.myApplications"
                :key="application.id"
                class="task-item"
                @click="navigateToTask(application.task_id)"
              >
                <div class="task-header">
                  <h3 class="task-title">{{ application.task.title }}</h3>
                  <span class="badge" :class="'application-' + application.status">
                    {{ applicationStatusLabel[application.status] }}
                  </span>
                </div>
                <div class="task-footer">
                  <div class="task-meta">
                    <span class="task-deadline">Applied {{ formatDate(application.created_at) }}</span>
                  </div>
                  <div class="task-creator">
                    <span>Posted by {{ application.task.creator?.username || 'unknown' }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { format } from 'date-fns'
import { useRouter, useRoute } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'
import { expiryLabel } from '@/utils/gigNote'

const tasksStore = useTasksStore()
const router = useRouter()
const route = useRoute()
const isLoading = ref(false)
const error = ref('')

// Which list: chosen from the side navigation (/my-gigs/created etc.)
type Tab = 'created' | 'assigned' | 'applications'
const TITLES: Record<Tab, string> = {
  created: 'Created Gigs',
  assigned: 'Assignments',
  applications: 'Applications'
}
const activeTab = computed<Tab>(() => {
  const tab = route.params.tab as Tab
  return tab in TITLES ? tab : 'created'
})
const pageTitle = computed(() => TITLES[activeTab.value])

const applicationStatusLabel = {
  pending: 'Waiting for reply',
  accepted: 'Accepted',
  declined: 'Not selected'
} as const

// Applications on the user's own gig still waiting for their decision
const pendingCount = (task: { applications?: { status?: string }[] }) =>
  (task.applications || []).filter(app => app.status === 'pending').length

const redirectToLogin = () => {
  const authStore = useAuthStore()
  authStore.logout() // Clear any existing auth state
  router.push({ name: 'login' })
}

const createdTasks = computed(() => {
  return tasksStore.userTasks || []
})

const assignedTasks = computed(() => {
  // Server-side filtering now handles excluding self-assigned tasks
  // We just return the filtered data from the API
  return tasksStore.assignedTasks || []
})

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

// Put an expired note back on the board for a fresh 24 hours
const repostingId = ref<number | null>(null)
const repost = async (taskId: number) => {
  repostingId.value = taskId
  try {
    await tasksStore.updateTaskStatus(taskId, 'open')
    await tasksStore.fetchUserTasks()
  } catch (err) {
    alert(err instanceof Error && err.message ? err.message : 'Could not repost the note. Please try again.')
  } finally {
    repostingId.value = null
  }
}

const navigateToTask = (taskId: number) => {
  router.push({ name: 'task-detail', params: { id: taskId } })
}

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
.dashboard-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.page-title {
  font-size: 2rem;
  font-weight: 700;
  color: #212529;
  margin: 0;
}

/* Stats Section */

/* Tabs Section */
.tabs-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.application-pending {
  background: #fef3c7;
  color: #92400e;
}

.application-accepted {
  background: #dcfce7;
  color: #166534;
}

.application-declined {
  background: #f3f4f6;
  color: #4b5563;
}

.tab-content {
  min-height: 400px;
}

.tab-pane {
  padding: 2rem;
}

/* Task List */
.task-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.task-item {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 1.5rem;
  cursor: pointer;
  transition: all 0.2s;
  border-left: 4px solid #dee2e6;
}

.task-item:hover {
  background: #e9ecef;
  border-left-color: var(--color-primary);
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.75rem;
}

.task-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #212529;
  margin: 0;
  flex: 1;
  margin-right: 1rem;
}

.task-description {
  color: #6c757d;
  margin-bottom: 1rem;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.task-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.task-meta {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.task-deadline {
  color: #6c757d;
  font-size: 0.875rem;
}

.task-stats, .task-creator {
  color: #6c757d;
  font-size: 0.875rem;
}

.applications-count {
  background: #eef2ff;
  color: var(--color-primary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

/* Badge Styles */
.badge {
  padding: 0.25rem 0.75rem;
  border-radius: 50px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.badge-open {
  background: #e8f5e9;
  color: #2e7d32;
}

.badge-in_progress {
  background: #fff3e0;
  color: #f57c00;
}

.badge-completed {
  background: #eef2ff;
  color: var(--color-primary);
}

.badge-cancelled {
  background: #ffebee;
  color: #d32f2f;
}

/* Loading and Error States */
.loading-spinner {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 0 auto;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  color: #6c757d;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: #f8f9fa;
  margin-bottom: 1.5rem;
  color: #adb5bd;
}

.empty-state h3 {
  font-size: 1.5rem;
  font-weight: 600;
  color: #495057;
  margin-bottom: 0.5rem;
}

.empty-state p {
  margin-bottom: 2rem;
  font-size: 1rem;
  line-height: 1.5;
}

.btn {
  display: inline-block;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s;
}

.btn-primary {
  background-color: var(--color-primary);
  color: white;
}

.btn-primary:hover {
  background-color: var(--color-primary-dark);
  color: white;
  text-decoration: none;
}

/* Responsive */
@media (max-width: 768px) {
  .dashboard-container {
    padding: 1rem;
  }
  
  .page-header {
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .page-title {
    font-size: 1.6rem;
  }

  /* Three tabs fit side by side; labels wrap instead of scrolling sideways */

  .tab-pane {
    padding: 1rem;
  }

  .task-item {
    padding: 1rem;
  }

  .task-header {
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  /* Three compact counters side by side instead of a screen per card */
  
  .tasks-grid {
    grid-template-columns: 1fr;
  }
}
</style>
