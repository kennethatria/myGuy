<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div v-if="task" class="bg-white shadow overflow-hidden sm:rounded-lg">
      <div class="px-4 py-5 sm:px-6">
        <div class="flex justify-between items-center">
          <h3 class="text-lg leading-6 font-medium text-gray-900">{{ task.title }}</h3>
          <span
            :class="[
              statusClasses[task.status],
              'px-2 inline-flex text-xs leading-5 font-semibold rounded-full'
            ]"
          >
            {{ task.status }}
          </span>
        </div>
      </div>
      <div class="border-t border-gray-200 px-4 py-5 sm:px-6">
        <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <dt class="text-sm font-medium text-gray-500">Description</dt>
            <dd class="mt-1 text-sm text-gray-900">{{ task.description }}</dd>
          </div>
          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-gray-500">Created by</dt>
            <dd class="mt-1 text-sm text-gray-900">{{ task.createdBy }}</dd>
          </div>
          <div class="sm:col-span-1">
            <dt class="text-sm font-medium text-gray-500">Deadline</dt>
            <dd class="mt-1 text-sm text-gray-900">{{ formatDate(task.deadline) }}</dd>
          </div>
          <div v-if="task.assignedTo" class="sm:col-span-1">
            <dt class="text-sm font-medium text-gray-500">Assigned to</dt>
            <dd class="mt-1 text-sm text-gray-900">{{ task.assignedTo }}</dd>
          </div>
          <div v-if="task.fee" class="sm:col-span-1">
            <dt class="text-sm font-medium text-gray-500">Fee</dt>
            <dd class="mt-1 text-sm text-gray-900">${{ task.fee }}</dd>
          </div>
        </dl>
      </div>

      <!-- Action buttons based on task status and user role -->
      <div class="border-t border-gray-200 px-4 py-5 sm:px-6">
        <div class="flex justify-end space-x-3">
          <button
            v-if="canApply"
            @click="handleApply"
            class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
          >
            Apply for Task
          </button>
          <button
            v-if="canComplete"
            @click="handleComplete"
            class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
          >
            Mark as Complete
          </button>
        </div>
      </div>

      <!-- Applications section for task owner -->
      <div v-if="isOwner && applications.length > 0" class="border-t border-gray-200">
        <div class="px-4 py-5 sm:px-6">
          <h4 class="text-lg font-medium text-gray-900">Applications</h4>
          <ul role="list" class="mt-4 divide-y divide-gray-200">
            <li v-for="application in applications" :key="application.id" class="py-4">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-900">{{ application.applicant.username }}</p>
                  <p class="text-sm text-gray-500">Proposed fee: ${{ application.proposedFee }}</p>
                </div>
                <div v-if="application.status === 'pending'" class="flex space-x-2">
                  <button
                    @click="handleAcceptApplication(application.id)"
                    class="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                  >
                    Accept
                  </button>
                  <button
                    @click="handleDeclineApplication(application.id)"
                    class="inline-flex items-center px-3 py-1.5 border border-gray-300 text-xs font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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
        <div class="px-4 py-5 sm:px-6">
          <h4 class="text-lg font-medium text-gray-900">Messages</h4>
          <div class="mt-4 space-y-4">
            <div v-for="message in messages" :key="message.id" class="flex space-x-3">
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h3 class="text-sm font-medium text-gray-900">{{ message.sender.username }}</h3>
                  <p class="text-sm text-gray-500">{{ formatDate(message.createdAt) }}</p>
                </div>
                <p class="mt-1 text-sm text-gray-900">{{ message.content }}</p>
              </div>
            </div>
          </div>
          <!-- New message form -->
          <div class="mt-6">
            <form @submit.prevent="handleSendMessage" class="flex space-x-3">
              <div class="flex-1">
                <input
                  type="text"
                  v-model="newMessage"
                  placeholder="Type your message..."
                  class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
                />
              </div>
              <button
                type="submit"
                class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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

const statusClasses = {
  open: 'bg-green-100 text-green-800',
  in_progress: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-gray-100 text-gray-800'
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

onMounted(async () => {
  const taskId = parseInt(route.params.id as string)
  try {
    const [taskData, applicationsData, messagesData] = await Promise.all([
      tasksStore.getTask(taskId),
      tasksStore.getTaskApplications(taskId),
      messagesStore.fetchTaskMessages(taskId)
    ])
    
    task.value = taskData
    applications.value = applicationsData
    messages.value = messagesData
  } catch (error) {
    console.error('Failed to fetch task details:', error)
    // Show error notification
  }
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
