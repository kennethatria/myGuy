<template>
  <div class="app-layout">
    <!-- Sidebar -->
    <!-- Phones: the sidebar is a drawer over the page; tapping outside closes it -->
    <div v-if="isMobileMenuOpen" class="sidebar-backdrop" @click="isMobileMenuOpen = false"></div>

    <aside class="sidebar" :class="{ 'collapsed': isSidebarCollapsed, 'mobile-open': isMobileMenuOpen }">
      <div class="sidebar-header">
        <router-link :to="{ name: 'dashboard' }" class="logo-link">
          <img class="logo-icon" src="../assets/myguy-icon.svg" alt="MyGuy" />
          <span v-if="!isSidebarCollapsed" class="logo-text">MyGuy</span>
        </router-link>
      </div>
      
      <nav class="sidebar-nav">
        <ul class="nav-list">
          <li v-for="item in mainNavigation" :key="item.key">
            <button
              v-if="item.action === 'sign-out'"
              type="button"
              class="nav-item nav-button"
              :title="item.text"
              @click="handleSignOut"
            >
              <span class="nav-icon" v-html="item.icon"></span>
              <span v-if="!isSidebarCollapsed" class="nav-text">{{ item.text }}</span>
            </button>
            <router-link
              v-else
              :to="item.to!"
              class="nav-item"
              :class="{ 'active': isActiveRoute(item) }"
              :title="item.text"
            >
              <span class="nav-icon" v-html="item.icon"></span>
              <span v-if="!isSidebarCollapsed" class="nav-text">{{ item.text }}</span>
              <span v-if="item.badge && !isSidebarCollapsed" class="nav-badge">{{ item.badge }}</span>
            </router-link>
          </li>
        </ul>
      </nav>

      <div class="sidebar-footer">
        <router-link :to="{ name: 'profile' }" class="user-section" :title="user?.fullName || 'Profile'">
          <div class="user-avatar">
            <span>{{ userInitials }}</span>
          </div>
          <div v-if="!isSidebarCollapsed" class="user-info">
            <div class="user-name">{{ user?.fullName || 'User' }}</div>
            <div class="user-email">{{ user?.email || '' }}</div>
          </div>
        </router-link>
      </div>
    </aside>
    
    <!-- Main Content -->
    <div class="main-wrapper">
      <!-- Top Bar -->
      <header class="top-bar">
        <button class="sidebar-toggle" aria-label="Toggle navigation" @click="toggleSidebar">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path d="M3 12H21M3 6H21M3 18H21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </button>
        
        
      </header>
      
      <!-- Page Content -->
      <main class="main-content" :class="{ 'has-chat-widget': route.name !== 'messages' }">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter, useRoute, type RouteLocationRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const chatStore = useChatStore()

const isSidebarCollapsed = ref(false)
const isMobileMenuOpen = ref(false)

// Matches the stylesheet's mobile breakpoint
const mobileQuery = window.matchMedia('(max-width: 768px)')
const isMobile = ref(mobileQuery.matches)
const onViewportChange = (e: MediaQueryListEvent) => {
  isMobile.value = e.matches
  if (!e.matches) isMobileMenuOpen.value = false
}

const user = computed(() => authStore.user)
const totalUnreadCount = computed(() => {
  try {
    return chatStore.totalUnreadCount || 0
  } catch (error) {
    console.warn('Chat store unavailable:', error)
    return 0
  }
})

interface NavItem {
  key: string
  text: string
  icon: string
  to?: RouteLocationRaw
  // Highlighted when on this route (and, for My Gigs, this list)
  route?: string
  tab?: string
  badge?: number
  action?: 'sign-out'
}

