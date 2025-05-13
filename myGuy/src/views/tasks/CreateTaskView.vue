<template>
  <div class="container py-4">
    <h1 class="mb-4">Post a New Gig</h1>
    
    <div class="row">
      <!-- Form description -->
      <div class="col">
        <div class="card mb-4">
          <h3>Guidelines</h3>
          <p class="mt-2 text-gray">
            Provide the details for your gig. Be specific about your requirements and deadline.
            Clear descriptions help potential applicants understand what you need.
          </p>
          <ul class="mt-3">
            <li class="mb-1">Set a clear title that describes the gig</li>
            <li class="mb-1">Explain all requirements in detail</li>
            <li class="mb-1">Set a realistic deadline (at least one day from now)</li>
          </ul>
        </div>
      </div>

      <!-- Gig creation form -->
      <div class="col">
        <div class="card">
          <form @submit.prevent="handleSubmit">
            <div class="form-group">
              <label for="title" class="form-label">Gig Title</label>
              <input
                type="text"
                name="title"
                id="title"
                v-model="task.title"
                class="form-input"
                :class="{ 'is-invalid': formErrors.title }"
                placeholder="E.g., Website Development, Logo Design, Data Entry"
                required
              />
              <div v-if="formErrors.title" class="invalid-feedback">{{ formErrors.title }}</div>
            </div>

            <div class="form-group">
              <label for="description" class="form-label">Description</label>
              <textarea
                id="description"
                name="description"
                rows="5"
                v-model="task.description"
                class="form-input"
                :class="{ 'is-invalid': formErrors.description }"
                placeholder="Describe your requirements in detail..."
                required
              ></textarea>
              <div v-if="formErrors.description" class="invalid-feedback">{{ formErrors.description }}</div>
            </div>

            <div class="form-group">
              <label for="deadline" class="form-label">Deadline</label>
              <input
                type="datetime-local"
                name="deadline"
                id="deadline"
                v-model="task.deadline"
                :min="minDeadlineString"
                class="form-input"
                :class="{ 'is-invalid': formErrors.deadline }"
                required
              />
              <p class="form-helper">Deadline must be at least one day in the future</p>
              <div v-if="formErrors.deadline" class="invalid-feedback">{{ formErrors.deadline }}</div>
            </div>

            <div class="form-group" v-if="formError">
              <div class="invalid-feedback">{{ formError }}</div>
            </div>

            <div class="flex justify-end mt-4">
              <button
                type="button"
                @click="$router.back()"
                class="btn btn-outline mr-2"
              >
                Cancel
              </button>
              <button
                type="submit"
                class="btn btn-primary"
                :disabled="isSubmitting"
              >
                <span v-if="isSubmitting">Posting...</span>
                <span v-else>Post Gig</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'

const router = useRouter()
const isSubmitting = ref(false)
const formError = ref('')
const formErrors = ref({
  title: '',
  description: '',
  deadline: ''
})

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

const validateForm = (): boolean => {
  let isValid = true
  formErrors.value = {
    title: '',
    description: '',
    deadline: ''
  }
  formError.value = ''
  
  // Validate title
  if (!task.value.title.trim()) {
    formErrors.value.title = 'Title is required'
    isValid = false
  } else if (task.value.title.length < 5) {
    formErrors.value.title = 'Title must be at least 5 characters'
    isValid = false
  }
  
  // Validate description
  if (!task.value.description.trim()) {
    formErrors.value.description = 'Description is required'
    isValid = false
  } else if (task.value.description.length < 20) {
    formErrors.value.description = 'Description must be at least 20 characters'
    isValid = false
  }
  
  // Validate deadline
  if (!task.value.deadline) {
    formErrors.value.deadline = 'Deadline is required'
    isValid = false
  } else if (!validateDeadline(task.value.deadline)) {
    formErrors.value.deadline = 'Deadline must be at least one day in the future'
    isValid = false
  }
  
  return isValid
}

const handleSubmit = async () => {
  if (!validateForm()) {
    return
  }
  
  try {
    isSubmitting.value = true
    formError.value = ''
    
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
    formError.value = error.message || 'Failed to create gig. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>
