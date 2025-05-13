<template>
  <div class="container py-4">
    <h1 class="mb-4">My Dashboard</h1>

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

    <!-- Gig lists -->
    <div class="row">
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
