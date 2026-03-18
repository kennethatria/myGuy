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
              <label for="fee" class="form-label">Budget/Fee (UGX)</label>
              <input
                type="number"
                name="fee"
                id="fee"
                v-model.number="task.fee"
                class="form-input"
                :class="{ 'is-invalid': formErrors.fee }"
                placeholder="Enter your budget (e.g., 50000)"
                min="0.01"
                step="0.01"
                required
              />
              <p class="form-helper">Enter the amount you're willing to pay for this gig</p>
              <div v-if="formErrors.fee" class="invalid-feedback">{{ formErrors.fee }}</div>
            </div>

            <div class="form-group">
              <label class="form-label">Message Privacy</label>
              <div class="privacy-toggle-container">
                <div class="privacy-option">
                  <input
                    type="radio"
                    id="private-messages"
                    name="message-privacy"
                    :value="false"
                    v-model="task.isMessagesPublic"
                    class="privacy-radio"
                  />
                  <label for="private-messages" class="privacy-label">
                    <span class="privacy-title">🔒 Private Messages (Recommended)</span>
                    <span class="privacy-description">Only you and the assigned person can see messages</span>
                  </label>
                </div>
                <div class="privacy-option">
                  <input
                    type="radio"
                    id="public-messages"
                    name="message-privacy"
                    :value="true"
                    v-model="task.isMessagesPublic"
                    class="privacy-radio"
                  />
                  <label for="public-messages" class="privacy-label">
                    <span class="privacy-title">🌐 Public Messages</span>
                    <span class="privacy-description">Anyone viewing the gig can see all messages</span>
                  </label>
                </div>
              </div>
              <p class="form-helper">Choose whether messages on this gig should be private or public</p>
            </div>

            <div class="form-group">
              <label class="form-label">Deadline</label>
              
              <!-- Quick preset options -->
              <div class="deadline-presets">
                <button
                  type="button"
                  v-for="preset in deadlinePresets"
                  :key="preset.label"
                  @click="setDeadlinePreset(preset.days)"
                  class="preset-btn"
                  :class="{ 'active': isPresetActive(preset.days) }"
                >
                  {{ preset.label }}
                </button>
              </div>

              <!-- Custom date and time inputs -->
              <div class="datetime-inputs">
                <div class="date-input-group">
                  <label for="deadline-date" class="input-label">Date</label>
                  <input
                    type="date"
                    id="deadline-date"
                    v-model="deadlineDate"
                    :min="minDeadlineDate"
                    class="form-input"
                    :class="{ 'is-invalid': formErrors.deadline }"
                    required
                  />
                </div>
                <div class="time-input-group">
                  <label for="deadline-time" class="input-label">Time</label>
                  <select
                    id="deadline-time"
                    v-model="deadlineTime"
                    class="form-input"
                    :class="{ 'is-invalid': formErrors.deadline }"
                    required
                  >
                    <option value="">Select time</option>
                    <option v-for="time in timeOptions" :key="time.value" :value="time.value">
                      {{ time.label }}
                    </option>
                  </select>
                </div>
              </div>

              <p class="form-helper">Choose a deadline at least 24 hours from now</p>
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
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'

const router = useRouter()
const isSubmitting = ref(false)
const formError = ref('')
const formErrors = ref({
  title: '',
  description: '',
  fee: '',
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
  fee: null as number | null,
  deadline: minDeadlineString.value, // Initialize with the minimum valid date
  isMessagesPublic: false // Default to private messages
})

// Separate deadline components for better UX
const deadlineDate = ref('')
const deadlineTime = ref('')

// Preset deadline options
const deadlinePresets = [
  { label: '1 Day', days: 1 },
  { label: '3 Days', days: 3 },
  { label: '1 Week', days: 7 },
  { label: '2 Weeks', days: 14 },
  { label: '1 Month', days: 30 }
]

// Time options in 30-minute intervals
const timeOptions = [
  { value: '09:00', label: '9:00 AM' },
  { value: '09:30', label: '9:30 AM' },
  { value: '10:00', label: '10:00 AM' },
  { value: '10:30', label: '10:30 AM' },
  { value: '11:00', label: '11:00 AM' },
  { value: '11:30', label: '11:30 AM' },
  { value: '12:00', label: '12:00 PM' },
  { value: '12:30', label: '12:30 PM' },
  { value: '13:00', label: '1:00 PM' },
  { value: '13:30', label: '1:30 PM' },
  { value: '14:00', label: '2:00 PM' },
  { value: '14:30', label: '2:30 PM' },
  { value: '15:00', label: '3:00 PM' },
  { value: '15:30', label: '3:30 PM' },
  { value: '16:00', label: '4:00 PM' },
  { value: '16:30', label: '4:30 PM' },
  { value: '17:00', label: '5:00 PM' },
  { value: '17:30', label: '5:30 PM' },
  { value: '18:00', label: '6:00 PM' },
  { value: '18:30', label: '6:30 PM' },
  { value: '19:00', label: '7:00 PM' },
  { value: '19:30', label: '7:30 PM' },
  { value: '20:00', label: '8:00 PM' },
  { value: '20:30', label: '8:30 PM' },
  { value: '21:00', label: '9:00 PM' }
]

