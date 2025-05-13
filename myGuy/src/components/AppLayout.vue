<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Navigation -->
    <nav class="bg-white shadow">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex">
            <div class="flex-shrink-0 flex items-center">
              <router-link :to="{ name: 'home' }">
                <img class="h-8 w-auto" src="/src/assets/logo.svg" alt="MyGuy" />
              </router-link>
            </div>
            <div class="hidden sm:ml-6 sm:flex sm:space-x-8">
              <router-link
                v-for="item in navigation"
                :key="item.name"
                :to="item.to"
                :class="[
                  $route.name === item.name
                    ? 'border-indigo-500 text-gray-900'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
                  'inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium'
                ]"
              >
                {{ item.text }}
              </router-link>
            </div>
          </div>
          <div class="hidden sm:ml-6 sm:flex sm:items-center">
            <!-- Profile dropdown -->
            <div class="ml-3 relative">
              <div>
                <button
                  @click="isProfileMenuOpen = !isProfileMenuOpen"
                  type="button"
                  class="bg-white rounded-full flex text-sm focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                  id="user-menu-button"
                  aria-expanded="false"
                  aria-haspopup="true"
                >
                  <span class="sr-only">Open user menu</span>
                  <span class="inline-flex items-center justify-center h-8 w-8 rounded-full bg-gray-500">
                    <span class="text-sm font-medium leading-none text-white">{{ userInitials }}</span>
                  </span>
                </button>
              </div>

              <div
                v-if="isProfileMenuOpen"
                class="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg py-1 bg-white ring-1 ring-black ring-opacity-5 focus:outline-none"
                role="menu"
                aria-orientation="vertical"
                aria-labelledby="user-menu-button"
                tabindex="-1"
              >
                <router-link
                  v-for="item in profileMenu"
                  :key="item.name"
                  :to="item.to"
                  class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  role="menuitem"
                  tabindex="-1"
                  @click="isProfileMenuOpen = false"
                >
                  {{ item.text }}
                </router-link>
                <a
                  href="#"
                  class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  role="menuitem"
                  tabindex="-1"
                  @click="handleSignOut"
                >
                  Sign out
                </a>
              </div>
            </div>
          </div>

          <!-- Mobile menu button -->
          <div class="-mr-2 flex items-center sm:hidden">
            <button
              @click="isMobileMenuOpen = !isMobileMenuOpen"
              type="button"
              class="inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-gray-500 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-indigo-500"
              aria-controls="mobile-menu"
              aria-expanded="false"
            >
              <span class="sr-only">Open main menu</span>
              <svg
                class="h-6 w-6"
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
      <div v-if="isMobileMenuOpen" class="sm:hidden" id="mobile-menu">
        <div class="pt-2 pb-3 space-y-1">
          <router-link
            v-for="item in navigation"
            :key="item.name"
            :to="item.to"
            :class="[
              $route.name === item.name
                ? 'bg-indigo-50 border-indigo-500 text-indigo-700'
                : 'border-transparent text-gray-600 hover:bg-gray-50 hover:border-gray-300 hover:text-gray-800',
              'block pl-3 pr-4 py-2 border-l-4 text-base font-medium'
            ]"
            @click="isMobileMenuOpen = false"
          >
            {{ item.text }}
          </router-link>
        </div>
        <div class="pt-4 pb-3 border-t border-gray-200">
          <div class="flex items-center px-4">
            <div class="flex-shrink-0">
              <span class="inline-flex items-center justify-center h-10 w-10 rounded-full bg-gray-500">
                <span class="text-sm font-medium leading-none text-white">{{ userInitials }}</span>
              </span>
            </div>
            <div class="ml-3">
              <div class="text-base font-medium text-gray-800">{{ user?.fullName }}</div>
              <div class="text-sm font-medium text-gray-500">{{ user?.email }}</div>
            </div>
          </div>
          <div class="mt-3 space-y-1">
            <router-link
              v-for="item in profileMenu"
              :key="item.name"
              :to="item.to"
              class="block px-4 py-2 text-base font-medium text-gray-500 hover:text-gray-800 hover:bg-gray-100"
              @click="isMobileMenuOpen = false"
            >
              {{ item.text }}
            </router-link>
            <a
              href="#"
              class="block px-4 py-2 text-base font-medium text-gray-500 hover:text-gray-800 hover:bg-gray-100"
              @click="handleSignOut"
            >
              Sign out
            </a>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main content -->
    <main>
      <router-view></router-view>
    </main>
  </div>
</template>

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
  { name: 'tasks', to: { name: 'tasks' }, text: 'Browse Tasks' },
  { name: 'create-task', to: { name: 'create-task' }, text: 'Create Task' }
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
