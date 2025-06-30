<template>
  <div class="h-full flex flex-col justify-center p-4">
    <div class="container mx-auto" style="max-width: 480px;">
      <div class="text-center mb-4">
        <div class="flex justify-center items-center mb-4">
          <img class="h-12 w-auto" src="../../assets/myguy-icon.svg" alt="MyGuy" />
          <span class="ml-3 text-xl font-bold text-primary">MyGuy</span>
        </div>
        <h1>Sign in to your account</h1>
      </div>

      <div class="card p-4">
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="email" class="form-label">Email address</label>
            <input
              id="email"
              v-model="email"
              name="email"
              type="email"
              required
              class="form-input"
            />
          </div>

          <div class="form-group">
            <label for="password" class="form-label">Password</label>
            <input
              id="password"
              v-model="password"
              name="password"
              type="password"
              required
              class="form-input"
            />
          </div>

          <div v-if="error" class="text-red-500 mb-4">
            {{ error }}
          </div>

          <div class="mb-4">
            <button
              type="submit"
              class="btn btn-primary w-full"
              :disabled="loading"
            >
              {{ loading ? 'Signing in...' : 'Sign in' }}
            </button>
          </div>
        </form>

        <div class="relative py-2">
          <div class="w-full" style="border-top: 1px solid var(--color-border);"></div>
          <div class="flex justify-center" style="margin-top: -12px;">
            <span class="px-2 bg-white text-sm">
              Don't have an account?
              <router-link :to="{ name: 'register' }" class="text-primary font-semibold">
                Register
              </router-link>
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const handleSubmit = async () => {
  if (loading.value) return
  error.value = ''
  loading.value = true

  try {
    await authStore.login(email.value, password.value)
    await router.push({ name: 'dashboard' })
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>
