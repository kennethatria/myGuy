<template>
  <div class="reviews-container">
    <div class="reviews-header">
      <h3>Reviews</h3>
      <div v-if="reviews.length > 0" class="rating-summary">
        <div class="average-rating">
          <span class="rating-value">{{ averageRating }}</span>
          <div class="stars">
            <svg 
              v-for="star in 5" 
              :key="star"
              xmlns="http://www.w3.org/2000/svg" 
              width="20" 
              height="20" 
              viewBox="0 0 24 24" 
              :fill="star <= Math.round(averageRating) ? '#ffd700' : '#ddd'"
            >
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
            </svg>
          </div>
          <span class="review-count">({{ reviews.length }} {{ reviews.length === 1 ? 'review' : 'reviews' }})</span>
        </div>
      </div>
    </div>

    <div v-if="loading" class="text-center py-4">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading reviews...</span>
      </div>
    </div>

    <div v-else-if="error" class="alert alert-danger">
      {{ error }}
    </div>

    <div v-else-if="reviews.length === 0" class="no-reviews">
      <p class="text-muted">No reviews yet.</p>
    </div>

    <div v-else class="reviews-list">
      <div v-for="review in reviews" :key="review.id" class="review-item">
        <div class="review-header">
          <div class="reviewer-info">
            <strong>{{ review.reviewer?.username || 'Anonymous' }}</strong>
            <span class="review-date">{{ formatDate(review.created_at) }}</span>
          </div>
          <div class="review-rating">
            <svg 
              v-for="star in 5" 
              :key="star"
              xmlns="http://www.w3.org/2000/svg" 
              width="16" 
              height="16" 
              viewBox="0 0 24 24" 
              :fill="star <= review.rating ? '#ffd700' : '#ddd'"
            >
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
            </svg>
          </div>
        </div>
        <div class="review-content">
          <p v-if="review.task" class="task-reference">
            <small>Task: <em>{{ review.task.title }}</em></small>
          </p>
          <p class="review-comment">{{ review.comment }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useReviewsStore } from '@/stores/reviews'

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

interface Props {
  reviews: Review[]
  loading?: boolean
  error?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  error: null
})

const reviewsStore = useReviewsStore()

const averageRating = computed(() => {
  return reviewsStore.calculateAverageRating(props.reviews)
})

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  const now = new Date()
  const diffInHours = (now.getTime() - date.getTime()) / (1000 * 60 * 60)
  
  if (diffInHours < 24) {
    if (diffInHours < 1) {
      const diffInMinutes = Math.floor(diffInHours * 60)
      return `${diffInMinutes} ${diffInMinutes === 1 ? 'minute' : 'minutes'} ago`
    }
    const hours = Math.floor(diffInHours)
    return `${hours} ${hours === 1 ? 'hour' : 'hours'} ago`
  } else if (diffInHours < 168) { // 7 days
    const days = Math.floor(diffInHours / 24)
    return `${days} ${days === 1 ? 'day' : 'days'} ago`
  } else {
    return date.toLocaleDateString()
  }
}
</script>

<style scoped>
.reviews-container {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.reviews-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e0e0e0;
}

.reviews-header h3 {
  margin: 0;
}

.rating-summary {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.average-rating {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.rating-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #333;
}

.stars {
  display: flex;
  gap: 2px;
}

.review-count {
  color: #666;
  font-size: 0.875rem;
}

.no-reviews {
  text-align: center;
  padding: 2rem;
}

.reviews-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.review-item {
  padding: 1rem;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  background-color: #f9f9f9;
}

.review-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.reviewer-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.reviewer-info strong {
  color: #333;
}

.review-date {
  font-size: 0.875rem;
  color: #666;
}

.review-rating {
  display: flex;
  gap: 2px;
}

.review-content {
  color: #555;
}

.task-reference {
  margin-bottom: 0.5rem;
  color: #666;
}

.review-comment {
  margin: 0;
  line-height: 1.5;
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
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
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

.text-muted {
  color: #6c757d;
}

.text-center {
  text-align: center;
}

.py-4 {
  padding-top: 1.5rem;
  padding-bottom: 1.5rem;
}
</style>