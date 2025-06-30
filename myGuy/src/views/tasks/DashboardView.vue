<template>
  <div class="dashboard-container">
    <!-- Page Header -->
    <div class="page-header">
      <h1 class="page-title">My Dashboard</h1>
      <router-link :to="{ name: 'create-task' }" class="btn btn-primary">
        + Post New Gig
      </router-link>
    </div>

    <!-- Loading and error states -->
    <div v-if="isLoading" class="card p-4 mb-4 text-center">
      <div class="loading-spinner mb-2"></div>
      <p>Loading dashboard data...</p>
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
        <button @click="fetchDashboardData" class="btn btn-outline">Retry</button>
      </div>
    </div>

    <div v-else>
      <!-- Stats Cards -->
      <div class="stats-section">
        <div class="stat-card created">
          <div class="stat-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 2L2 7V12C2 16.5 4.5 20.5 12 22C19.5 20.5 22 16.5 22 12V7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="stat-value">{{ stats.createdTasks }}</div>
          <div class="stat-label">Created Gigs</div>
        </div>
        
        <div class="stat-card assigned">
          <div class="stat-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M16 21V19C16 17.9391 15.5786 16.9217 14.8284 16.1716C14.0783 15.4214 13.0609 15 12 15H5C3.93913 15 2.92172 15.4214 2.17157 16.1716C1.42143 16.9217 1 17.9391 1 19V21" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <circle cx="8.5" cy="7" r="4" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <div class="stat-value">{{ stats.assignedTasks }}</div>
          <div class="stat-label">Assigned to Me</div>
        </div>
        
        <div class="stat-card completed">
          <div class="stat-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M20 6L9 17L4 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="stat-value">{{ stats.completedTasks }}</div>
          <div class="stat-label">Completed</div>
        </div>
      </div>

      <!-- Tab Navigation -->
      <div class="tabs-section">
        <div class="tab-nav">
          <button 
            class="tab-button" 
            :class="{ active: activeTab === 'created' }"
            @click="activeTab = 'created'"
          >
            My Created Gigs
          </button>
          <button 
            class="tab-button" 
            :class="{ active: activeTab === 'assigned' }"
            @click="activeTab = 'assigned'"
          >
            Gigs Assigned to Me
          </button>
        </div>

        <!-- Tab Content -->
        <div class="tab-content">
          <!-- My Created Gigs Tab -->
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
                    <span class="task-fee">${{ task.fee || 0 }}</span>
                    <span class="task-deadline">Due: {{ formatDate(task.deadline) }}</span>
                  </div>
                  <div class="task-stats">
                    <span v-if="task.applications?.length" class="applications-count">
                      {{ task.applications.length }} applications
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Gigs Assigned to Me Tab -->
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
                    <span class="task-fee">${{ task.fee || 0 }}</span>
                    <span class="task-deadline">Due: {{ formatDate(task.deadline) }}</span>
                  </div>
                  <div class="task-creator">
                    <span>Created by: {{ task.creator?.username || 'Anonymous' }}</span>
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
import { useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'

interface Task {
  id: number,
  title: string,
  description: string,
  status: 'open' | 'in_progress' | 'completed',
  deadline: string,
  fee?: number,
  creator?: {
    username: string
  },
  rating?: number,
  application_count?: number
}

interface Stats {
  createdTasks: number,
  assignedTasks: number,
  completedTasks: number
}

const tasksStore = useTasksStore()
const router = useRouter()
const isLoading = ref(false)
const error = ref('')
const activeTab = ref<'created' | 'assigned'>('created')

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

const stats = computed<Stats>(() => {
  const created = createdTasks.value.length
  const assigned = assignedTasks.value.length
  
  // Only count completed tasks once, don't double-count tasks that appear in both lists
  const completedCreated = createdTasks.value.filter(task => task.status === 'completed').length
  const completedAssigned = assignedTasks.value.filter(task => task.status === 'completed').length
  const completed = completedCreated + completedAssigned
  
  return {
    createdTasks: created,
    assignedTasks: assigned,
    completedTasks: completed,
  }
})

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

const navigateToTask = (taskId: number) => {
  router.push({ name: 'task-detail', params: { id: taskId } })
}

const fetchDashboardData = async () => {
  isLoading.value = true
  error.value = ''
  
  try {
    // Make sure we have a valid token before trying to fetch data
    const authStore = useAuthStore()
    if (!authStore.token) {
      error.value = 'Please log in to view your dashboard.'
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
      tasksStore.fetchAssignedTasks()
    ])
  } catch (err: any) {
    console.error('Failed to fetch dashboard data:', err)
    
    // Check if it's an authentication error
    if (err.message && err.message.includes('log in again')) {
      error.value = err.message
    } else {
      error.value = 'Failed to load dashboard data. Please try again later.'
    }
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  await fetchDashboardData()
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
.stats-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.5rem;
  margin-bottom: 3rem;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  text-align: center;
  transition: transform 0.2s;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border: 2px solid transparent;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.stat-card.created {
  border-color: #4CAF50;
}

.stat-card.assigned {
  border-color: #2196F3;
}

.stat-card.completed {
  border-color: #FF9800;
}

.stat-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.1);
  margin-bottom: 1rem;
}

.stat-icon svg {
  color: #333;
}

.stat-value {
  font-size: 2.5rem;
  font-weight: 700;
  color: #212529;
  margin-bottom: 0.5rem;
}

.stat-label {
  font-size: 1rem;
  color: #6c757d;
  font-weight: 500;
}

/* Tabs Section */
.tabs-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.tab-nav {
  display: flex;
  border-bottom: 1px solid #e9ecef;
}

.tab-button {
  flex: 1;
  padding: 1rem 2rem;
  background: none;
  border: none;
  font-size: 1rem;
  font-weight: 500;
  color: #6c757d;
  cursor: pointer;
  transition: all 0.2s;
  border-bottom: 3px solid transparent;
}

.tab-button:hover {
  background: #f8f9fa;
  color: #495057;
}

.tab-button.active {
  color: #1976d2;
  border-bottom-color: #1976d2;
  background: #f8f9fa;
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
  border-left-color: #1976d2;
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

.task-fee {
  font-weight: 600;
  color: #28a745;
  font-size: 1.1rem;
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
  background: #e3f2fd;
  color: #1976d2;
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
  background: #e3f2fd;
  color: #1976d2;
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
  background-color: #1976d2;
  color: white;
}

.btn-primary:hover {
  background-color: #1565c0;
  color: white;
  text-decoration: none;
}

/* Responsive */
@media (max-width: 768px) {
  .dashboard-container {
    padding: 1rem;
  }
  
  .stats-section {
    grid-template-columns: 1fr;
    gap: 1rem;
  }
  
  .tasks-grid {
    grid-template-columns: 1fr;
  }
}
</style>
