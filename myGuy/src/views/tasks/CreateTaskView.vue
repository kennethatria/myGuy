<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="md:grid md:grid-cols-3 md:gap-6">
      <div class="md:col-span-1">
        <div class="px-4 sm:px-0">
          <h3 class="text-lg font-medium leading-6 text-gray-900">Post a New Gig</h3>
          <p class="mt-1 text-sm text-gray-600">
            Provide the details for your gig. Be specific about your requirements and deadline.
          </p>
        </div>
      </div>

      <div class="mt-5 md:mt-0 md:col-span-2">
        <form @submit.prevent="handleSubmit">
          <div class="shadow sm:rounded-md sm:overflow-hidden">
            <div class="px-4 py-5 bg-white space-y-6 sm:p-6">
              <div>
                <label for="title" class="block text-sm font-medium text-gray-700">Title</label>
                <div class="mt-1">
                  <input
                    type="text"
                    name="title"
                    id="title"
                    v-model="task.title"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
                    required
                  />
                </div>
              </div>

              <div>
                <label for="description" class="block text-sm font-medium text-gray-700">Description</label>
                <div class="mt-1">
                  <textarea
                    id="description"
                    name="description"
                    rows="3"
                    v-model="task.description"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md"
                    required
                  />
                </div>
              </div>

              <div>
                <label for="deadline" class="block text-sm font-medium text-gray-700">Deadline</label>
                <div class="mt-1">
                  <input
                    type="datetime-local"
                    name="deadline"
                    id="deadline"
                    v-model="task.deadline"
                    :min="minDeadlineString"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
                    required
                  />
                  <p class="mt-1 text-sm text-gray-500">Deadline must be at least one day in the future</p>
                </div>
              </div>
            </div>

            <div class="px-4 py-3 bg-gray-50 text-right sm:px-6">
              <button
                type="button"
                @click="$router.back()"
                class="inline-flex justify-center py-2 px-4 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 mr-3"
              >
                Cancel
              </button>
              <button
                type="submit"
                class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              >
                Post Gig
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'

const router = useRouter()

// Helper to format datetime-local string
const formatDatetimeLocal = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

// Calculate the minimum deadline (now + 1 day)
const minDeadlineString = computed(() => {
  const now = new Date()
  const minDeadline = new Date(now)
  minDeadline.setDate(minDeadline.getDate() + 1)
  return formatDatetimeLocal(minDeadline)
})

const task = ref({
  title: '',
  description: '',
  deadline: minDeadlineString.value // Initialize with the minimum valid date
})

const validateDeadline = (deadlineStr: string): boolean => {
  const deadlineDate = new Date(deadlineStr)
  const now = new Date()
  
  // Add 1 day to the current time (matching backend validation)
  const minDeadline = new Date(now)
  minDeadline.setDate(minDeadline.getDate() + 1)
  
  return deadlineDate >= minDeadline
}

const handleSubmit = async () => {
  try {
    // Validate the deadline is at least one day in the future
    if (!validateDeadline(task.value.deadline)) {
      alert('Deadline must be at least one day in the future')
      return
    }
    
    // Format the deadline to RFC3339 format with timezone
    const deadlineDate = new Date(task.value.deadline)
    const formattedDeadline = deadlineDate.toISOString()
    
    // Create a new object with formatted deadline
    const taskData = {
      title: task.value.title,
      description: task.value.description,
      deadline: formattedDeadline,
      fee: 0 // Default fee required by backend
    }
    
    // Import and use the tasks store
    const tasksStore = useTasksStore()
    await tasksStore.createTask(taskData)
    
    await router.push({ name: 'tasks' })
  } catch (error) {
    console.error('Failed to create task:', error)
    alert('Error: ' + (error.message || 'Failed to create task. Make sure deadline is at least one day in the future.'))
  }
}
</script>
