<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900">Available Tasks</h1>
      <router-link
        :to="{ name: 'create-task' }"
        class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
      >
        Create Task
      </router-link>
    </div>

    <div class="bg-white shadow overflow-hidden sm:rounded-md">
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
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { format } from 'date-fns'

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed'
  deadline: string
}

const statusClasses = {
  open: 'bg-green-100 text-green-800',
  in_progress: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-gray-100 text-gray-800'
}

const tasks = ref<Task[]>([])

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

// TODO: Implement fetch tasks logic
</script>
