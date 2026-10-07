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
              <!-- Who they've worked or traded with, ratings only -->
              <router-link :to="{ name: 'user-network', params: { userId } }" class="see-network">
                See {{ user.username }}'s network →
              </router-link>
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
import { useRoute } from 'vue-router'
import { format } from 'date-fns'
import { useUsersStore } from '@/stores/users'
import { useReviewsStore, type Review } from '@/stores/reviews'
import ReviewList from '@/components/ReviewList.vue'

interface User {
  id: number
  username: string
  email?: string
  fullName?: string
  bio?: string
  averageRating?: number
  created_at?: string
}


const route = useRoute()
const usersStore = useUsersStore()
const reviewsStore = useReviewsStore()

const user = ref<User | null>(null)
const reviews = ref<Review[]>([])
const loading = ref(true)
const loadingReviews = ref(false)
const error = ref('')
const reviewsError = ref<string | null>(null)

const userId = computed(() => Number(route.params.id))

// One rating from everything others said: task reviews and store ratings
const averageRating = computed(() => reviewsStore.calculateAverageRating(reviews.value))

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
  } catch {
    console.warn('Invalid date format:', dateString)
    return 'Unknown'
  }
}

const loadUserData = async () => {
  loading.value = true
  error.value = ''

  try {
    // Fetch user data (allow viewing own profile in public view)
    const userData = await usersStore.getUserById(userId.value)
    if (!userData) {
      throw new Error('User not found')
    }
    user.value = userData
    
    // Fetch user reviews
    loadingReviews.value = true
    reviewsError.value = null
    try {
      const userReviews = await reviewsStore.fetchAllRatings(userId.value)
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

.text-center {
  text-align: center;
}

.see-network {
  display: inline-block;
  margin-top: 0.75rem;
  color: var(--color-primary, #4f46e5);
  font-weight: 600;
  font-size: 0.875rem;
}
</style>
