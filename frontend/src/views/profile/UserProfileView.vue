<template>
  <div class="container py-4">
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else-if="error" class="alert alert-danger">
      {{ error }}
    </div>

    <div v-else-if="user">
      <div class="row">
        <!-- User information -->
        <div class="col-md-4">
          <div class="card mb-4">
            <h2>{{ user.fullName || user.username }}</h2>
            <p class="text-muted">@{{ user.username }}</p>
            
            <div class="rating-summary mt-4">
              <h4>User Rating</h4>
              <div class="flex items-center mt-2">
                <div class="rating-display">
                  <span class="rating-value">{{ averageRating.toFixed(1) }}</span>
                  <span class="rating-star">★</span>
                </div>
                <span class="text-sm text-gray ml-2">from {{ reviews.length }} reviews</span>
              </div>
            </div>

            <div v-if="user.bio" class="mt-4">
              <h4>Bio</h4>
              <p class="text-gray">{{ user.bio }}</p>
            </div>

            <div class="mt-4">
              <p class="text-sm text-gray">Member since {{ formatDate(user.created_at) }}</p>
            </div>
          </div>
        </div>

        <!-- Reviews -->
        <div class="col-md-8">
          <ReviewList 
            :reviews="reviews" 
            :loading="loadingReviews"
            :error="reviewsError"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { format } from 'date-fns'
import { useUsersStore } from '@/stores/users'
import { useReviewsStore } from '@/stores/reviews'
import { useAuthStore } from '@/stores/auth'
import ReviewList from '@/components/ReviewList.vue'

interface User {
  id: number
  username: string
  email: string
  fullName?: string
  bio?: string
  averageRating?: number
  created_at: string
}

interface Review {
  id: number
  taskId: number
  reviewerId: number
  reviewedUserId: number
  rating: number
  comment: string
  created_at: string
  reviewer?: {
    id: number
    username: string
    fullName?: string
  }
  task?: {
    id: number
    title: string
  }
}

const route = useRoute()
const router = useRouter()
const usersStore = useUsersStore()
const reviewsStore = useReviewsStore()
const authStore = useAuthStore()

const user = ref<User | null>(null)
const reviews = ref<Review[]>([])
const loading = ref(true)
const loadingReviews = ref(false)
const error = ref('')
const reviewsError = ref<string | null>(null)

const userId = computed(() => Number(route.params.id))

const averageRating = computed(() => {
  if (user.value?.averageRating !== undefined) {
    return user.value.averageRating
  }
  return reviewsStore.calculateAverageRating(reviews.value)
})

const formatDate = (dateString: string | null | undefined): string => {
  if (!dateString) {
    return 'Unknown'
  }
  
  try {
    const date = new Date(dateString)
    if (isNaN(date.getTime())) {
      return 'Unknown'
    }
    return format(date, 'MMMM yyyy')
  } catch (error) {
    console.warn('Invalid date format:', dateString)
    return 'Unknown'
  }
}

const loadUserData = async () => {
  loading.value = true
  error.value = ''
  
  try {
    // Check if viewing own profile and redirect
    if (authStore.user && authStore.user.id === userId.value) {
      router.replace('/profile')
      return
    }
    
    // Fetch user data
    const userData = await usersStore.getUserById(userId.value)
    if (!userData) {
      throw new Error('User not found')
    }
    user.value = userData
    
    // Fetch user reviews
    loadingReviews.value = true
    reviewsError.value = null
    try {
      const userReviews = await reviewsStore.fetchUserReviews(userId.value)
      reviews.value = userReviews
    } catch (err) {
      console.error('Failed to fetch reviews:', err)
      reviewsError.value = err instanceof Error ? err.message : 'Failed to load reviews'
      reviews.value = []
    } finally {
      loadingReviews.value = false
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load user profile'
    console.error('Error loading user profile:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadUserData()
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

.col-md-4 {
  flex: 0 0 33.333333%;
  max-width: 33.333333%;
  padding-right: 15px;
  padding-left: 15px;
}

.col-md-8 {
  flex: 0 0 66.666667%;
  max-width: 66.666667%;
  padding-right: 15px;
  padding-left: 15px;
}

@media (max-width: 768px) {
  .col-md-4,
  .col-md-8 {
    flex: 0 0 100%;
    max-width: 100%;
  }
}

.card {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.card h2 {
  margin: 0 0 0.5rem 0;
  color: #333;
}

.card h4 {
  margin: 0 0 0.5rem 0;
  color: #555;
  font-size: 1rem;
}

.text-muted {
  color: #6c757d;
}

.text-gray {
  color: #718096;
}

.text-sm {
  font-size: 0.875rem;
}

.rating-summary {
  border-top: 1px solid #e0e0e0;
  padding-top: 1rem;
}

.rating-display {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.rating-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #333;
}

.rating-star {
  font-size: 1.5rem;
  color: #ffd700;
}

.flex {
  display: flex;
}

.items-center {
  align-items: center;
}

.ml-2 {
  margin-left: 0.5rem;
}

.mt-2 {
  margin-top: 0.5rem;
}

.mt-4 {
  margin-top: 1.5rem;
}

.mb-4 {
  margin-bottom: 1.5rem;
}

.alert {
  padding: 0.75rem 1rem;
  border-radius: 4px;
  margin-bottom: 1rem;
}

.alert-danger {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
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