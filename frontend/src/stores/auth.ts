import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import config from '@/config'
import { useUserStore } from './user'
import { trackEvent } from '@/utils/analytics'

interface User {
  id: number
  username: string
  email: string
  fullName: string
  name?: string
  bio?: string
  averageRating?: number
  createdAt: string
}

// The API sends snake_case (full_name, average_rating); the app reads camelCase.
function toUser(data: User & { full_name?: string; average_rating?: number }): User {
  return {
    ...data,
    fullName: data.fullName ?? data.full_name ?? '',
    averageRating: data.averageRating ?? data.average_rating
  }
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(localStorage.getItem('token'))

  const setAuthHeaders = (token: string) => {
    localStorage.setItem('token', token)
    return {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  }

  const clearAuth = () => {
    user.value = null
    token.value = null
    localStorage.removeItem('token')

    // Clear user store cache on logout
    const userStore = useUserStore()
    userStore.clearCache()
  }

  const cacheCurrentUser = () => {
    if (user.value) {
      const userStore = useUserStore()
      userStore.cacheCurrentUser()
    }
  }

  const startSession = (data: { user: User; token: string }) => {
    user.value = toUser(data.user)
    token.value = data.token

    // Set token in localStorage and update default headers
    setAuthHeaders(data.token)

    // Cache current user in user store
    cacheCurrentUser()
  }

  const postAuth = async <T>(url: string, body: object, fallbackError: string): Promise<T> => {
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      throw new Error(data.error || fallbackError)
    }
    return data as T
  }

  // Passwordless sign-in: email a one-time code to the address.
  const requestCode = async (email: string): Promise<void> => {
    await postAuth(config.ENDPOINTS.AUTH_REQUEST_CODE, { email }, 'Could not send code')
  }

  // Signs in an existing user, or returns a signup token when the verified
  // email has no account yet (finish with completeSignup).
  const verifyCode = async (email: string, code: string): Promise<{ signupToken?: string }> => {
    const data = await postAuth<{ user?: User; token?: string; signup_token?: string }>(
      config.ENDPOINTS.AUTH_VERIFY_CODE,
      { email, code },
      'Could not verify code'
    )
    if (data.signup_token) {
      return { signupToken: data.signup_token }
    }
    startSession(data as { user: User; token: string })
    trackEvent('sign-in')
    return {}
  }

  const completeSignup = async (signupToken: string, fullName: string): Promise<void> => {
    const data = await postAuth<{ user: User; token: string }>(
      config.ENDPOINTS.AUTH_COMPLETE_SIGNUP,
      { signup_token: signupToken, full_name: fullName },
      'Could not create account'
    )
    startSession(data)
    trackEvent('sign-up')
  }

  const logout = () => {
    clearAuth()
  }

  const checkAuth = async (): Promise<boolean> => {
    if (!token.value) return false

    try {
      const response = await fetch(config.ENDPOINTS.PROFILE, {
        headers: {
          'Authorization': `Bearer ${token.value}`,
          'Content-Type': 'application/json'
        }
      })

      if (!response.ok) {
        clearAuth()
        return false
      }

      user.value = toUser(await response.json())

      // Cache current user in user store
      cacheCurrentUser()

      return true
    } catch (error) {
      console.error('Auth check failed:', error)
      clearAuth()
      return false
    }
  }

  // Computed property to check if user is authenticated
  const isAuthenticated = computed(() => user.value !== null && token.value !== null)

  // Initialize auth state
  if (token.value) {
    checkAuth().catch(console.error)
  }

  return {
    user,
    token,
    isAuthenticated,
    requestCode,
    verifyCode,
    completeSignup,
    logout,
    checkAuth,
    setAuthHeaders
  }
})
