<template>
  <div class="dashboard-container">

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
        <div class="stat-card completed">
          <div class="stat-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M20 6L9 17L4 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="stat-value">{{ stats.completedTasks }}</div>
          <div class="stat-label">Completed</div>
        </div>
        
        <div class="stat-card certificates">
          <div class="stat-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 2L2 7V12C2 16.5 4.5 20.5 12 22C19.5 20.5 22 16.5 22 12V7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="stat-value">{{ stats.assignedTasks }}</div>
          <div class="stat-label">Certificates</div>
        </div>
        
        <div class="stat-card achievements">
          <div class="stat-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 15L9.5 17.5L10.5 20.5L12 19L13.5 20.5L14.5 17.5L12 15Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <circle cx="12" cy="8" r="6" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <div class="stat-value">{{ stats.createdTasks }}</div>
          <div class="stat-label">Achievements</div>
        </div>
      </div>
    </div>

    <!-- Recommended Tasks -->
    <div v-if="!isLoading && !error">
      <section class="tasks-section">
        <h2 class="section-title">Recommended for you</h2>
        <div class="tasks-grid">
          <div 
            v-for="task in recommendedTasks" 
            :key="task.id"
            class="task-card"
            @click="navigateToTask(task.id)"
          >
            <div class="task-image">
              <img v-if="getTaskImage(task)" :src="getTaskImage(task)" alt="Task image" />
              <div v-else class="task-image-placeholder">
                <span>{{ task.title.charAt(0).toUpperCase() }}</span>
              </div>
            </div>
            <div class="task-info">
              <div class="task-meta">
                <span class="task-category">{{ getTaskCategory(task) }}</span>
                <span class="task-fee">${{ task.fee || 0 }}</span>
              </div>
              <h3 class="task-title">{{ task.title }}</h3>
              <p class="task-creator">{{ task.creator?.username || 'Anonymous' }}</p>
              <div class="task-rating">
                <span class="stars">⭐</span>
                <span class="rating-value">{{ task.rating || 4.3 }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Popular Tasks -->
      <section class="tasks-section">
        <h2 class="section-title">Popular tasks</h2>
        <div class="tasks-grid">
          <div 
            v-for="task in popularTasks" 
            :key="task.id"
            class="task-card"
            @click="navigateToTask(task.id)"
          >
            <div class="task-image">
              <img v-if="getTaskImage(task)" :src="getTaskImage(task)" alt="Task image" />
              <div v-else class="task-image-placeholder" :style="{ backgroundColor: getRandomColor() }">
                <span>{{ task.title.charAt(0).toUpperCase() }}</span>
              </div>
            </div>
            <div class="task-info">
              <div class="task-meta">
                <span class="task-category">{{ getTaskCategory(task) }}</span>
                <span class="task-fee">${{ task.fee || 0 }}</span>
              </div>
              <h3 class="task-title">{{ task.title }}</h3>
              <p class="task-creator">{{ task.creator?.username || 'Anonymous' }}</p>
              <div class="task-rating">
                <span class="stars">⭐</span>
                <span class="rating-value">{{ task.rating || 4.3 }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Original Gig lists (hidden) -->
    <div v-if="false" class="row">
      <!-- Posted Gigs -->
      <div class="col">
        <h2 class="mb-3">Gigs Posted by Me</h2>
        <div class="card mb-4">
          <div v-if="createdTasks.length === 0" class="p-4 text-center">
            <p>No gigs posted yet.</p>
          </div>
          <ul v-else class="divide-y">
            <li v-for="task in createdTasks" :key="task.id">
              <router-link :to="{ name: 'task-details', params: { id: task.id }}" class="block p-4 hover-card">
                <div class="flex justify-between items-center mb-2">
                  <h4 class="font-semibold text-primary">{{ task.title }}</h4>
                  <span class="badge" :class="'badge-' + task.status">
                    {{ task.status.replace('_', ' ') }}
                  </span>
                </div>
                <div class="flex justify-between items-start">
                  <p class="text-sm truncate mb-0" style="max-width: 65%;">
                    {{ task.description }}
                  </p>
                  <div class="text-sm text-right ml-2">
                    <strong>Due:</strong> {{ formatDate(task.deadline) }}
                  </div>
                </div>
              </router-link>
            </li>
          </ul>
        </div>
      </div>

      <!-- Assigned Gigs -->
      <div class="col">
        <h2 class="mb-3">Gigs Assigned to Me</h2>
        <div class="card">
          <div v-if="assignedTasks.length === 0" class="p-4 text-center">
            <p>No gigs assigned yet.</p>
          </div>
          <div v-else-if="debug" class="p-4 bg-blue-50">
            <h4 class="font-semibold">Debug Info (admin only)</h4>
            <pre class="text-xs overflow-auto mt-2 p-2 bg-gray-100 rounded">{{ JSON.stringify(assignedTasks, null, 2) }}</pre>
          </div>
          <ul v-else class="divide-y">
            <li v-for="task in assignedTasks" :key="task.id">
              <router-link :to="{ name: 'task-details', params: { id: task.id }}" class="block p-4 hover-card">
                <div class="flex justify-between items-center mb-2">
                  <h4 class="font-semibold text-primary">{{ task.title }}</h4>
                  <span class="badge" :class="'badge-' + task.status">
                    {{ task.status.replace('_', ' ') }}
                  </span>
                </div>
                <div class="flex justify-between items-start">
                  <p class="text-sm truncate mb-0" style="max-width: 65%;">
                    {{ task.description }}
                  </p>
                  <div class="text-sm text-right ml-2">
                    <strong>Due:</strong> {{ formatDate(task.deadline) }}
                  </div>
                </div>
              </router-link>
            </li>
          </ul>
        </div>
      </div>
    </div>
    
    <!-- Empty state for mobile view testing -->
    <div v-if="false" class="mt-4">
      <p class="text-center text-gray">This is an empty placeholder view for testing responsiveness.</p>
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
const debug = ref(false) // Set to false to hide debug info

const recommendedTasks = ref<Task[]>([])
const popularTasks = ref<Task[]>([])

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

const getTaskImage = (task: Task) => {
  // Return null for now, could be extended to return actual task images
  return null
}

const getTaskCategory = (task: Task) => {
  // Simple category determination based on keywords in title/description
  const text = (task.title + ' ' + task.description).toLowerCase()
  if (text.includes('design') || text.includes('logo')) return 'Design'
  if (text.includes('web') || text.includes('app') || text.includes('code')) return 'Development'
  if (text.includes('write') || text.includes('content')) return 'Writing'
  if (text.includes('data') || text.includes('analysis')) return 'Data Analysis'
  return 'General'
}

const getRandomColor = () => {
  const colors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', '#FECA57', '#B983FF', '#FD79A8', '#A0E7E5']
  return colors[Math.floor(Math.random() * colors.length)]
}

// Load sample data for development
const loadSampleData = () => {
  // Generated sample data
  const sampleTasks = [
    {
      id: 1,
      title: "Website Redesign",
      description: "Looking for a skilled web designer to refresh our company site with modern UI elements and improved user flow.",
      status: "open",
      createdBy: 1,
      deadline: "2023-12-31T00:00:00Z",
      created_at: "2023-09-15T10:30:00Z",
    },
    {
      id: 2,
      title: "Mobile App Bug Fixes",
      description: "Need developer to fix several critical bugs in our iOS application. Familiarity with Swift required.",
      status: "in_progress",
      createdBy: 1,
      assignedTo: 2,
      deadline: "2023-11-15T00:00:00Z",
      created_at: "2023-09-10T14:20:00Z",
    },
    {
      id: 3,
      title: "Content Writing for Blog",
      description: "Create five SEO-optimized blog posts about digital marketing trends.",
      status: "completed",
      createdBy: 1,
      assignedTo: 3,
      deadline: "2023-10-01T00:00:00Z",
      created_at: "2023-08-25T09:15:00Z",
    }
  ]
  
  // Replace store data with sample data
  tasksStore.userTasks = sampleTasks
  
  // Sample assigned tasks
  tasksStore.assignedTasks = [
    {
      id: 4,
      title: "Logo Design for Startup",
      description: "Create a modern logo for a tech startup focusing on AI solutions.",
      status: "open",
      createdBy: 2,
      assignedTo: 1,
      deadline: "2023-11-30T00:00:00Z",
      created_at: "2023-09-18T11:45:00Z",
    },
    {
      id: 5,
      title: "Data Analysis Project",
      description: "Analyze customer data and create visualization dashboard using Python and Tableau.",
      status: "in_progress",
      createdBy: 3,
      assignedTo: 1,
      deadline: "2023-12-15T00:00:00Z",
      created_at: "2023-09-05T16:30:00Z",
    }
  ]
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
    
    // Fetch recommended and popular tasks
    try {
      const allTasks = await tasksStore.fetchTasks({
        status: 'open',
        sort_by: 'created_at',
        sort_order: 'desc',
        per_page: 8
      })
      
      // Split tasks for recommended and popular sections
      if (allTasks && allTasks.data && Array.isArray(allTasks.data)) {
        recommendedTasks.value = allTasks.data.slice(0, 4)
        popularTasks.value = allTasks.data.slice(4, 8)
      } else if (allTasks && Array.isArray(allTasks)) {
        // Handle case where allTasks is directly an array
        recommendedTasks.value = allTasks.slice(0, 4)
        popularTasks.value = allTasks.slice(4, 8)
      }
    } catch (err) {
      console.error('Failed to fetch recommended tasks:', err)
      // Set empty arrays on error
      recommendedTasks.value = []
      popularTasks.value = []
    }
    
    // If the backend is down or no data is available, uncomment this for testing:
    // loadSampleData()
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

/* Stats Section */
.stats-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin-bottom: 3rem;
}

.stat-card {
  background: #E3F2FD;
  border-radius: 12px;
  padding: 2rem;
  text-align: center;
  transition: transform 0.2s;
  cursor: pointer;
}

.stat-card:hover {
  transform: translateY(-4px);
}

.stat-card.completed {
  background: #E8F5E9;
}

.stat-card.certificates {
  background: #E3F2FD;
}

.stat-card.achievements {
  background: #FFF3E0;
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

/* Tasks Section */
.tasks-section {
  margin-bottom: 3rem;
}

.section-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: #212529;
  margin-bottom: 1.5rem;
}

.tasks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1.5rem;
}

.task-card {
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: all 0.3s;
}

.task-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.task-image {
  height: 160px;
  background: #f8f9fa;
  position: relative;
  overflow: hidden;
}

.task-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.task-image-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-size: 3rem;
  font-weight: 700;
  color: white;
}

.task-info {
  padding: 1.25rem;
}

.task-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
  font-size: 0.875rem;
}

.task-category {
  color: #6c757d;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.task-fee {
  font-weight: 600;
  color: #212529;
}

.task-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #212529;
  margin-bottom: 0.5rem;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.task-creator {
  color: #6c757d;
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.task-rating {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.stars {
  color: #ffc107;
  font-size: 1rem;
}

.rating-value {
  color: #212529;
  font-weight: 600;
  font-size: 0.875rem;
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
