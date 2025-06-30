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
            <p v-if="task.creator && task.creator.username">
              <router-link 
                :to="{ name: 'user-profile', params: { id: task.creator.id } }" 
                class="text-primary hover:underline"
              >
                {{ task.creator.username }}
              </router-link>
            </p>
            <p v-else-if="creator && creator.username">
              <router-link 
                :to="{ name: 'user-profile', params: { id: creator.id } }" 
                class="text-primary hover:underline"
              >
                {{ creator.username }}
              </router-link>
            </p>
            <p v-else>{{ task.createdBy ? 'User ' + task.createdBy : 'Unknown User' }}</p>
          </div>
          <div>
            <h4 class="font-medium text-sm text-gray">Deadline</h4>
            <p>{{ formatDate(task.deadline) }}</p>
          </div>
          <div v-if="task.assignedTo || task.assignee || assignee">
            <h4 class="font-medium text-sm text-gray">Assigned to</h4>
            <p v-if="task.assignee && task.assignee.username">
              <router-link 
                :to="{ name: 'user-profile', params: { id: task.assignee.id } }" 
                class="text-primary hover:underline"
              >
                {{ task.assignee.username }}
              </router-link>
            </p>
            <p v-else-if="assignee && assignee.username">
              <router-link 
                :to="{ name: 'user-profile', params: { id: assignee.id } }" 
                class="text-primary hover:underline"
              >
                {{ assignee.username }}
              </router-link>
            </p>
            <p v-else>{{ task.assignedTo ? 'User ' + task.assignedTo : 'Not assigned' }}</p>
          </div>
          <div v-if="task.fee">
            <h4 class="font-medium text-sm text-gray">Fee</h4>
            <p>UGX {{ formatCurrency(task.fee) }}</p>
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
          <button
            v-if="canReview"
            @click="() => router.push(`/reviews/create/${task.id}`)"
            class="btn btn-primary"
          >
            Leave Review
          </button>
        </div>
      </div>

      <!-- Applications section -->
      <div v-if="(isOwner || hasApplied) && applications.length > 0" class="border-t border-gray-200">
        <div class="p-4">
          <h3 class="mb-3">Applications</h3>
          <div class="space-y-3">
            <ApplicationDetail
              v-for="application in visibleApplications"
              :key="application.id"
              :application="application"
              :task-owner-id="task.created_by"
              @accept="handleAcceptApplication"
              @decline="handleDeclineApplication"
              @message-sent="handleApplicationMessageSent"
            />
          </div>
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
                  <span class="font-medium">{{ message.sender?.username || 'Unknown User' }}</span>
                  <span class="text-sm text-gray">{{ formatDate(message.createdAt || new Date()) }}</span>
                </div>
                <p>{{ message.content || 'No message content' }}</p>
              </div>
            </div>
          </div>
          
          <!-- New message form -->
          <div v-if="canSendMessage" class="mt-4">
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
          <div v-else-if="isOwner && task?.status === 'open'" class="mt-4 p-3 bg-gray-100 rounded text-gray-600">
            <p>Messages will be available once you assign this task to someone.</p>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Application Modal -->
    <ApplicationModal 
      v-if="task"
      :is-open="showApplicationModal"
      :task="task"
      @close="showApplicationModal = false"
      @submit="handleApplicationSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { format } from 'date-fns'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import { useMessagesStore } from '@/stores/messages'
import { useUsersStore } from '@/stores/users'
import { useReviewsStore } from '@/stores/reviews'
import ApplicationDetail from '@/components/ApplicationDetail.vue'
import ApplicationModal from '@/components/ApplicationModal.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const tasksStore = useTasksStore()
const messagesStore = useMessagesStore()
const usersStore = useUsersStore()
const reviewsStore = useReviewsStore()

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed'
  created_by: number  // Changed from createdBy to match API
  assigned_to?: number  // Changed from assignedTo to match API
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
const hasReviewed = ref(false)
const messages = ref<Message[]>([])
const newMessage = ref('')
const isLoading = ref(true)
const error = ref('')
const creator = ref<any>(null)
const assignee = ref<any>(null)

const statusClasses = {
  open: 'badge-open',
  in_progress: 'badge-in_progress',
  completed: 'badge-completed'
}

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy h:mm a')
}

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('en-UG', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount)
}

const isOwner = computed(() => {
  if (!task.value) return false
  return task.value.created_by === authStore.user?.id
})

const canApply = computed(() => {
  if (!task.value || !authStore.user) return false
  return (
    task.value.status === 'open' &&
    task.value.created_by !== authStore.user.id &&
    !applications.value.some(app => app.applicant.id === authStore.user?.id)
  )
})

const canSendMessage = computed(() => {
  if (!task.value || !authStore.user) return false
  
  // Non-owners can always send messages to task creators
  if (!isOwner.value) return true
  
  // Owners can only send messages if task is assigned
  return task.value.assigned_to !== null && task.value.assigned_to !== undefined
})

const canComplete = computed(() => {
  if (!task.value || !authStore.user) return false
  const userId = authStore.user.id
  return (
    task.value.status === 'in_progress' &&
    (task.value.created_by === userId || task.value.assigned_to === userId)
  )
})

