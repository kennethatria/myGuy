import { defineStore } from 'pinia'
import { ref } from 'vue'

interface Profile {
  id: number
  username: string
  email: string
  fullName: string
  bio: string
  averageRating: number
  totalReviews: number
}

interface Review {
  id: number
  taskId: number
  reviewerId: number
  reviewedUserId: number
  rating: number
  comment: string
  created_at: string
}

export const useProfileStore = defineStore('profile', () => {
  const profile = ref<Profile | null>(null)
  const reviews = ref<Review[]>([])

  const fetchProfile = async (userId: number) => {
    try {
      const response = await fetch(`/api/users/${userId}`)
      if (!response.ok) throw new Error('Failed to fetch profile')
      profile.value = await response.json()
    } catch (error) {
      console.error('Error fetching profile:', error)
      throw error
    }
  }

  const updateProfile = async (userId: number, data: Partial<Profile>) => {
    try {
      const response = await fetch(`/api/users/${userId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      })
      if (!response.ok) throw new Error('Failed to update profile')
      profile.value = await response.json()
    } catch (error) {
      console.error('Error updating profile:', error)
      throw error
    }
  }

  const fetchReviews = async (userId: number) => {
    try {
      const response = await fetch(`/api/users/${userId}/reviews`)
      if (!response.ok) throw new Error('Failed to fetch reviews')
      reviews.value = await response.json()
    } catch (error) {
      console.error('Error fetching reviews:', error)
      throw error
    }
  }

  const createReview = async (taskId: number, reviewedUserId: number, data: Pick<Review, 'rating' | 'comment'>) => {
    try {
      const response = await fetch(`/api/tasks/${taskId}/reviews`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ ...data, reviewedUserId }),
      })
      if (!response.ok) throw new Error('Failed to create review')
      const newReview = await response.json()
      reviews.value.push(newReview)
      return newReview
    } catch (error) {
      console.error('Error creating review:', error)
      throw error
    }
  }

  return {
    profile,
    reviews,
    fetchProfile,
    updateProfile,
    fetchReviews,
    createReview,
  }
})
