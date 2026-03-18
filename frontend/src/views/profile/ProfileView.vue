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
                disabled
                title="Username cannot be changed"
              />
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
              <div class="alert alert-danger">{{ formError }}</div>
            </div>
            
            <div class="form-group" v-if="successMessage">
              <div class="alert alert-success">{{ successMessage }}</div>
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
      <ReviewList 
        :reviews="reviews" 
        :loading="isLoadingReviews"
        :error="reviewsError"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import config from '@/config'
import { useAuthStore } from '@/stores/auth'
import { useReviewsStore } from '@/stores/reviews'
import ReviewList from '@/components/ReviewList.vue'

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
const successMessage = ref('')
const isLoading = ref(true)
const isLoadingReviews = ref(false)
const reviewsError = ref<string | null>(null)
const formErrors = ref({
  username: '',
  email: '',
  fullName: '',
  bio: ''
})

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
  const reviewsStore = useReviewsStore()
  
  isLoadingReviews.value = true
  reviewsError.value = null
  
  try {
    const userReviews = await reviewsStore.fetchUserReviews(userId)
    reviews.value = userReviews
    
    // Update total reviews count in profile
    profile.value.totalReviews = reviews.value.length
    
    // Calculate average rating using the store's helper
    profile.value.averageRating = reviewsStore.calculateAverageRating(reviews.value)
  } catch (error) {
    console.error('Error fetching user reviews:', error)
    reviewsError.value = error instanceof Error ? error.message : 'Failed to load reviews'
    reviews.value = []
  } finally {
    isLoadingReviews.value = false
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
  successMessage.value = ''
  
  // Validate email
  if (!profile.value.email.trim()) {
    formErrors.value.email = 'Email is required'
    isValid = false
  } else if (!/^\S+@\S+\.\S+$/.test(profile.value.email)) {
    formErrors.value.email = 'Please enter a valid email address'
    isValid = false
  }
  
  // Validate fullName
  if (!profile.value.fullName.trim()) {
    formErrors.value.fullName = 'Full Name is required'
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
    successMessage.value = ''
    
    const authStore = useAuthStore()
    if (!authStore.user) {
      throw new Error('User not authenticated')
    }
    
    const response = await fetch(config.ENDPOINTS.PROFILE, {
      method: 'PUT',  // Changed from PATCH to PUT to match backend route
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        full_name: profile.value.fullName,
        email: profile.value.email,
        bio: profile.value.bio,
        phone_number: '' // Including empty phone_number to match backend struct
      })
    })
    
    if (!response.ok) {
      // Safely try to parse error response as JSON
      try {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update profile')
      } catch {
        // If JSON parsing fails, use status text
        throw new Error(`Failed to update profile: ${response.statusText}`)
      }
    }
    
    // Get updated user data from response
    const updatedUser = await response.json()
    
    // Update the profile with the returned data
    if (updatedUser) {
      // Update local profile data
      profile.value = {
        ...profile.value,
        username: updatedUser.username,
        email: updatedUser.email,
        fullName: updatedUser.fullName || updatedUser.full_name,
        bio: updatedUser.bio || ''
      }
      
      // Also update the auth store to ensure consistency
      await authStore.checkAuth()
    }
    
    // Show success message below the form
    formError.value = ''
    successMessage.value = 'Profile updated successfully!'
    
    // Auto-hide success message after 5 seconds
    setTimeout(() => {
      successMessage.value = ''
    }, 5000)
  } catch (error) {
    console.error('Failed to update profile:', error)
    // Make sure we handle error properly whether it's an Error object or something else
    formError.value = error instanceof Error 
      ? error.message 
      : 'Failed to update profile. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.alert {
  position: relative;
  padding: 0.75rem 1.25rem;
  margin-bottom: 1rem;
  border: 1px solid transparent;
  border-radius: 0.25rem;
}

.alert-danger {
  color: #721c24;
  background-color: #f8d7da;
  border-color: #f5c6cb;
}

.alert-success {
  color: #155724;
  background-color: #d4edda;
  border-color: #c3e6cb;
}

.invalid-feedback {
  display: block;
  width: 100%;
  margin-top: 0.25rem;
  color: #dc3545;
}
</style>
