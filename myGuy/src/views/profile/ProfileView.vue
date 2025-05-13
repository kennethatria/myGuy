<template>
  <div class="container py-4">
    <h1 class="mb-4">My Profile</h1>
    
    <div class="row">
      <!-- Profile information -->
      <div class="col">
        <div class="card mb-4">
          <h3>Profile Information</h3>
          <p class="text-gray mt-2">
            Manage your personal information and review your gig history.
          </p>
          
          <div class="rating-summary mt-4">
            <h4>Your Rating</h4>
            <div class="flex items-center mt-2">
              <div class="rating-display">
                <span class="rating-value">{{ profile.averageRating.toFixed(1) }}</span>
                <span class="rating-star">★</span>
              </div>
              <span class="text-sm text-gray ml-2">from {{ profile.totalReviews }} reviews</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Profile form -->
      <div class="col">
        <div class="card">
          <form @submit.prevent="handleSubmit">
            <div class="form-group">
              <label for="username" class="form-label">Username</label>
              <input
                type="text"
                name="username"
                id="username"
                v-model="profile.username"
                class="form-input"
                :class="{ 'is-invalid': formErrors.username }"
                required
              />
              <div v-if="formErrors.username" class="invalid-feedback">{{ formErrors.username }}</div>
            </div>

            <div class="form-group">
              <label for="email" class="form-label">Email</label>
              <input
                type="email"
                name="email"
                id="email"
                v-model="profile.email"
                class="form-input"
                :class="{ 'is-invalid': formErrors.email }"
                required
              />
              <div v-if="formErrors.email" class="invalid-feedback">{{ formErrors.email }}</div>
            </div>

            <div class="form-group">
              <label for="fullName" class="form-label">Full Name</label>
              <input
                type="text"
                name="fullName"
                id="fullName"
                v-model="profile.fullName"
                class="form-input"
                :class="{ 'is-invalid': formErrors.fullName }"
              />
              <div v-if="formErrors.fullName" class="invalid-feedback">{{ formErrors.fullName }}</div>
            </div>

            <div class="form-group">
              <label for="bio" class="form-label">Bio</label>
              <textarea
                id="bio"
                name="bio"
                rows="4"
                v-model="profile.bio"
                class="form-input"
                :class="{ 'is-invalid': formErrors.bio }"
                placeholder="Tell others a bit about yourself..."
              ></textarea>
              <p class="form-helper">Share your skills, experience, and interests with the community.</p>
              <div v-if="formErrors.bio" class="invalid-feedback">{{ formErrors.bio }}</div>
            </div>

            <div class="form-group" v-if="formError">
              <div class="invalid-feedback">{{ formError }}</div>
            </div>

            <div class="flex justify-end mt-4">
              <button
                type="submit"
                class="btn btn-primary"
                :disabled="isSubmitting"
              >
                <span v-if="isSubmitting">Saving...</span>
                <span v-else>Save Profile</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Reviews Section -->
    <div class="mt-4">
      <h2 class="mb-3">Reviews</h2>
      <div class="card">
        <div v-if="isLoading" class="p-4 text-center">
          <p class="text-gray">Loading reviews...</p>
        </div>
        <div v-else-if="reviews.length === 0" class="p-4 text-center">
          <p class="text-gray">No reviews exist at this time.</p>
        </div>
        <ul v-else class="divide-y">
          <li v-for="review in reviews" :key="review.id" class="p-4">
            <div class="flex justify-between items-center mb-2">
              <h4 class="font-semibold">{{ review.reviewer?.fullName || review.reviewer?.username }}</h4>
              <div class="flex items-center">
                <span class="rating-star mr-1">★</span>
                <span class="text-sm font-semibold">{{ review.rating }}/5</span>
              </div>
            </div>
            <p class="text-sm">{{ review.comment }}</p>
            <p class="text-sm text-gray mt-2 text-right">{{ formatDate(review.created_at) }}</p>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { format } from 'date-fns'
import config from '@/config'
import { useAuthStore } from '@/stores/auth'