// One list: the pages, then your own gigs, then your account
const mainNavigation = computed<NavItem[]>(() => [
  { key: 'home', route: 'dashboard', to: { name: 'dashboard' }, text: 'Home', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9L12 2L21 9V20C21 20.5304 20.7893 21.0391 20.4142 21.4142C20.0391 21.7893 19.5304 22 19 22H5C4.46957 22 3.96086 21.7893 3.58579 21.4142C3.21071 21.0391 3 20.5304 3 20V9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
  { key: 'tasks', route: 'tasks', to: { name: 'tasks' }, text: 'Browse Gigs', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M21 21L15 15M17 10C17 13.866 13.866 17 10 17C6.13401 17 3 13.866 3 10C3 6.13401 6.13401 3 10 3C13.866 3 17 6.13401 17 10Z" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>' },
  { key: 'store', route: 'store', to: { name: 'store' }, text: 'Marketplace', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9V21H21V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 9H21L19 3H5L3 9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M12 3V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
  {
    key: 'messages', route: 'messages', to: { name: 'messages' }, text: 'Messages',
    badge: totalUnreadCount.value > 0 ? totalUnreadCount.value : undefined,
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M21 15C21 15.5304 20.7893 16.0391 20.4142 16.4142C20.0391 16.7893 19.5304 17 19 17H7L3 21V5C3 4.46957 3.21071 3.96086 3.58579 3.58579C3.96086 3.21071 4.46957 3 5 3H19C19.5304 3 20.0391 3.21071 20.4142 3.58579C20.7893 3.96086 21 4.46957 21 5V15Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>'
  },
  { key: 'created', route: 'my-gigs', tab: 'created', to: { name: 'my-gigs', params: { tab: 'created' } }, text: 'My Created Gigs', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M14 2H6C4.9 2 4 2.9 4 4V20C4 21.1 4.9 22 6 22H18C19.1 22 20 21.1 20 20V8L14 2Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/><path d="M14 2V8H20M8 13H16M8 17H16" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
  { key: 'assigned', route: 'my-gigs', tab: 'assigned', to: { name: 'my-gigs', params: { tab: 'assigned' } }, text: 'Gigs Assigned to Me', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M16 21V19C16 16.8 14.2 15 12 15H5C2.8 15 1 16.8 1 19V21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><circle cx="8.5" cy="7" r="4" stroke="currentColor" stroke-width="2"/><path d="M17 11L19 13L23 9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
  { key: 'applications', route: 'my-gigs', tab: 'applications', to: { name: 'my-gigs', params: { tab: 'applications' } }, text: 'My Applications', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M22 2L11 13M22 2L15 22L11 13L2 9L22 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
  { key: 'profile', route: 'profile', to: { name: 'profile' }, text: 'Profile', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M20 21V19C20 16.8 18.2 15 16 15H8C5.8 15 4 16.8 4 19V21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><circle cx="12" cy="7" r="4" stroke="currentColor" stroke-width="2"/></svg>' },
  { key: 'sign-out', text: 'Sign out', action: 'sign-out', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M9 21H5C3.9 21 3 20.1 3 19V5C3 3.9 3.9 3 5 3H9M16 17L21 12L16 7M21 12H9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' }
])

const userInitials = computed(() => {
  if (!user.value?.fullName) return '?'
  return user.value.fullName
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
})

// Desktop: collapse to icons. Phones: open or close the drawer (never
// collapsed there, so labels and the user menu stay visible).
const toggleSidebar = () => {
  if (isMobile.value) {
    isSidebarCollapsed.value = false
    isMobileMenuOpen.value = !isMobileMenuOpen.value
  } else {
    isSidebarCollapsed.value = !isSidebarCollapsed.value
  }
}

// Close the drawer once the user has picked a page
watch(() => route.fullPath, () => {
  isMobileMenuOpen.value = false
})

const handleSignOut = async () => {
  try {
    authStore.logout()
    await router.push({ name: 'login' })
  } catch (error) {
    console.error('Sign out failed:', error)
  }
}

const isActiveRoute = (item: NavItem) =>
  route.name === item.route && (!item.tab || route.params.tab === item.tab)

onBeforeUnmount(() => mobileQuery.removeEventListener('change', onViewportChange))

onMounted(async () => {
  mobileQuery.addEventListener('change', onViewportChange)
  if (authStore.token) {
    await authStore.checkAuth()
    // Temporarily disable chat connection until SQL issues are fixed
    // chatStore.connectSocket()
  }
})
</script>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  height: 100dvh; /* phones: the visible height, excluding browser toolbars */
  background-color: #f5f5f5;
}

/* Sidebar */
.sidebar {
  width: 240px;
  background-color: #ffffff;
  border-right: 1px solid #e0e0e0;
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  position: relative;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-header {
  padding: 1.5rem 1rem;
  border-bottom: 1px solid #e0e0e0;
}

.logo-link {
  display: flex;
  align-items: center;
  text-decoration: none;
  gap: 0.75rem;
}

.logo-icon {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
}

.logo-text {
  font-size: 1.25rem;
  font-weight: 700;
  color: #212529;
  transition: opacity 0.3s;
}

.sidebar.collapsed .logo-text {
  opacity: 0;
  visibility: hidden;
}

/* Navigation */
.sidebar-nav {
  flex: 1;
  padding: 1rem 0;
  overflow-y: auto;
}

.nav-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.nav-button {
  width: 100%;
  border: none;
  background: none;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.nav-item {
  display: flex;
  align-items: center;
  padding: 0.75rem 1rem;
  margin: 0.25rem 0.5rem;
  text-decoration: none;
  color: #6c757d;
  border-radius: 8px;
  transition: all 0.2s;
  position: relative;
  gap: 0.75rem;
}

.nav-item:hover {
  background-color: #f8f9fa;
  color: #212529;
}

.nav-item.active {
  background-color: #eef2ff;
  color: var(--color-primary);
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.nav-text {
  font-size: 0.875rem;
  font-weight: 500;
  white-space: nowrap;
  transition: opacity 0.3s;
}

.sidebar.collapsed .nav-text {
  opacity: 0;
  visibility: hidden;
}

.nav-badge {
  margin-left: auto;
  background-color: #dc3545;
  color: white;
  font-size: 0.75rem;
  padding: 0.125rem 0.5rem;
  border-radius: 12px;
  font-weight: 600;
}

/* User Section */
.sidebar-footer {
  border-top: 1px solid #e0e0e0;
  padding: 1rem;
  position: relative;
}

.user-section {
  text-decoration: none;
  color: inherit;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.user-section:hover {
  background-color: #f8f9fa;
}

.user-avatar {
  width: 40px;
  height: 40px;
  background-color: var(--color-primary);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
  flex-shrink: 0;
}

.user-info {
  overflow: hidden;
}

.user-name {
  font-size: 0.875rem;
  font-weight: 600;
  color: #212529;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-email {
  font-size: 0.75rem;
  color: #6c757d;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar.collapsed .user-info {
  display: none;
}

/* Main Wrapper */
.main-wrapper {
  flex: 1;
  min-width: 0; /* let wide content shrink instead of widening the page */
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Top Bar */
.top-bar {
  height: 64px;
  background: white;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  align-items: center;
  padding: 0 2rem;
  gap: 2rem;
}

.sidebar-toggle {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: #6c757d;
  border-radius: 4px;
  transition: all 0.2s;
}

.sidebar-toggle:hover {
  background-color: #f8f9fa;
  color: #212529;
}





.user-avatar-small {
  width: 32px;
  height: 32px;
  background-color: var(--color-primary);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.75rem;
  cursor: pointer;
  transition: opacity 0.2s;
}

.user-avatar-small:hover {
  opacity: 0.8;
}

/* Main Content */
.main-content {
  flex: 1;
  overflow-y: auto;
  background-color: #f5f5f5;
}

/* Responsive */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 1000;
    transform: translateX(-100%);
    transition: transform 0.3s ease;
  }
  
  .sidebar.mobile-open {
    transform: translateX(0);
    box-shadow: 0 0 24px rgba(0, 0, 0, 0.2);
  }

  .sidebar-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    z-index: 999;
  }
  
  .main-wrapper {
    margin-left: 0;
  }
  
  .top-bar {
    padding: 0 1rem;
  }

  /* Room to scroll the last buttons above the floating chat button */
  .main-content.has-chat-widget {
    padding-bottom: 5.5rem;
  }
}
</style>