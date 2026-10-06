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
      <div class="p-4 note-stage">
        <StickyNote :seed="task.id" size="large">
          <template #header>
            <span class="badge note-status" :class="statusClasses[task.status]">
              {{ statusLabel }}
            </span>
            <h1 class="note-detail-headline">{{ task.title }}</h1>
            <p class="note-detail-body">{{ task.description }}</p>
          </template>
          <template #footer>
            <span>Posted {{ formatDate(task.created_at) }}</span>
            <span v-if="task.status === 'open' && expiryLabel(task.deadline)">{{ expiryLabel(task.deadline) }}</span>
          </template>
        </StickyNote>
      </div>
      <div class="p-4 border-t border-gray-200">
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
            <p v-else>{{ task.created_by ? 'User ' + task.created_by : 'Unknown User' }}</p>
          </div>
          <div v-if="task.assigned_to || task.assignee || assignee">
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
            <p v-else>{{ task.assigned_to ? 'User ' + task.assigned_to : 'Not assigned' }}</p>
          </div>
        </div>
      </div>

      <!-- Action buttons based on task status and user role -->
      <div class="border-t border-gray-200 p-4">
        <p v-if="applyNotice" class="apply-notice" role="status">{{ applyNotice }}</p>
        <p v-if="applyError" class="apply-error" role="alert">{{ applyError }}</p>
        <p v-if="isOwner && task.status === 'expired'" class="expired-note">
          Nobody replied within 24 hours, so this note came off the board.
        </p>
        <div class="task-actions flex justify-end space-x-3">
          <button
            v-if="isOwner && task.status === 'expired'"
            @click="handleRepost"
            class="btn btn-primary"
          >
            Repost for 24 hours
          </button>
          <button
            v-if="canApply"
            @click="handleApply"
            class="btn btn-primary"
            :disabled="applying"
          >
            {{ applying ? 'Applying...' : 'Apply' }}
          </button>
          <button
            v-if="chatPartner"
            @click="openChat"
            class="btn btn-outline"
          >
            Message {{ chatPartner.name }}
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
            @click="() => router.push(`/reviews/create/${task!.id}`)"
            class="btn btn-primary"
          >
            Leave Review
          </button>
        </div>
      </div>

      <!-- Applications: the poster's to answer -->
      <div v-if="isOwner && applications.length > 0" class="border-t border-gray-200">
        <div class="p-4">
          <h3 class="mb-3">Applications</h3>
          <div class="space-y-3">
            <ApplicationDetail
              v-for="application in applications"
              :key="application.id"
              :application="application"
              :task-owner-id="task.created_by"
              @accept="handleAcceptApplication"
              @decline="handleDeclineApplication"
              @message="handleMessageApplicant"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { setPageTitle } from '@/utils/pageTitle'
import { useRoute, useRouter } from 'vue-router'
import { format } from 'date-fns'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import { useChatStore } from '@/stores/chat'
import { useUsersStore } from '@/stores/users'
import { useReviewsStore } from '@/stores/reviews'
import ApplicationDetail from '@/components/ApplicationDetail.vue'
import StickyNote from '@/components/StickyNote.vue'
import { expiryLabel } from '@/utils/gigNote'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const tasksStore = useTasksStore()
const chatStore = useChatStore()
const usersStore = useUsersStore()
const reviewsStore = useReviewsStore()

interface Task {
  id: number
  title: string
  description: string
  status: 'open' | 'in_progress' | 'completed' | 'cancelled' | 'expired'
  created_by: number  // Changed from createdBy to match API
  assigned_to?: number  // Changed from assignedTo to match API
  deadline: string
  created_at: string
  is_messages_public?: boolean
  
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
  task_id: number
  applicant_id: number
  applicant: {
    id: number
    username: string
  }
  status: 'pending' | 'accepted' | 'declined'
  message?: string
  created_at: string
}

const task = ref<Task | null>(null)
const applications = ref<Application[]>([])
const hasReviewed = ref(false)
const isLoading = ref(true)
const error = ref('')
const creator = ref<{ id: number; username: string; fullName?: string } | null>(null)
const assignee = ref<{ id: number; username: string; fullName?: string } | null>(null)

const statusClasses = {
  open: 'badge-open',
  in_progress: 'badge-in_progress',
  completed: 'badge-completed',
  cancelled: 'badge-cancelled',
  expired: 'badge-cancelled'
}

const statusLabel = computed(() => (task.value?.status ?? '').replace('_', ' '))

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy h:mm a')
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
    (task.value.created_by === userId || task.value.assigned_to === userId)
  )
})

