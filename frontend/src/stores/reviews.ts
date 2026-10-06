import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'
import { useAuthStore } from './auth'
import { useUserStore } from './user'
import type { Interaction } from '@/utils/network'

export interface Review {
  // Task reviews use their numeric id; store ratings use "store-<booking id>".
  id: number | string
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
  // Store ratings: the item, and whether the user was rated as its seller or buyer
  item?: {
    id: number
    title: string
    ratedAs: 'seller' | 'buyer'
  }
}

interface StoreRating {
  booking_id: number
  item_id: number
  item_title: string
  rater_id: number
  rated_as: 'seller' | 'buyer'
  rating: number
  review: string
  rated_at: string
}

// A gig review as the backend sends it
interface GigReview {
  id: number
  task_id: number
  reviewer_id: number
  reviewed_user_id: number
  rating: number
  comment: string
  created_at: string
  task?: { id: number; title: string }
}

// A store rating either way round (GET /user/ratings)
interface BookingRating extends StoreRating {
  rated_id: number
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

  // Store ratings the user received as a seller or buyer, in review form.
  const fetchStoreRatings = async (userId: number): Promise<Review[]> => {
    const authStore = useAuthStore()
    const response = await fetch(`${config.STORE_API_URL}/users/${userId}/ratings`, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (!response.ok) {
      throw new Error('Failed to fetch store ratings')
    }
    const ratings: StoreRating[] = await response.json()

    const userStore = useUserStore()
    await userStore.fetchUsers([...new Set(ratings.map(r => r.rater_id))])

    return ratings.map(r => {
      const rater = userStore.getUserById(r.rater_id)
      return {
        id: `store-${r.booking_id}`,
        taskId: 0,
        reviewerId: r.rater_id,
        reviewedUserId: userId,
        rating: r.rating,
        comment: r.review,
        created_at: r.rated_at,
        reviewer: rater ? { id: rater.id, username: rater.username } : undefined,
        item: { id: r.item_id, title: r.item_title, ratedAs: r.rated_as }
      }
    })
  }

  // Everything others have said about the user — task reviews and store
  // ratings — newest first, for one combined rating on their profile. If the
  // store service is unavailable, task reviews are still shown.
  const fetchAllRatings = async (userId: number): Promise<Review[]> => {
    const [taskReviews, storeRatings] = await Promise.all([
      fetchUserReviews(userId),
      fetchStoreRatings(userId).catch(err => {
        console.warn('Store ratings unavailable:', err)
        return [] as Review[]
      })
    ])
    return [...taskReviews, ...storeRatings].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )
  }

  // Every review the signed-in user gave or received, from gigs and the
  // marketplace, as interactions with the other person (for their network).
  // If one service is unavailable, the other's reviews are still shown.
  const fetchMyInteractions = async (): Promise<Interaction[]> => {
    const authStore = useAuthStore()
    const me = authStore.user?.id
    if (!me) return []
    const headers = { 'Authorization': `Bearer ${authStore.token}` }
    const get = async <T>(url: string): Promise<T[]> => {
      const response = await fetch(url, { headers })
      if (!response.ok) throw new Error(`Failed to load ${url}`)
      return response.json()
    }

    const [gigReviews, storeRatings] = await Promise.all([
      get<GigReview>(`${config.API_URL}/user/reviews`),
      get<BookingRating>(`${config.STORE_API_URL}/user/ratings`).catch(err => {
        console.warn('Store ratings unavailable:', err)
        return [] as BookingRating[]
      })
    ])

    const fromGigs = gigReviews.map((r): Interaction => {
      const given = r.reviewer_id === me
      return {
        otherId: given ? r.reviewed_user_id : r.reviewer_id,
        direction: given ? 'given' : 'received',
        rating: r.rating,
        comment: r.comment,
        via: 'gig',
        title: r.task?.title || 'A gig',
        linkId: r.task_id,
        deal: `gig-${r.task_id}`,
        at: r.created_at
      }
    })
    const fromStore = storeRatings.map((r): Interaction => {
      const given = r.rater_id === me
      return {
        otherId: given ? r.rated_id : r.rater_id,
        direction: given ? 'given' : 'received',
        rating: r.rating,
        comment: r.review,
        via: 'item',
        title: r.item_title || 'A marketplace item',
        linkId: r.item_id,
        deal: `item-${r.booking_id}`,
        at: r.rated_at
      }
    })

    const interactions = [...fromGigs, ...fromStore].filter(i => i.otherId && i.otherId !== me)
    await useUserStore().fetchUsers(interactions.map(i => i.otherId))
    return interactions
  }

  const hasReviewedTask = async (taskId: number): Promise<boolean> => {
    const authStore = useAuthStore()
    const userId = authStore.user?.id
    
    if (!userId) return false

    // Ask the backend whether this user wrote a review for the task
    const response = await fetch(`${config.ENDPOINTS.TASKS}/${taskId}/reviews/mine`, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (!response.ok) {
      throw new Error('Failed to check review status')
    }
    const data: { reviewed: boolean } = await response.json()
    return data.reviewed
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
    fetchStoreRatings,
    fetchAllRatings,
    fetchMyInteractions,
    hasReviewedTask,
    calculateAverageRating
  }
})