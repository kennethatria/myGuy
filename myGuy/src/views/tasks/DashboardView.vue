<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <h1 class="text-2xl font-semibold text-gray-900 mb-6">My Dashboard</h1>

    <!-- Tasks overview -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div class="bg-white overflow-hidden shadow rounded-lg">
        <div class="p-5">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <svg class="h-6 w-6 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <div class="ml-5 w-0 flex-1">
              <dl>
                <dt class="text-sm font-medium text-gray-500 truncate">Gigs Posted</dt>
                <dd class="flex items-baseline">
                  <div class="text-2xl font-semibold text-gray-900">{{ stats.createdTasks }}</div>
                </dd>
              </dl>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-white overflow-hidden shadow rounded-lg">
        <div class="p-5">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <svg class="h-6 w-6 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
              </svg>
            </div>
            <div class="ml-5 w-0 flex-1">
              <dl>
                <dt class="text-sm font-medium text-gray-500 truncate">Gigs Assigned</dt>
                <dd class="flex items-baseline">
                  <div class="text-2xl font-semibold text-gray-900">{{ stats.assignedTasks }}</div>
                </dd>
              </dl>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-white overflow-hidden shadow rounded-lg">
        <div class="p-5">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <svg class="h-6 w-6 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
              </svg>
            </div>
            <div class="ml-5 w-0 flex-1">
              <dl>
                <dt class="text-sm font-medium text-gray-500 truncate">Completed Gigs</dt>
                <dd class="flex items-baseline">
                  <div class="text-2xl font-semibold text-gray-900">{{ stats.completedTasks }}</div>
                </dd>
              </dl>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tasks lists -->
    <div class="mt-8">
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Created Tasks -->
        <div>
          <h2 class="text-lg font-medium text-gray-900 mb-4">Gigs Posted by Me</h2>
          <div class="bg-white shadow overflow-hidden sm:rounded-md">
            <ul role="list" class="divide-y divide-gray-200">
              <li v-for="task in createdTasks" :key="task.id">
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
                        <p class="flex items-center text-sm text-gray-500">
                          {{ task.description }}
                        </p>
                      </div>
                      <div class="mt-2 flex items-center text-sm text-gray-500 sm:mt-0">
                        <p>
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

        <!-- Assigned Tasks -->
        <div>
          <h2 class="text-lg font-medium text-gray-900 mb-4">Gigs Assigned to Me</h2>
          <div class="bg-white shadow overflow-hidden sm:rounded-md">
            <ul role="list" class="divide-y divide-gray-200">
              <li v-for="task in assignedTasks" :key="task.id">
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
                        <p class="flex items-center text-sm text-gray-500">
                          {{ task.description }}
                        </p>
                      </div>
                      <div class="mt-2 flex items-center text-sm text-gray-500 sm:mt-0">
                        <p>
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
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { format } from 'date-fns'

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed'
  deadline: string
}

interface Stats {
  createdTasks: number
  assignedTasks: number
  completedTasks: number
}

const statusClasses = {
  open: 'bg-green-100 text-green-800',
  in_progress: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-gray-100 text-gray-800'
}

const stats = ref<Stats>({
  createdTasks: 0,
  assignedTasks: 0,
  completedTasks: 0
})

const createdTasks = ref<Task[]>([])
const assignedTasks = ref<Task[]>([])

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

onMounted(async () => {
  try {
    // TODO: Implement fetch dashboard data logic
  } catch (error) {
    console.error('Failed to fetch dashboard data:', error)
  }
})
</script>
