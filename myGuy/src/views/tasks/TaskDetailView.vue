<template>
  <div class="container py-4">
    <!-- Loading state -->
    <div v-if="isLoading" class="card p-4 mb-4 text-center">
      <div class="loading-spinner mb-2"></div>
      <p>Loading gig details...</p>
    </div>
    
    <!-- Error state -->
    <div v-else-if="error" class="card p-4 mb-4 bg-red-100 text-danger">
      <p>{{ error }}</p>
      <button @click="loadTaskData" class="btn btn-outline mt-2">Retry</button>
    </div>
    
    <div v-else-if="task" class="card overflow-hidden">
      <div class="p-4 pb-0">
        <div class="flex justify-between items-center mb-3">
          <h1 class="text-xl font-semibold">{{ task.title }}</h1>
          <span class="badge" :class="statusClasses[task.status]">
            {{ task.status.replace('_', ' ') }}
          </span>
        </div>
      </div>
      <div class="p-4 border-t border-gray-200">
        <div class="mb-4">
          <h3 class="font-medium mb-2">Description</h3>
          <p>{{ task.description }}</p>
        </div>
        
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-3">
          <div>
            <h4 class="font-medium text-sm text-gray">Created by</h4>
            <p>{{ task.creator ? task.creator.username : 'User ' + task.createdBy }}</p>
          </div>
          <div>
            <h4 class="font-medium text-sm text-gray">Deadline</h4>
            <p>{{ formatDate(task.deadline) }}</p>
          </div>
          <div v-if="task.assignedTo || task.assignee">
            <h4 class="font-medium text-sm text-gray">Assigned to</h4>
            <p>{{ task.assignee ? task.assignee.username : 'User ' + task.assignedTo }}</p>
          </div>
          <div v-if="task.fee">
            <h4 class="font-medium text-sm text-gray">Fee</h4>
            <p>${{ task.fee }}</p>
          </div>
        </div>
      </div>

      <!-- Action buttons based on task status and user role -->
      <div class="border-t border-gray-200 p-4">
        <div class="flex justify-end space-x-3">
          <button
            v-if="canApply"
            @click="handleApply"
            class="btn btn-primary"
          >
            Apply for Gig
          </button>
          <button
            v-if="canComplete"
            @click="handleComplete"
            class="btn btn-secondary"
          >
            Mark as Complete
          </button>
        </div>
      </div>

      <!-- Applications section for task owner -->
      <div v-if="isOwner && applications.length > 0" class="border-t border-gray-200">
        <div class="p-4">
          <h3 class="mb-3">Applications</h3>
          <ul class="divide-y">
            <li v-for="application in applications" :key="application.id" class="py-3">
              <div class="flex items-center justify-between">
                <div>
                  <p class="font-medium">{{ application.applicant.username }}</p>
                  <p class="text-sm text-gray">Proposed fee: ${{ application.proposedFee }}</p>
                </div>
                <div v-if="application.status === 'pending'" class="flex space-x-2">
                  <button
                    @click="handleAcceptApplication(application.id)"
                    class="btn btn-sm btn-primary"
                  >
                    Accept
                  </button>
                  <button
                    @click="handleDeclineApplication(application.id)"
                    class="btn btn-sm btn-outline"
                  >
                    Decline
                  </button>
                </div>
              </div>
            </li>
          </ul>
        </div>
      </div>

      <!-- Messages section -->
      <div class="border-t border-gray-200">
        <div class="p-4">
          <h3 class="mb-3">Messages</h3>
          
          <div v-if="messages.length === 0" class="p-4 text-center text-gray">
            <p>No messages yet</p>
          </div>
          
          <div v-else class="mb-4 space-y-4">
            <div v-for="message in messages" :key="message.id" class="card p-3">
              <div class="flex-1">
                <div class="flex items-center justify-between mb-1">
                  <span class="font-medium">{{ message.sender.username }}</span>
                  <span class="text-sm text-gray">{{ formatDate(message.createdAt) }}</span>
                </div>
                <p>{{ message.content }}</p>
              </div>
            </div>
          </div>
          
          <!-- New message form -->
          <div class="mt-4">
            <form @submit.prevent="handleSendMessage" class="flex space-x-2">
              <div class="flex-1">
                <input
                  type="text"
                  v-model="newMessage"
                  placeholder="Type your message..."
                  class="form-input"
                />
              </div>
              <button
                type="submit"
                class="btn btn-primary"
              >
                Send
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { format } from 'date-fns'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import { useMessagesStore } from '@/stores/messages'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const tasksStore = useTasksStore()
const messagesStore = useMessagesStore()

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed'
  createdBy: number
  assignedTo?: number
  deadline: string
  fee?: number
  created_at: string
  
  // Related data from database preloading
  creator?: {
    id: number
    username: string
    fullName?: string
  }
  assignee?: {
    id: number
    username: string
    fullName?: string
  }
  applications?: Application[]
}

interface Application {
  id: number
  applicant: {
    id: number
    username: string
  }
  proposedFee: number
  status: 'pending' | 'accepted' | 'declined'
  message?: string
}

