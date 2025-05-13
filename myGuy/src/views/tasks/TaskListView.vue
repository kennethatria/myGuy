<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900">Available Gigs</h1>
      <router-link
        :to="{ name: 'create-task' }"
        class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
      >
        Create Gig
      </router-link>
    </div>
    
    <!-- Loading state -->
    <div v-if="isLoading" class="card p-4 mb-4 text-center">
      <div class="loading-spinner mb-2"></div>
      <p>Loading available gigs...</p>
    </div>
    
    <!-- Error state -->
    <div v-else-if="error" class="card p-4 mb-4 bg-red-100 text-danger">
      <p>{{ error }}</p>
      <button @click="fetchTasks" class="btn btn-outline mt-2">Retry</button>
    </div>
    
    <!-- Empty state -->
    <div v-else-if="tasks.length === 0" class="card p-8 text-center">
      <p class="text-lg mb-4">No available gigs found</p>
      <p class="text-gray mb-4">There are currently no open gigs from other users.</p>
      <router-link :to="{ name: 'create-task' }" class="btn btn-primary">
        Create Your First Gig
      </router-link>
    </div>

    <!-- Task list -->
    <div v-else class="bg-white shadow overflow-hidden sm:rounded-md">
      <ul role="list" class="divide-y divide-gray-200">
        <li v-for="task in tasks" :key="task.id">
          <router-link :to="{ name: 'task-details', params: { id: task.id }}" class="block hover:bg-gray-50">
            <div class="px-4 py-4 sm:px-6">
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium text-indigo-600 truncate">{{ task.title }}</p>
                <div class="ml-2 flex-shrink-0 flex">
                  <p
                    :class="[
                      statusClasses[task.status],
                      'px-2 inline-flex text-xs leading-5 font-semibold rounded-full'
                    ]"
                  >
                    {{ task.status }}
                  </p>
                </div>
              </div>
              <div class="mt-2 sm:flex sm:justify-between">
                <div class="sm:flex">
                  <p class="flex items-center text-sm text-gray-500 truncate max-w-md">
                    {{ task.description }}
                  </p>
                </div>
                <div class="mt-2 flex items-center text-sm text-gray-500 sm:mt-0 whitespace-nowrap">
                  <p>
                    Posted by: {{ task.creator ? task.creator.username : `User ${task.createdBy}` }}
                    <br/>
                    Deadline: {{ formatDate(task.deadline) }}
                  </p>
                </div>
              </div>
            </div>
          </router-link>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { format } from 'date-fns'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed'
  deadline: string
  createdBy: number
  creator?: {
    id: number
    username: string
  }
}

const statusClasses = {
  open: 'bg-green-100 text-green-800',
  in_progress: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-gray-100 text-gray-800'
}

const tasksStore = useTasksStore()
const authStore = useAuthStore()
const tasks = ref<Task[]>([])
const isLoading = ref(true)
const error = ref('')

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

const fetchTasks = async () => {
  isLoading.value = true
  error.value = ''
  
  try {
    // Fetch all available tasks
    await tasksStore.fetchTasks()
    
    // Filter out tasks created by the current user
    // Only show gigs from others that are still open
    const currentUserId = authStore.user?.id
    
    console.log('Current user ID:', currentUserId)
    console.log('All tasks:', tasksStore.tasks)
    
    tasks.value = tasksStore.tasks.filter(task => {
      // Convert IDs to strings for comparison to avoid type issues
      const taskCreatorId = String(task.createdBy)
      const userId = currentUserId ? String(currentUserId) : ''
      
      console.log(`Task ${task.id} - Created by: ${taskCreatorId}, User: ${userId}, Match: ${taskCreatorId === userId}`)
      
      // Only show tasks that are:
      // 1. Not created by the current user (show other users' tasks)
      // 2. Have an 'open' status (not in progress or completed)
      return userId && taskCreatorId !== userId && task.status === 'open'
    })
    
    console.log('Filtered tasks:', tasks.value)
    
  } catch (err: any) {
    console.error('Failed to fetch tasks:', err)
    error.value = err.message || 'Failed to load available gigs. Please try again.'
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  const isAuthenticated = await authStore.checkAuth()
  console.log("Is authenticated:", isAuthenticated)
  console.log("User after auth check:", authStore.user)
  
  await fetchTasks()
})
</script>
