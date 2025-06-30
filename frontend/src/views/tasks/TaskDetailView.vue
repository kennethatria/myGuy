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
          <div class="gig-chat-header">
            <h3 class="chat-title">Communication</h3>
            <div class="chat-status">
              <span v-if="task?.status === 'open'" class="status-badge status-open">Open for Applications</span>
              <span v-else-if="task?.status === 'assigned'" class="status-badge status-assigned">Task Assigned</span>
              <span v-else-if="task?.status === 'completed'" class="status-badge status-completed">Completed</span>
            </div>
          </div>
          
          <div class="chat-content">
            <div v-if="messages.length === 0" class="no-messages">
              <div class="no-messages-icon">
                <i class="fas fa-comments"></i>
              </div>
              <p>No messages yet</p>
              <p class="no-messages-subtitle" v-if="task?.status === 'open'">
                Apply for this gig to start messaging
              </p>
              <p class="no-messages-subtitle" v-else-if="task?.status === 'assigned'">
                Start the conversation about this task
              </p>
            </div>
            
            <div v-else class="chat-messages">
              <div v-for="message in messages" :key="message.id" class="message" :class="{ 'own-message': message.sender?.id === authStore.user?.id }">
                <div class="message-header">
                  <span class="sender">{{ message.sender?.id === authStore.user?.id ? 'You' : (message.sender?.username || 'Unknown User') }}</span>
                  <span class="timestamp">{{ formatMessageTime(message.createdAt || new Date()) }}</span>
                </div>
                <div class="message-content">{{ message.content || 'No message content' }}</div>
              </div>
            </div>
          </div>
          
          <!-- Message input section -->
          <div class="chat-input-section">
            <div v-if="canSendMessage && userCanSendMore" class="chat-input">
              <form @submit.prevent="handleSendMessage">
                <textarea 
                  v-model="newMessage"
                  placeholder="Type your message about this gig..."
                  :maxlength="500"
                  rows="3"
                  @keydown.enter.ctrl="handleSendMessage"
                  class="message-textarea"
                ></textarea>
                <div class="input-footer">
                  <div class="message-info">
                    <span class="message-count">{{ userMessageCount }}/{{ currentMessageLimit }} messages sent</span>
                    <span v-if="!isTaskAssigned && !isOwner" class="limit-info">
                      • Limit increases to 15 when task is assigned
                    </span>
                  </div>
                  <button 
                    type="submit"
                    :disabled="!newMessage.trim()"
                    class="btn btn-primary btn-sm"
                  >
                    Send
                  </button>
                </div>
              </form>
            </div>
            
            <div v-else-if="canSendMessage && !userCanSendMore" class="message-limit-reached">
              <div class="limit-reached-content">
                <i class="fas fa-info-circle"></i>
                <div>
                  <p><strong>Message limit reached</strong></p>
                  <p v-if="!isTaskAssigned && !isOwner" class="suggestion">
                    The task owner can assign this task to you to unlock more messages (up to 15 total).
                  </p>
                  <p v-else class="suggestion">
                    You've reached the maximum number of messages for this conversation.
                  </p>
                </div>
              </div>
            </div>
            
            <div v-else-if="isOwner && task?.status === 'open'" class="assignment-required">
              <div class="assignment-content">
                <i class="fas fa-user-plus"></i>
                <div>
                  <p><strong>Assign this task to enable messaging</strong></p>
                  <p>Messages will be available once you assign this task to an applicant.</p>
                </div>
              </div>
            </div>
            
            <div v-else-if="!isOwner && task?.status === 'open'" class="application-required">
              <div class="application-content">
                <i class="fas fa-paper-plane"></i>
                <div>
                  <p><strong>Apply for this gig to start messaging</strong></p>
                  <p>Submit an application to communicate with the task owner.</p>
                </div>
              </div>
            </div>
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

const formatMessageTime = (dateString: string | Date): string => {
  try {
    const date = new Date(dateString)
    return format(date, 'MMM d, h:mm a')
  } catch (error) {
    return 'Unknown time'
  }
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

// Message limit computed properties
const isTaskAssigned = computed(() => {
  return task.value?.status === 'assigned' || task.value?.assigned_to !== null
})

const currentMessageLimit = computed(() => {
  // 3 messages before assignment, 15 after assignment
  return isTaskAssigned.value || isOwner.value ? 15 : 3
})

const userMessageCount = computed(() => {
  if (!authStore.user || !messages.value) return 0
  return messages.value.filter(msg => msg.sender?.id === authStore.user?.id).length
})

const userCanSendMore = computed(() => {
  return userMessageCount.value < currentMessageLimit.value
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

  // Check message limit
  if (!userCanSendMore.value) {
    alert(`You've reached the message limit (${currentMessageLimit.value} messages). ${isTaskAssigned.value ? '' : 'The limit will increase to 15 once the task is assigned.'}`)
    return
  }

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

<style scoped>
/* Gig Chat Interface Styles */
.gig-chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e5e7eb;
}

.chat-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.chat-status {
  display: flex;
  align-items: center;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.status-open {
  background: #dbeafe;
  color: #1e40af;
}

.status-assigned {
  background: #fef3c7;
  color: #92400e;
}

.status-completed {
  background: #d1fae5;
  color: #065f46;
}

.chat-content {
  margin-bottom: 1.5rem;
}

.no-messages {
  text-align: center;
  padding: 3rem 1rem;
  color: #6b7280;
}

.no-messages-icon {
  font-size: 3rem;
  color: #d1d5db;
  margin-bottom: 1rem;
}

.no-messages-subtitle {
  font-size: 0.875rem;
  color: #9ca3af;
  margin-top: 0.5rem;
}

.chat-messages {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-height: 400px;
  overflow-y: auto;
  padding: 0.5rem;
}

.message {
  padding: 0.75rem;
  border-radius: 0.5rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
}

.message.own-message {
  background: #dbeafe;
  border-color: #93c5fd;
  margin-left: 2rem;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.sender {
  font-weight: 600;
  color: #374151;
  font-size: 0.875rem;
}

.timestamp {
  font-size: 0.75rem;
  color: #6b7280;
}

.message-content {
  color: #111827;
  line-height: 1.5;
  white-space: pre-wrap;
}

.chat-input-section {
  border-top: 1px solid #e5e7eb;
  padding-top: 1rem;
}

.chat-input form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.message-textarea {
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  padding: 0.75rem;
  font-size: 0.875rem;
  resize: vertical;
  min-height: 80px;
  font-family: inherit;
  width: 100%;
}

.message-textarea:focus {
  outline: none;
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.input-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.message-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.message-count {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
}

.limit-info {
  font-size: 0.75rem;
  color: #059669;
}

.btn-sm {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
}

.message-limit-reached,
.assignment-required,
.application-required {
  background: #f3f4f6;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  padding: 1rem;
}

.limit-reached-content,
.assignment-content,
.application-content {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.limit-reached-content i,
.assignment-content i,
.application-content i {
  color: #6b7280;
  font-size: 1.25rem;
  margin-top: 0.125rem;
}

.limit-reached-content div p:first-child,
.assignment-content div p:first-child,
.application-content div p:first-child {
  margin: 0 0 0.5rem 0;
  color: #374151;
}

.suggestion {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

/* Mobile responsiveness */
@media (max-width: 768px) {
  .gig-chat-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }
  
  .message.own-message {
    margin-left: 1rem;
  }

  .input-footer {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .message-info {
    align-self: stretch;
  }

  .btn-sm {
    align-self: flex-end;
  }
}
</style>
