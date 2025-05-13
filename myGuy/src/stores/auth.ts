import { defineStore } from 'pinia'
import { ref } from 'vue'
import config from '@/config'

interface User {
  id: number
  username: string
  email: string
  fullName: string
  bio?: string
  averageRating?: number
  createdAt: string
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
  }

  const login = async (email: string, password: string): Promise<boolean> => {
    try {
      const response = await fetch(config.ENDPOINTS.LOGIN, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Login failed')
      }

      const data = await response.json()
      user.value = data.user
      token.value = data.token
      
      // Set token in localStorage and update default headers
      setAuthHeaders(data.token)
      return true
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    }
  }

  const register = async (username: string, email: string, password: string, fullName: string): Promise<boolean> => {
    try {
      const response = await fetch(config.ENDPOINTS.REGISTER, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, email, password, full_name: fullName })
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Registration failed')
      }

      return true
    } catch (error) {
      console.error('Registration failed:', error)
      throw error
    }
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

      user.value = await response.json()
      return true
    } catch (error) {
      console.error('Auth check failed:', error)
      clearAuth()
      return false
    }
  }

  // Initialize auth state
  if (token.value) {
    checkAuth().catch(console.error)
  }

  return {
    user,
    token,
    login,
    register,
    logout,
    checkAuth,
    setAuthHeaders
  }
})
