import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'
import { useAuthStore } from './auth'

interface Review {
  id: number
  taskId: number
  reviewerId: number
  reviewedUserId: number
  rating: number
  comment: string
  created_at: string
  
  // Related data
  reviewer?: {
    id: number
    username: string
    fullName?: string
  }
  reviewedUser?: {
    id: number
    username: string
    fullName?: string
  }
  task?: {
    id: number
    title: string
  }
}

interface CreateReviewInput {
  rating: number
  comment: string
  reviewedUserId?: number
}

export const useReviewsStore = defineStore('reviews', () => {
  const userReviews = ref<Review[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const createReview = async (taskId: number, reviewInput: CreateReviewInput) => {
    const authStore = useAuthStore()
    const token = authStore.token
    
    loading.value = true
    error.value = null
    
    try {
      const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/reviews`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          rating: reviewInput.rating,
          comment: reviewInput.comment
        })
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create review')
      }
      
      const review = await response.json()
      return review
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'An error occurred'
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchUserReviews = async (userId: number) => {
    const authStore = useAuthStore()
    const token = authStore.token
    
    loading.value = true
    error.value = null
    
    try {
      const response = await fetch(`${config.ENDPOINTS.USERS}/${userId}/reviews`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      
      if (!response.ok) {
        throw new Error('Failed to fetch user reviews')
      }
      
      userReviews.value = await response.json()
      return userReviews.value
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'An error occurred'
      throw err
    } finally {
      loading.value = false
    }
  }

  const hasReviewedTask = async (taskId: number): Promise<boolean> => {
    const authStore = useAuthStore()
    const userId = authStore.user?.id
    
    if (!userId) return false
    
    // Check if user has already reviewed this task
    const reviews = await fetchUserReviews(userId)
    return reviews.some(review => review.taskId === taskId)
  }

  const calculateAverageRating = (reviews: Review[]): number => {
    if (reviews.length === 0) return 0
    const sum = reviews.reduce((acc, review) => acc + review.rating, 0)
    return Math.round((sum / reviews.length) * 10) / 10 // Round to 1 decimal place
  }

  return {
    userReviews,
    loading,
    error,
    createReview,
    fetchUserReviews,
    hasReviewedTask,
    calculateAverageRating
  }
})