interface Message {
  id: number
  sender: {
    id: number
    username: string
  }
  content: string
  createdAt: string
}

const task = ref<Task | null>(null)
const applications = ref<Application[]>([])
const messages = ref<Message[]>([])
const newMessage = ref('')
const isLoading = ref(true)
const error = ref('')

const statusClasses = {
  open: 'badge-open',
  in_progress: 'badge-in_progress',
  completed: 'badge-completed'
}

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy h:mm a')
}

const isOwner = computed(() => {
  if (!task.value) return false
  return task.value.createdBy === authStore.user?.id
})

const canApply = computed(() => {
  if (!task.value || !authStore.user) return false
  return (
    task.value.status === 'open' &&
    task.value.createdBy !== authStore.user.id &&
    !applications.value.some(app => app.applicant.id === authStore.user?.id)
  )
})

const canComplete = computed(() => {
  if (!task.value || !authStore.user) return false
  const userId = authStore.user.id
  return (
    task.value.status === 'in_progress' &&
    (task.value.createdBy === userId || task.value.assignedTo === userId)
  )
})

const loadTaskData = async () => {
  const taskId = parseInt(route.params.id as string)
  if (isNaN(taskId)) {
    error.value = 'Invalid gig ID. Please check the URL and try again.'
    return
  }
  
  isLoading.value = true
  error.value = ''
  
  try {
    // Load task data first
    const taskData = await tasksStore.getTask(taskId)
    task.value = taskData
    
    // If task loaded successfully, load applications (if not already included in task)
    let applicationsData = taskData.applications || []
    if (!taskData.applications) {
      try {
        applicationsData = await tasksStore.getTaskApplications(taskId)
      } catch (appErr) {
        console.error('Failed to fetch applications:', appErr)
        // Non-critical error, don't show to user but log it
      }
    }
    applications.value = applicationsData
    
    // Load messages separately since they're not included in the task data
    try {
      const messagesData = await messagesStore.fetchTaskMessages(taskId)
      messages.value = messagesData
    } catch (msgErr) {
      console.error('Failed to fetch messages:', msgErr)
      // Non-critical error, don't show to user but log it
    }
    
  } catch (err) {
    console.error('Failed to fetch task details:', err)
    error.value = 'Failed to load gig details. Please try again.'
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  await loadTaskData()
})

const handleApply = async () => {
  if (!task.value) return

  try {
    const fee = parseFloat(prompt('Enter your proposed fee:') || '0')
    if (isNaN(fee) || fee < 0) {
      alert('Please enter a valid fee amount')
      return
    }

    const message = prompt('Add a message with your application:') || ''
    await tasksStore.applyForTask(task.value.id, { proposedFee: fee, message })
    
    // Refresh applications list
    applications.value = await tasksStore.getTaskApplications(task.value.id)
  } catch (error) {
    console.error('Failed to apply for task:', error)
    alert('Failed to apply for task. Please try again.')
  }
}

const handleComplete = async () => {
  if (!task.value) return
  
  if (!confirm('Are you sure you want to mark this task as completed?')) {
    return
  }

  try {
    await tasksStore.updateTaskStatus(task.value.id, 'completed')
    task.value.status = 'completed'
    
    // If current user is task creator, prompt for review
    if (isOwner.value) {
      router.push(`/reviews/create/${task.value.id}`)
    }
  } catch (error) {
    console.error('Failed to complete task:', error)
    alert('Failed to complete task. Please try again.')
  }
}

const handleAcceptApplication = async (applicationId: number) => {
  if (!task.value) return

  try {
    await tasksStore.respondToApplication(task.value.id, applicationId, 'accepted')
    task.value.status = 'in_progress'
    
    // Refresh applications list
    applications.value = await tasksStore.getTaskApplications(task.value.id)
  } catch (error) {
    console.error('Failed to accept application:', error)
    alert('Failed to accept application. Please try again.')
  }
}

const handleDeclineApplication = async (applicationId: number) => {
  if (!task.value) return

  try {
    await tasksStore.respondToApplication(task.value.id, applicationId, 'declined')
    
    // Refresh applications list
    applications.value = await tasksStore.getTaskApplications(task.value.id)
  } catch (error) {
    console.error('Failed to decline application:', error)
    alert('Failed to decline application. Please try again.')
  }
}

const handleSendMessage = async () => {
  if (!task.value || !newMessage.value.trim() || !authStore.user) return

  try {
    const recipientId = isOwner.value 
      ? task.value.assignedTo || messages.value[0]?.sender.id
      : task.value.createdBy

    if (!recipientId) {
      throw new Error('No recipient found for message')
    }

    await messagesStore.sendMessage(task.value.id, recipientId, newMessage.value.trim())
    newMessage.value = ''
    
    // Refresh messages
    messages.value = await messagesStore.fetchTaskMessages(task.value.id)
  } catch (error) {
    console.error('Failed to send message:', error)
    alert('Failed to send message. Please try again.')
  }
}
</script>