// Minimum deadline date (tomorrow)
const minDeadlineDate = computed(() => {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  return tomorrow.toISOString().split('T')[0]
})

// Methods for deadline management
const setDeadlinePreset = (days: number) => {
  const presetDate = new Date()
  presetDate.setDate(presetDate.getDate() + days)
  
  deadlineDate.value = presetDate.toISOString().split('T')[0]
  deadlineTime.value = '17:00' // Default to 5:00 PM
  
  updateTaskDeadline()
}

const isPresetActive = (days: number): boolean => {
  if (!deadlineDate.value || !deadlineTime.value) return false
  
  const selectedDate = new Date(`${deadlineDate.value}T${deadlineTime.value}`)
  const presetDate = new Date()
  presetDate.setDate(presetDate.getDate() + days)
  presetDate.setHours(17, 0, 0, 0) // 5:00 PM
  
  return Math.abs(selectedDate.getTime() - presetDate.getTime()) < 60000 // Within 1 minute
}

const updateTaskDeadline = () => {
  if (deadlineDate.value && deadlineTime.value) {
    const combinedDateTime = new Date(`${deadlineDate.value}T${deadlineTime.value}`)
    task.value.deadline = formatDatetimeLocal(combinedDateTime)
  }
}

// Watch for changes in date/time inputs
watch([deadlineDate, deadlineTime], updateTaskDeadline)

// Initialize deadline with 1 week preset
onMounted(() => {
  setDeadlinePreset(7) // Default to 1 week from now
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
    fee: '',
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
  
  // Validate fee
  if (task.value.fee === null || task.value.fee === undefined) {
    formErrors.value.fee = 'Budget/Fee is required'
    isValid = false
  } else if (task.value.fee <= 0) {
    formErrors.value.fee = 'Budget/Fee must be greater than UGX 0'
    isValid = false
  } else if (task.value.fee > 50000000) {
    formErrors.value.fee = 'Budget/Fee cannot exceed UGX 50,000,000'
    isValid = false
  }
  
  // Validate deadline
  if (!deadlineDate.value || !deadlineTime.value) {
    formErrors.value.deadline = 'Please select both date and time for the deadline'
    isValid = false
  } else if (!validateDeadline(task.value.deadline)) {
    formErrors.value.deadline = 'Deadline must be at least 24 hours in the future'
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
      fee: task.value.fee!, // Use the actual fee from the form
      is_messages_public: task.value.isMessagesPublic
    }
    
    // Import and use the tasks store
    const tasksStore = useTasksStore()
    await tasksStore.createTask(taskData)
    
    await router.push({ name: 'tasks' })
  } catch (error) {
    console.error('Failed to create task:', error)
    formError.value = error instanceof Error ? error.message : 'Failed to create gig. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
/* Deadline preset buttons */
.deadline-presets {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.preset-btn {
  padding: 0.5rem 1rem;
  border: 2px solid #e9ecef;
  background: white;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  color: #6c757d;
}

.preset-btn:hover {
  border-color: #1976d2;
  color: #1976d2;
}

.preset-btn.active {
  background: #1976d2;
  border-color: #1976d2;
  color: white;
}

/* Date and time input layout */
.datetime-inputs {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 1rem;
  margin-bottom: 0.5rem;
}

.date-input-group,
.time-input-group {
  display: flex;
  flex-direction: column;
}

.input-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  margin-bottom: 0.25rem;
}

/* Privacy toggle styles */
.privacy-toggle-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 0.5rem;
}

.privacy-option {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  border: 2px solid #e9ecef;
  border-radius: 8px;
  transition: all 0.2s;
  cursor: pointer;
}

.privacy-option:hover {
  border-color: #1976d2;
  background: #f8fafc;
}

.privacy-option:has(.privacy-radio:checked) {
  border-color: #1976d2;
  background: #e3f2fd;
}

.privacy-radio {
  margin-top: 0.125rem;
  accent-color: #1976d2;
}

.privacy-label {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  cursor: pointer;
  flex: 1;
}

.privacy-title {
  font-weight: 600;
  color: #374151;
  font-size: 0.9rem;
}

.privacy-description {
  color: #6b7280;
  font-size: 0.8rem;
  line-height: 1.4;
}

/* Responsive adjustments */
@media (max-width: 640px) {
  .deadline-presets {
    flex-direction: column;
  }
  
  .preset-btn {
    width: 100%;
    text-align: center;
  }
  
  .datetime-inputs {
    grid-template-columns: 1fr;
  }

  .privacy-toggle-container {
    gap: 0.75rem;
  }

  .privacy-option {
    padding: 0.75rem;
  }
}
</style>
