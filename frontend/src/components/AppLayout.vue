<template>
  <div class="h-full">
    <!-- Navigation -->
    <nav class="nav">
      <div class="container nav-container">
        <div class="nav-wrapper">
          <div class="nav-start">
            <router-link :to="{ name: 'dashboard' }" class="nav-logo">
              <img class="logo-image" src="../assets/myguy-icon.svg" alt="MyGuy" />
              <span class="logo-text">MyGuy</span>
            </router-link>
            <div class="nav-links">
              <router-link
                v-for="item in navigation"
                :key="item.name"
                :to="item.to"
                class="nav-link"
                :class="{ 'active': $route.name === item.name }"
              >
                {{ item.text }}
                <span v-if="item.name === 'messages' && totalUnreadCount > 0" class="nav-badge">
                  {{ totalUnreadCount }}
                </span>
              </router-link>
            </div>
          </div>

          <div class="nav-end">
            <!-- Logout button -->
            <button
              @click="handleSignOut"
              class="logout-btn"
            >
              Sign out
            </button>
          </div>

          <!-- Mobile menu button -->
          <div class="mobile-menu-button">
            <button
              @click="isMobileMenuOpen = !isMobileMenuOpen"
              type="button"
              aria-controls="mobile-menu"
              aria-expanded="false"
              :class="{ 'open': isMobileMenuOpen }"
              style="background: transparent; border: none; box-shadow: none;"
            >
              <span class="sr-only">Toggle menu</span>
              <!-- Three horizontal lines for hamburger menu -->
              <span class="hamburger-line"></span>
              <span class="hamburger-line"></span>
              <span class="hamburger-line"></span>
            </button>
          </div>
        </div>
      </div>

      <!-- Mobile menu -->
      <transition name="slide-down">
        <div v-if="isMobileMenuOpen" class="mobile-menu" id="mobile-menu">
          <div class="mobile-nav-section">
            <router-link
              v-for="item in navigation"
              :key="item.name"
              :to="item.to"
              class="mobile-nav-link"
              :class="{ 'active': $route.name === item.name }"
              @click="isMobileMenuOpen = false"
            >
              {{ item.text }}
            </router-link>
          </div>
          <div class="mobile-profile-section">
            <div class="mobile-user-info">
              <div class="mobile-user-details">
                <div class="user-name">{{ user?.fullName || 'User' }}</div>
                <div class="user-email">{{ user?.email || 'user@example.com' }}</div>
              </div>
            </div>
            <div class="mobile-menu-items">
              <a
                href="#"
                class="mobile-menu-link"
                @click="handleSignOut"
              >
                Sign out
              </a>
            </div>
          </div>
        </div>
      </transition>
    </nav>

    <!-- Main content -->
    <main class="container py-4">
      <router-view></router-view>
    </main>
  </div>
</template>

<!-- Component-specific styles moved to custom.css -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'


const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()
const isMobileMenuOpen = ref(false)

const user = computed(() => authStore.user)
const totalUnreadCount = computed(() => chatStore.totalUnreadCount)

const navigation = [
  { name: 'dashboard', to: { name: 'dashboard' }, text: 'Dashboard' },
  { name: 'tasks', to: { name: 'tasks' }, text: 'Browse Gigs' },
  { name: 'create-task', to: { name: 'create-task' }, text: 'Post a Gig' },
  { name: 'store', to: { name: 'store' }, text: 'Store' },
  { name: 'messages', to: { name: 'messages' }, text: 'Messages' }
]

const handleSignOut = async () => {
  try {
    isMobileMenuOpen.value = false // Close the mobile menu
    authStore.logout() // Clear the auth state
    await router.push({ name: 'login' }) // Redirect to login page
  } catch (error) {
    console.error('Sign out failed:', error)
  }
}

// Initialize user data
onMounted(async () => {
  // Try to check authentication status on component mount
  if (authStore.token) {
    await authStore.checkAuth()
    // Connect to chat if authenticated
    chatStore.connectSocket()
  }
})
</script>
