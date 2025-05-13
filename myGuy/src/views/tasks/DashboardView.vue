<template>
  <div class="container py-4">
    <h1 class="mb-4">My Dashboard</h1>

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
      <!-- Stats overview -->
      <div class="row mb-4">
        <div class="col">
          <div class="card">
            <div class="flex items-center">
              <div class="mr-3">
                <svg class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="var(--color-primary)">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
              </div>
              <div>
                <h3 class="text-sm mb-1">Gigs Posted</h3>
                <div class="text-xl font-bold">{{ stats.createdTasks }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="col">
          <div class="card">
            <div class="flex items-center">
              <div class="mr-3">
                <svg class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="var(--color-primary)">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
                </svg>
              </div>
              <div>
                <h3 class="text-sm mb-1">Gigs Assigned</h3>
                <div class="text-xl font-bold">{{ stats.assignedTasks }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="col">
          <div class="card">
            <div class="flex items-center">
              <div class="mr-3">
                <svg class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="var(--color-primary)">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
                </svg>
              </div>
              <div>
                <h3 class="text-sm mb-1">Completed Gigs</h3>
                <div class="text-xl font-bold">{{ stats.completedTasks }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Gig lists -->
    <div v-if="!isLoading && !error" class="row">
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
  deadline: string
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