interface Profile {
  username: string
  email: string
  fullName: string
  bio: string
  averageRating: number
  totalReviews: number
}

interface Review {
  id: number
  reviewer: {
    id: number
    username: string
    fullName?: string
  }
  rating: number
  comment: string
  created_at: string
}

const profile = ref<Profile>({
  username: '',
  email: '',
  fullName: '',
  bio: '',
  averageRating: 0,
  totalReviews: 0
})

const reviews = ref<Review[]>([])
const isSubmitting = ref(false)
const formError = ref('')
const isLoading = ref(true)
const formErrors = ref({
  username: '',
  email: '',
  fullName: '',
  bio: ''
})

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

const fetchProfileData = async () => {
  const authStore = useAuthStore()
  if (!authStore.user) {
    await authStore.checkAuth()
  }
  
  if (authStore.user) {
    // Set profile data from auth user
    profile.value = {
      username: authStore.user.username,
      email: authStore.user.email,
      fullName: authStore.user.fullName,
      bio: authStore.user.bio || '',
      averageRating: authStore.user.averageRating || 0,
      totalReviews: 0  // Will be updated from reviews count
    }
    
    // Fetch user reviews
    await fetchUserReviews(authStore.user.id)
  }
}

const fetchUserReviews = async (userId: number) => {
  const authStore = useAuthStore()
  const token = authStore.token
  
  try {
    const response = await fetch(`${config.API_URL}/users/${userId}/reviews`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw new Error('Failed to fetch reviews')
    }
    
    const reviewsData = await response.json()
    reviews.value = reviewsData
    
    // Update total reviews count in profile
    if (reviews.value && Array.isArray(reviews.value)) {
      profile.value.totalReviews = reviews.value.length
      
      // Calculate average rating if there are reviews
      if (reviews.value.length > 0) {
        const totalRating = reviews.value.reduce((sum, review) => sum + review.rating, 0)
        profile.value.averageRating = totalRating / reviews.value.length
      }
    }
  } catch (error) {
    console.error('Error fetching user reviews:', error)
  }
}

onMounted(async () => {
  isLoading.value = true
  try {
    await fetchProfileData()
  } catch (error) {
    console.error('Failed to fetch profile data:', error)
    formError.value = 'Failed to load profile data. Please try refreshing the page.'
  } finally {
    isLoading.value = false
  }
})

const validateForm = (): boolean => {
  let isValid = true
  formErrors.value = {
    username: '',
    email: '',
    fullName: '',
    bio: ''
  }
  formError.value = ''
  
  // Validate username
  if (!profile.value.username.trim()) {
    formErrors.value.username = 'Username is required'
    isValid = false
  } else if (profile.value.username.length < 3) {
    formErrors.value.username = 'Username must be at least 3 characters'
    isValid = false
  }
  
  // Validate email
  if (!profile.value.email.trim()) {
    formErrors.value.email = 'Email is required'
    isValid = false
  } else if (!/^\S+@\S+\.\S+$/.test(profile.value.email)) {
    formErrors.value.email = 'Please enter a valid email address'
    isValid = false
  }
  
  // Bio validation (optional)
  if (profile.value.bio && profile.value.bio.length > 500) {
    formErrors.value.bio = 'Bio must be less than 500 characters'
    isValid = false
  }
  
  return isValid
}

const handleSubmit = async () => {
  if (!validateForm()) {
    return
  }
  
  try {
    isSubmitting.value = true
    formError.value = ''
    
    const authStore = useAuthStore()
    if (!authStore.user) {
      throw new Error('User not authenticated')
    }
    
    const response = await fetch(config.ENDPOINTS.PROFILE, {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        full_name: profile.value.fullName,
        email: profile.value.email,
        bio: profile.value.bio
      })
    })
    
    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(errorData.error || 'Failed to update profile')
    }
    
    // Update the user data in auth store
    await authStore.checkAuth()
    
    // Show success message
    alert('Profile updated successfully!')
  } catch (error) {
    console.error('Failed to update profile:', error)
    formError.value = error.message || 'Failed to update profile. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>
