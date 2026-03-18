<template>
  <div class="container py-4">
    <div class="row justify-content-center">
      <div class="col-md-8">
        <div v-if="loading" class="text-center py-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
        </div>

        <div v-else-if="error" class="alert alert-danger">
          {{ error }}
        </div>

        <div v-else-if="!canReview" class="alert alert-warning">
          <h4>Cannot Review This Task</h4>
          <p>{{ reviewError }}</p>
          <router-link :to="`/tasks/${taskId}`" class="btn btn-primary mt-2">
            Back to Task
          </router-link>
        </div>

        <div v-else>
          <div class="task-info mb-4">
            <h2>Review for Task: {{ task?.title }}</h2>
            <p class="text-muted">
              You are reviewing: <strong>{{ reviewedUserName }}</strong>
            </p>
          </div>

          <ReviewForm 
            :key="taskId"
            :task-id="taskId"
            :reviewed-user-id="reviewedUserId"
            @review-submitted="handleReviewSubmitted"
            @cancel="handleCancel"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'
import { useReviewsStore } from '@/stores/reviews'
import ReviewForm from '@/components/ReviewForm.vue'

const route = useRoute()
const router = useRouter()
const tasksStore = useTasksStore()
const authStore = useAuthStore()
const reviewsStore = useReviewsStore()

const taskId = computed(() => Number(route.params.taskId))
const task = ref<{
  id: number;
  title?: string;
  status?: string;
  created_by?: number;
  createdBy?: number;
  assignedTo?: number;
  creator?: { id: number; username: string };
  assignee?: { id: number; username: string };
} | null>(null)
const loading = ref(true)
const error = ref('')
const canReview = ref(false)
const reviewError = ref('')

const currentUserId = computed(() => authStore.user?.id)

const reviewedUserId = computed((): number | undefined => {
  if (!task.value || !currentUserId.value) return undefined
  
  // If current user is the task creator, they review the assignee
  if (task.value.createdBy === currentUserId.value) {
    return task.value.assignedTo
  }
  
  // If current user is the assignee, they review the creator
  if (task.value.assignedTo === currentUserId.value) {
    return task.value.createdBy
  }
  
  return undefined
})

const reviewedUserName = computed(() => {
  if (!task.value || !reviewedUserId.value) return 'Unknown'
  
  // Check if we're reviewing the creator
  if (reviewedUserId.value === task.value.createdBy) {
    return task.value.creator?.username || 'Task Creator'
  }
  
  // Check if we're reviewing the assignee
  if (reviewedUserId.value === task.value.assignedTo) {
    return task.value.assignee?.username || 'Task Assignee'
  }
  
  return 'Unknown'
})

const checkCanReview = async () => {
  canReview.value = false
  reviewError.value = ''
  
  if (!task.value) {
    reviewError.value = 'Task not found'
    return
  }
  
  if (task.value.status !== 'completed') {
    reviewError.value = 'This task must be completed before it can be reviewed.'
    return
  }
  
  if (!currentUserId.value) {
    reviewError.value = 'You must be logged in to leave a review.'
    return
  }
  
  // Check if user is a participant (either creator or assignee)
  const isCreator = task.value.createdBy === currentUserId.value
  const isAssignee = task.value.assignedTo === currentUserId.value
  
  if (!isCreator && !isAssignee) {
    reviewError.value = 'You can only review tasks you created or were assigned to.'
    return
  }
  
  if (!reviewedUserId.value) {
    reviewError.value = 'Cannot determine who to review.'
    return
  }
  
  // Check if already reviewed
  try {
    const hasReviewed = await reviewsStore.hasReviewedTask(taskId.value)
    if (hasReviewed) {
      reviewError.value = 'You have already reviewed this task.'
      return
    }
  } catch (err) {
    console.error('Error checking review status:', err)
  }
  
  canReview.value = true
}

const handleReviewSubmitted = async () => {
  // Navigate back to task detail
  router.push(`/tasks/${taskId.value}`)
}

const handleCancel = () => {
  router.push(`/tasks/${taskId.value}`)
}

onMounted(async () => {
  try {
    loading.value = true
    task.value = await tasksStore.getTask(taskId.value)
    await checkCanReview()
  } catch (err) {
    error.value = 'Failed to load task details'
    console.error('Error loading task:', err)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.container {
  max-width: 1200px;
  margin: 0 auto;
}

.py-4 {
  padding-top: 1.5rem;
  padding-bottom: 1.5rem;
}

.py-5 {
  padding-top: 3rem;
  padding-bottom: 3rem;
}

.row {
  display: flex;
  flex-wrap: wrap;
  margin-right: -15px;
  margin-left: -15px;
}

.justify-content-center {
  justify-content: center;
}

.col-md-8 {
  flex: 0 0 66.666667%;
  max-width: 66.666667%;
  padding-right: 15px;
  padding-left: 15px;
}

.task-info {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
  border-left: 4px solid #007bff;
}

.task-info h2 {
  margin: 0 0 0.5rem 0;
  color: #333;
}

.text-muted {
  color: #6c757d;
}

.alert {
  padding: 1rem 1.25rem;
  margin-bottom: 1rem;
  border: 1px solid transparent;
  border-radius: 0.25rem;
}

.alert h4 {
  margin-top: 0;
  margin-bottom: 0.5rem;
}

.alert-danger {
  color: #721c24;
  background-color: #f8d7da;
  border-color: #f5c6cb;
}

.alert-warning {
  color: #856404;
  background-color: #fff3cd;
  border-color: #ffeaa7;
}

.btn {
  display: inline-block;
  font-weight: 400;
  text-align: center;
  white-space: nowrap;
  vertical-align: middle;
  user-select: none;
  border: 1px solid transparent;
  padding: 0.375rem 0.75rem;
  font-size: 1rem;
  line-height: 1.5;
  border-radius: 0.25rem;
  transition: color 0.15s ease-in-out, background-color 0.15s ease-in-out;
  text-decoration: none;
}

.btn-primary {
  color: #fff;
  background-color: #007bff;
  border-color: #007bff;
}

.btn-primary:hover {
  color: #fff;
  background-color: #0056b3;
  border-color: #004085;
}

.mt-2 {
  margin-top: 0.5rem;
}

.mb-4 {
  margin-bottom: 1.5rem;
}

.spinner-border {
  display: inline-block;
  width: 2rem;
  height: 2rem;
  vertical-align: text-bottom;
  border: 0.25em solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spinner-border 0.75s linear infinite;
}

@keyframes spinner-border {
  to { transform: rotate(360deg); }
}

.visually-hidden {
  position: absolute !important;
  width: 1px !important;
  height: 1px !important;
  padding: 0 !important;
  margin: -1px !important;
  overflow: hidden !important;
  clip: rect(0, 0, 0, 0) !important;
  white-space: nowrap !important;
  border: 0 !important;
}

.text-center {
  text-align: center;
}
</style>