const hasApplied = computed(() => {
  if (!authStore.user || !applications.value) return false
  return applications.value.some(app => app.applicant.id === authStore.user?.id)
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
    
    task.value = taskData as unknown as Task;
    setPageTitle(task.value.title);
    console.log('Task data loaded successfully:', task.value);
    
    // Try to load user info for task creator and assignee
    if (task.value?.created_by && (!task.value?.creator || !task.value?.creator.username)) {
      try {
        console.log(`Fetching creator info for user ID ${task.value.created_by}`);
        const creatorData = await usersStore.getUserById(Number(task.value.created_by));
        if (creatorData) {
          creator.value = creatorData;
          // Also update the task creator for consistency
          if (task.value && !task.value.creator) {
            task.value.creator = creatorData;
          }
        }
      } catch (error) {
        console.error('Failed to fetch creator info:', error);
      }
    }
    
    if (task.value?.assigned_to && (!task.value?.assignee || !task.value?.assignee.username)) {
      try {
        console.log(`Fetching assignee info for user ID ${task.value.assigned_to}`);
        const assigneeData = await usersStore.getUserById(Number(task.value.assigned_to));
        if (assigneeData) {
          assignee.value = assigneeData;
          // Also update the task assignee for consistency
          if (task.value && !task.value.assignee) {
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
    applications.value = (applicationsData || []) as unknown as Application[];
    console.log(`Loaded ${applications.value.length} applications`);

    // Check if the current user has already reviewed this task
    if (task.value?.status === 'completed' && authStore.user) {
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

  // Connect to chat socket
  if (!chatStore.connected) {
    await chatStore.connectSocket()
  }
})

const errorMessage = (error: unknown, fallback: string) =>
  error instanceof Error && error.message ? error.message : fallback

// Applying is one tap: price and details are agreed in chat afterwards
const applying = ref(false)
const applyNotice = ref('')
const applyError = ref('')

const handleApply = async () => {
  if (!task.value || applying.value) return

  applying.value = true
  applyNotice.value = ''
  applyError.value = ''
  try {
    await tasksStore.applyForTask(task.value.id, { message: '' })
    applications.value = await tasksStore.getTaskApplications(task.value.id) as unknown as Application[]
    applyNotice.value = 'Applied. The poster can see it in Messages.'
  } catch (error) {
    console.error('Failed to apply for task:', error)
    applyError.value = errorMessage(error, 'Could not apply for the gig. Please try again.')
  } finally {
    applying.value = false
  }
}

// Expired notes go back on the board for a fresh 24 hours
const handleRepost = async () => {
  if (!task.value) return

  try {
    const updated = await tasksStore.updateTaskStatus(task.value.id, 'open')
    task.value.status = 'open'
    if (updated?.deadline) task.value.deadline = updated.deadline
  } catch (error) {
    console.error('Failed to repost task:', error)
    alert(errorMessage(error, 'Failed to repost the note. Please try again.'))
  }
}

const handleComplete = async () => {
  if (!task.value) return
  
  if (!confirm('Mark this gig as completed?')) {
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
    alert('Failed to complete the gig. Please try again.')
  }
}

const handleAcceptApplication = async (applicationId: number) => {
  if (!task.value) return

  try {
    await tasksStore.respondToApplication(task.value.id, applicationId, 'accepted')
  } catch (error) {
    console.error('Failed to accept application:', error)
    alert(errorMessage(error, 'Failed to accept application. Please try again.'))
  }

  // Reload either way: on success the task now has an assignee and the other
  // applications are declined; on failure (e.g. already assigned) the page
  // catches up with the server.
  await loadTaskData()
}

const handleDeclineApplication = async (applicationId: number) => {
  if (!task.value) return

  try {
    await tasksStore.respondToApplication(task.value.id, applicationId, 'declined')
    
    // Refresh applications list
    applications.value = await tasksStore.getTaskApplications(task.value.id) as unknown as Application[]
  } catch (error) {
    console.error('Failed to decline application:', error)
    alert(errorMessage(error, 'Failed to decline application. Please try again.'))
  }
}

// Who this person talks to about the gig, in the floating chat: the poster
// and the assignee with each other, and an applicant with the poster. The
// poster reaches other applicants from their application.
const chatPartner = computed<{ id: number; name: string } | null>(() => {
  if (!task.value || !authStore.user) return null
  if (isOwner.value) {
    const id = task.value.assigned_to
    return id ? { id, name: task.value.assignee?.username || assignee.value?.username || 'the assignee' } : null
  }
  if (!hasApplied.value && task.value.assigned_to !== authStore.user.id) return null
  return { id: task.value.created_by, name: task.value.creator?.username || creator.value?.username || 'the poster' }
})

const openChat = () => {
  if (!task.value || !chatPartner.value) return
  chatStore.openChat({ taskId: task.value.id, otherUserId: chatPartner.value.id, otherUserName: chatPartner.value.name })
}

const handleMessageApplicant = (applicantId: number) => {
  if (!task.value) return
  const applicant = applications.value.find(app => app.applicant_id === applicantId)?.applicant
  chatStore.openChat({ taskId: task.value.id, otherUserId: applicantId, otherUserName: applicant?.username })
}
</script>

<style scoped>
.note-stage {
  max-width: 560px;
  margin: 0 auto;
}

.note-status {
  align-self: flex-start;
  text-transform: capitalize;
}

.note-detail-headline {
  margin: 0;
  font-size: 1.6rem;
  font-weight: 700;
  line-height: 1.25;
}

.note-detail-body {
  margin: 0;
  font-size: 1.15rem;
  line-height: 1.45;
  flex: 1;
}

.expired-note {
  margin: 0 0 0.75rem;
  color: var(--color-text-light, #6b7280);
}

.apply-notice,
.apply-error {
  margin: 0 0 0.75rem;
}

.apply-notice {
  color: #166534;
}

.apply-error {
  color: var(--color-danger, #dc2626);
}

/* Mobile responsiveness */
@media (max-width: 768px) {
  /* Full-width actions stay readable and tappable beside the floating chat button */
  .task-actions {
    flex-direction: column;
    gap: 0.5rem;
  }

  .task-actions > .btn {
    width: 100%;
    margin-left: 0;
  }
}
</style>
