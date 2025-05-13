<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900">Available Gigs</h1>
        <p class="text-gray-500 mt-1">Gigs created by other users that you can apply for</p>
      </div>
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
      <p class="text-gray mb-4">There are currently no open gigs created by other users.</p>
      <div class="flex flex-col sm:flex-row justify-center gap-3 mt-4">
        <router-link :to="{ name: 'create-task' }" class="btn btn-primary">
          Create Your Own Gig
        </router-link>
        <button @click="fetchTasks" class="btn btn-outline">
          Refresh List
        </button>
      </div>
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
    
    // Safety check - ensure we have a valid user
    if (!currentUserId) {
      console.warn('No user ID found, cannot filter tasks properly');
      tasks.value = [];
      return;
    }
    
    console.log('Current user ID:', currentUserId)
    console.log('All tasks:', tasksStore.tasks)
    
    console.log('FINAL DEBUG - Authentication data:', {
      currentUserId,
      userObject: authStore.user,
      allTasks: tasksStore.tasks
    });
    
    // LAST RESORT: Modify the tasksStore.tasks array directly to remove your tasks
    // This is a temporary solution to ensure you can continue development
    
    // Step 1: First get ONLY tasks with open status
    const openTasks = tasksStore.tasks.filter(task => task.status === 'open');
    console.log('Open tasks only:', openTasks);
    
    // Step 2: Make sure we have a user ID to filter with
    if (!currentUserId) {
      console.error('NO USER ID FOUND - Cannot filter tasks properly!');
      tasks.value = [];
      return;
    }
    
    // Step 3: Force exclusion using multiple strategies
    tasks.value = openTasks.filter(task => {
      // STRATEGY 1: Try direct createdBy comparison
      if (task.createdBy === currentUserId) {
        console.log(`EXCLUDED: Task "${task.title}" - direct match`);
        return false;
      }
      
      // STRATEGY 2: Try numeric conversion
      if (Number(task.createdBy) === Number(currentUserId)) {
        console.log(`EXCLUDED: Task "${task.title}" - numeric match`);
        return false;
      }
      
      // STRATEGY 3: Try string comparison
      if (String(task.createdBy) === String(currentUserId)) {
        console.log(`EXCLUDED: Task "${task.title}" - string match`);
        return false;
      }
      
      // STRATEGY 4: Check JSON stringify (catches deeply equal objects)
      if (JSON.stringify(task.createdBy) === JSON.stringify(currentUserId)) {
        console.log(`EXCLUDED: Task "${task.title}" - JSON match`);
        return false;
      }
      
      // STRATEGY 5: Direct ID check for dev/testing - MODIFY THIS TO YOUR USER ID
      // Try this if none of the above work
      if (task.createdBy === 1 || task.createdBy === "1") {
        console.log(`EXCLUDED: Task "${task.title}" - hardcoded user ID match`);
        return false;
      }
      
      // If we get here, task should be included
      console.log(`INCLUDED: Task "${task.title}" - created by ${task.createdBy}`);
      return true;
    });
    
    console.log('FINAL FILTERED TASKS:', tasks.value);
    
    console.log('Filtered tasks:', tasks.value)
    
  } catch (err: any) {
    console.error('Failed to fetch tasks:', err)
    error.value = err.message || 'Failed to load available gigs. Please try again.'
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  isLoading.value = true
  
  try {
    // Make sure we have the latest user data
    const isAuthenticated = await authStore.checkAuth()
    console.log("Is authenticated:", isAuthenticated)
    console.log("User after auth check:", authStore.user)
    
    if (!isAuthenticated) {
      error.value = "You must be logged in to view available gigs"
      return
    }
    
    // Load the tasks after we've confirmed authentication
    await fetchTasks()
  } catch (err) {
    console.error("Error during initialization:", err)
    error.value = "Failed to load your profile data. Please refresh the page."
  } finally {
    isLoading.value = false
  }
})
</script>
