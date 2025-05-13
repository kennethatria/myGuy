<template>
  <div class="h-full">
    <!-- Navigation -->
    <nav class="nav">
      <div class="container nav-container">
        <div class="nav-wrapper">
          <div class="nav-start">
            <router-link :to="{ name: 'home' }" class="nav-logo">
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
              </router-link>
            </div>
          </div>

          <div class="user-profile">
            <!-- Profile dropdown -->
            <div class="profile-dropdown">
              <button
                @click="isProfileMenuOpen = !isProfileMenuOpen"
                type="button"
                class="user-avatar"
                id="user-menu-button"
                aria-expanded="false"
                aria-haspopup="true"
              >
                <span v-if="user?.fullName">{{ userInitials }}</span>
                <span v-else class="user-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" class="w-4 h-4">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                </span>
              </button>

              <transition name="fade">
                <div
                  v-if="isProfileMenuOpen"
                  class="user-menu"
                  role="menu"
                  aria-orientation="vertical"
                  aria-labelledby="user-menu-button"
                >
                  <router-link
                    v-for="item in profileMenu"
                    :key="item.name"
                    :to="item.to"
                    class="menu-item"
                    @click="isProfileMenuOpen = false"
                  >
                    {{ item.text }}
                  </router-link>
                  <a
                    href="#"
                    class="menu-item"
                    @click="handleSignOut"
                  >
                    Sign out
                  </a>
                </div>
              </transition>
            </div>
          </div>

          <!-- Mobile menu button -->
          <div class="mobile-menu-button">
            <button
              @click="isMobileMenuOpen = !isMobileMenuOpen"
              type="button"
              aria-controls="mobile-menu"
              aria-expanded="false"
            >
              <span class="sr-only">Toggle menu</span>
              <svg
                class="menu-icon"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path v-if="!isMobileMenuOpen" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
                <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
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
              <div class="mobile-avatar">
                {{ userInitials }}
              </div>
              <div class="mobile-user-details">
                <div class="user-name">{{ user?.fullName || 'User' }}</div>
                <div class="user-email">{{ user?.email || 'user@example.com' }}</div>
              </div>
            </div>
            <div class="mobile-menu-items">
              <router-link
                v-for="item in profileMenu"
                :key="item.name"
                :to="item.to"
                class="mobile-menu-link"
                @click="isMobileMenuOpen = false"
              >
                {{ item.text }}
              </router-link>
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
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

interface User {
  username: string
  email: string
  fullName: string
}

const router = useRouter()
const isProfileMenuOpen = ref(false)
const isMobileMenuOpen = ref(false)

const user = ref<User | null>(null)

const navigation = [
  { name: 'dashboard', to: { name: 'dashboard' }, text: 'Dashboard' },
  { name: 'tasks', to: { name: 'tasks' }, text: 'Browse Gigs' },
  { name: 'create-task', to: { name: 'create-task' }, text: 'Post a Gig' }
]

const profileMenu = [
  { name: 'profile', to: { name: 'profile' }, text: 'Your Profile' }
]

const userInitials = computed(() => {
  if (!user.value?.fullName) return '?'
  return user.value.fullName
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
})

const handleSignOut = async () => {
  try {
    // TODO: Implement sign out logic
    await router.push({ name: 'login' })
  } catch (error) {
    console.error('Sign out failed:', error)
  }
}
</script>
