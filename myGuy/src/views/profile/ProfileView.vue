<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="md:grid md:grid-cols-3 md:gap-6">
      <div class="md:col-span-1">
        <div class="px-4 sm:px-0">
          <h3 class="text-lg font-medium leading-6 text-gray-900">Profile</h3>
          <p class="mt-1 text-sm text-gray-600">
            Your personal information and task history.
          </p>
        </div>
      </div>

      <div class="mt-5 md:mt-0 md:col-span-2">
        <form @submit.prevent="handleSubmit">
          <div class="shadow sm:rounded-md sm:overflow-hidden">
            <div class="px-4 py-5 bg-white space-y-6 sm:p-6">
              <div>
                <label for="username" class="block text-sm font-medium text-gray-700">Username</label>
                <div class="mt-1">
                  <input
                    type="text"
                    name="username"
                    id="username"
                    v-model="profile.username"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
                    required
                  />
                </div>
              </div>

              <div>
                <label for="email" class="block text-sm font-medium text-gray-700">Email</label>
                <div class="mt-1">
                  <input
                    type="email"
                    name="email"
                    id="email"
                    v-model="profile.email"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
                    required
                  />
                </div>
              </div>

              <div>
                <label for="fullName" class="block text-sm font-medium text-gray-700">Full Name</label>
                <div class="mt-1">
                  <input
                    type="text"
                    name="fullName"
                    id="fullName"
                    v-model="profile.fullName"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
                  />
                </div>
              </div>

              <div>
                <label for="bio" class="block text-sm font-medium text-gray-700">Bio</label>
                <div class="mt-1">
                  <textarea
                    id="bio"
                    name="bio"
                    rows="3"
                    v-model="profile.bio"
                    class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md"
                  />
                </div>
                <p class="mt-2 text-sm text-gray-500">
                  Brief description for your profile.
                </p>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700">Rating</label>
                <div class="mt-1 flex items-center">
                  <span class="text-2xl font-semibold text-gray-900">{{ profile.averageRating }}</span>
                  <span class="ml-2 text-sm text-gray-500">/5 from {{ profile.totalReviews }} reviews</span>
                </div>
              </div>
            </div>

            <div class="px-4 py-3 bg-gray-50 text-right sm:px-6">
              <button
                type="submit"
                class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              >
                Save
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>

    <!-- Reviews Section -->
    <div class="mt-8">
      <h2 class="text-lg font-medium text-gray-900 mb-4">Reviews</h2>
      <div class="bg-white shadow overflow-hidden sm:rounded-lg">
        <ul role="list" class="divide-y divide-gray-200">
          <li v-for="review in reviews" :key="review.id" class="p-4">
            <div class="flex space-x-3">
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h3 class="text-sm font-medium">{{ review.reviewer }}</h3>
                  <p class="text-sm text-gray-500">{{ formatDate(review.createdAt) }}</p>
                </div>
                <div class="flex items-center mt-1">
                  <div class="flex-shrink-0">
                    <!-- Simple star rating display -->
                    <span class="text-yellow-400">★</span>
                    <span class="text-sm text-gray-900">{{ review.rating }}/5</span>
                  </div>
                </div>
                <p class="mt-2 text-sm text-gray-700">{{ review.comment }}</p>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { format } from 'date-fns'

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
  reviewer: string
  rating: number
  comment: string
  createdAt: string
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

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM dd, yyyy')
}

onMounted(async () => {
  try {
    // TODO: Implement fetch profile data logic
  } catch (error) {
    console.error('Failed to fetch profile data:', error)
  }
})

const handleSubmit = async () => {
  try {
    // TODO: Implement update profile logic
  } catch (error) {
    console.error('Failed to update profile:', error)
  }
}
</script>
