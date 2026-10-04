<template>
  <div class="container py-4">
    <h1 class="mb-4">My Profile</h1>

    <div class="card">
      <!-- Account details can't be edited: the email is how you sign in -->
      <dl class="account-details">
        <div>
          <dt>Username</dt>
          <dd>{{ profile.username }}</dd>
        </div>
        <div>
          <dt>Email</dt>
          <dd>{{ profile.email }}</dd>
          <dd class="form-helper">You sign in with this email.</dd>
        </div>
      </dl>

      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="fullName" class="form-label">Full Name</label>
          <input
            type="text"
            name="fullName"
            id="fullName"
            v-model="profile.fullName"
            class="form-input"
            :class="{ 'is-invalid': formErrors.fullName }"
            autocomplete="name"
            maxlength="100"
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
            maxlength="500"
            placeholder="Your skills, experience and interests"
          ></textarea>
          <p class="form-helper">{{ profile.bio.length }}/500</p>
          <div v-if="formErrors.bio" class="invalid-feedback">{{ formErrors.bio }}</div>
        </div>

        <div v-if="formError" class="alert alert-danger">{{ formError }}</div>
        <div v-if="successMessage" class="alert alert-success">{{ successMessage }}</div>

        <div class="flex justify-end mt-4">
          <button type="submit" class="btn btn-primary" :disabled="isSubmitting">
            {{ isSubmitting ? 'Saving...' : 'Save Profile' }}
          </button>
        </div>
      </form>
    </div>

    <!-- Your rating and reviews (the list shows the average) -->
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
import { useReviewsStore, type Review } from '@/stores/reviews'
import ReviewList from '@/components/ReviewList.vue'

const authStore = useAuthStore()
const reviewsStore = useReviewsStore()

const profile = ref({
  username: '',
  email: '',
  fullName: '',
  bio: ''
})

const reviews = ref<Review[]>([])
const isSubmitting = ref(false)
const formError = ref('')
const successMessage = ref('')
const isLoadingReviews = ref(false)
const reviewsError = ref<string | null>(null)
const formErrors = ref({ fullName: '', bio: '' })

const loadProfile = async () => {
  if (!authStore.user) {
    await authStore.checkAuth()
  }
  const user = authStore.user
  if (!user) return

  profile.value = {
    username: user.username,
    email: user.email,
    fullName: user.fullName || '',
    bio: user.bio || ''
  }

  isLoadingReviews.value = true
  reviewsError.value = null
  try {
    // Task reviews and store ratings together: one rating for the user
    reviews.value = await reviewsStore.fetchAllRatings(user.id)
  } catch (error) {
    console.error('Error fetching user reviews:', error)
    reviewsError.value = error instanceof Error ? error.message : 'Failed to load reviews'
    reviews.value = []
  } finally {
    isLoadingReviews.value = false
  }
}

onMounted(async () => {
  try {
    await loadProfile()
  } catch (error) {
    console.error('Failed to fetch profile data:', error)
    formError.value = 'Failed to load profile data. Please try refreshing the page.'
  }
})

const validateForm = (): boolean => {
  formErrors.value = { fullName: '', bio: '' }
  formError.value = ''
  successMessage.value = ''

  if (!profile.value.fullName.trim()) {
    formErrors.value.fullName = 'Full Name is required'
  }
  if (profile.value.bio.length > 500) {
    formErrors.value.bio = 'Bio must be 500 characters or fewer'
  }
  return !formErrors.value.fullName && !formErrors.value.bio
}

const handleSubmit = async () => {
  if (!validateForm()) return

  isSubmitting.value = true
  try {
    const response = await fetch(config.ENDPOINTS.PROFILE, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        full_name: profile.value.fullName,
        bio: profile.value.bio
      })
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `Failed to update profile: ${response.statusText}`)
    }

    // Refresh the signed-in user so the rest of the app shows the new name
    await authStore.checkAuth()

    successMessage.value = 'Profile updated successfully!'
    setTimeout(() => {
      successMessage.value = ''
    }, 5000)
  } catch (error) {
    console.error('Failed to update profile:', error)
    formError.value = error instanceof Error ? error.message : 'Failed to update profile. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.account-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin: 0 0 1.5rem;
}

.account-details dt {
  font-size: 0.875rem;
  color: #6b7280;
}

.account-details dd {
  margin: 0.25rem 0 0;
  font-weight: 500;
  overflow-wrap: anywhere;
}

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
