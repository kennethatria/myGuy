<template>
  <div class="h-full">
    <!-- Navigation -->
    <nav class="nav">
      <div class="container nav-container">
        <div class="flex justify-between items-center">
          <div class="flex items-center">
            <router-link :to="{ name: 'home' }" class="nav-logo">
              <img class="h-8 w-auto" src="../assets/myguy-icon.svg" alt="MyGuy" />
              <span class="ml-2">MyGuy</span>
            </router-link>
            <div class="nav-links ml-4">
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

          <div class="flex items-center">
            <!-- Profile dropdown -->
            <div class="relative">
              <div>
                <button
                  @click="isProfileMenuOpen = !isProfileMenuOpen"
                  type="button"
                  class="user-avatar"
                  id="user-menu-button"
                  aria-expanded="false"
                  aria-haspopup="true"
                >
                  {{ userInitials }}
                </button>
              </div>

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
                  @click="isProfileMenuOpen = false"
                >
                  {{ item.text }}
                </router-link>
                <a
                  href="#"
                  @click="handleSignOut"
                >
                  Sign out
                </a>
              </div>
            </div>
          </div>

          <!-- Mobile menu button -->
          <div class="flex items-center sm:hidden ml-4">
            <button
              @click="isMobileMenuOpen = !isMobileMenuOpen"
              type="button"
              class="p-2"
              aria-controls="mobile-menu"
              aria-expanded="false"
            >
              <span class="sr-only">Open main menu</span>
              <svg
                class="h-6 w-6"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="var(--color-text)"
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
      <div v-if="isMobileMenuOpen" class="sm:hidden shadow-lg" id="mobile-menu">
        <div class="py-2">
          <router-link
            v-for="item in navigation"
            :key="item.name"
            :to="item.to"
            class="block px-4 py-2"
            :class="{ 'bg-primary text-white': $route.name === item.name }"
            @click="isMobileMenuOpen = false"
          >
            {{ item.text }}
          </router-link>
        </div>
        <div class="py-2 border-t border-gray-200">
          <div class="flex items-center px-4 py-2">
            <div class="user-avatar">
              {{ userInitials }}
            </div>
            <div class="ml-3">
              <div class="font-medium">{{ user?.fullName || 'User' }}</div>
              <div class="text-sm text-gray-500">{{ user?.email || 'user@example.com' }}</div>
            </div>
          </div>
          <div class="mt-2">
            <router-link
              v-for="item in profileMenu"
              :key="item.name"
              :to="item.to"
              class="block px-4 py-2"
              @click="isMobileMenuOpen = false"
            >
              {{ item.text }}
            </router-link>
            <a
              href="#"
              class="block px-4 py-2"
              @click="handleSignOut"
            >
              Sign out
            </a>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main content -->
    <main class="container py-4">
      <router-view></router-view>
    </main>
  </div>
</template>

<style>
/* Component-specific styles */
.user-menu {
  position: absolute;
  right: 0;
  margin-top: 0.5rem;
  width: 12rem;
  background: white;
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  z-index: 10;
}

.user-menu a {
  display: block;
  padding: 0.5rem 1rem;
  color: var(--color-text);
}

.user-menu a:hover {
  background-color: var(--color-background);
}

.user-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border-radius: 9999px;
  background-color: var(--color-primary);
  color: white;
  font-weight: 600;
  cursor: pointer;
}
</style>

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