const canReview = computed(() => {
  if (!task.value || !authStore.user || hasReviewed.value) return false
  const userId = authStore.user.id
  return (
    task.value.status === 'completed' &&
    (task.value.createdBy === userId || task.value.assignedTo === userId)
  )
})

const hasApplied = computed(() => {
  if (!authStore.user || !applications.value) return false
  return applications.value.some(app => app.applicant.id === authStore.user?.id)
})

const visibleApplications = computed(() => {
  if (!authStore.user || !applications.value) return []
  
  // Task owner sees all applications
  if (isOwner.value) {
    return applications.value
  }
  
  // Applicants only see their own application
  return applications.value.filter(app => app.applicant.id === authStore.user?.id)
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
    console.log(`Loading task data for ID: ${taskId}`);
    
    // Load task data first
    const taskData = await tasksStore.getTask(taskId);
    
    // Validate we have a proper task object
    if (!taskData || typeof taskData !== 'object') {
      console.error('Invalid task data received:', taskData);
      error.value = 'Could not load gig details. Please try again.';
      isLoading.value = false;
      return;
    }
    
    task.value = taskData;
    console.log('Task data loaded successfully:', task.value);
    
    // Try to load user info for task creator and assignee
    if (task.value.created_by && (!task.value.creator || !task.value.creator.username)) {
      try {
        console.log(`Fetching creator info for user ID ${task.value.created_by}`);
        const creatorData = await usersStore.getUserById(Number(task.value.created_by));
        if (creatorData) {
          creator.value = creatorData;
          // Also update the task creator for consistency
          if (!task.value.creator) {
            task.value.creator = creatorData;
          }
        }
      } catch (error) {
        console.error('Failed to fetch creator info:', error);
      }
    }
    
    if (task.value.assigned_to && (!task.value.assignee || !task.value.assignee.username)) {
      try {
        console.log(`Fetching assignee info for user ID ${task.value.assigned_to}`);
        const assigneeData = await usersStore.getUserById(Number(task.value.assigned_to));
        if (assigneeData) {
          assignee.value = assigneeData;
          // Also update the task assignee for consistency
          if (!task.value.assignee) {
            task.value.assignee = assigneeData;
          }
        }
      } catch (error) {
        console.error('Failed to fetch assignee info:', error);
      }
    }
    
    // Load applications (if not already included in task)
    let applicationsData = taskData.applications || [];
    if (!taskData.applications) {
      console.log('Applications not included in task data, fetching separately');
      try {
        applicationsData = await tasksStore.getTaskApplications(taskId);
      } catch (appErr) {
        console.error('Failed to fetch applications:', appErr);
        // Non-critical error, don't show to user but log it
        applicationsData = []; // Ensure we have an empty array at minimum
      }
    }
    applications.value = applicationsData || [];
    console.log(`Loaded ${applications.value.length} applications`);
    
    // Load messages separately since they're not included in the task data
    try {
      const messagesData = await messagesStore.fetchTaskMessages(taskId);
      messages.value = messagesData || [];
      console.log(`Loaded ${messages.value.length} messages`);
    } catch (msgErr) {
      console.error('Failed to fetch messages:', msgErr);
      // Non-critical error, don't show to user but log it
      messages.value = []; // Ensure we have an empty array
    }
    
    // Check if the current user has already reviewed this task
    if (task.value.status === 'completed' && authStore.user) {
      try {
        hasReviewed.value = await reviewsStore.hasReviewedTask(taskId);
      } catch (err) {
        console.error('Failed to check review status:', err);
        hasReviewed.value = false;
      }
    }
    
  } catch (err) {
    console.error('Failed to fetch task details:', err);
    error.value = 'Failed to load gig details. Please try again.';
  } finally {
    isLoading.value = false;
  }
}

onMounted(async () => {
  await loadTaskData()
})

const showApplicationModal = ref(false)

const handleApply = () => {
  showApplicationModal.value = true
}

const handleApplicationSubmit = async (data: { proposedFee: number; message: string }) => {
  if (!task.value) return

  try {
    const result = await tasksStore.applyForTask(task.value.id, data)
    console.log('Application result:', result)
    
    showApplicationModal.value = false
    
    // Refresh applications list
    applications.value = await tasksStore.getTaskApplications(task.value.id)
    
    // Show success message
    alert('Application submitted successfully!')
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

const handleApplicationMessageSent = () => {
  // Optionally refresh applications or show a notification
  console.log('Application message sent successfully')
}

const handleSendMessage = async () => {
  if (!task.value || !newMessage.value.trim() || !authStore.user) return

  try {
    let recipientId: number | undefined
    
    console.log('Sending message - Debug info:', {
      isOwner: isOwner.value,
      taskCreatedBy: task.value.created_by,
      taskAssignedTo: task.value.assigned_to,
      currentUserId: authStore.user.id,
      taskCreator: task.value.creator
    })
    
    if (isOwner.value) {
      // Task owner sending message
      if (task.value.assigned_to) {
        // If task is assigned, send to assignee
        recipientId = task.value.assigned_to
      } else {
        // For open tasks, owner can't send messages to themselves
        alert('This task is not assigned yet. Messages will be available once someone is assigned.')
        return
      }
    } else {
      // Non-owner sending message to task creator
      recipientId = task.value.created_by || task.value.creator?.id
    }

    if (!recipientId) {
      console.error('Recipient ID not found. Task data:', task.value)
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
