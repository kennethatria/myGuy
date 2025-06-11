<template>
  <div class="app-layout">
    <!-- Sidebar -->
    <aside class="sidebar" :class="{ 'collapsed': isSidebarCollapsed }">
      <div class="sidebar-header">
        <router-link :to="{ name: 'dashboard' }" class="logo-link">
          <img class="logo-icon" src="../assets/myguy-icon.svg" alt="MyGuy" />
          <span v-if="!isSidebarCollapsed" class="logo-text">myguy</span>
        </router-link>
      </div>
      
      <nav class="sidebar-nav">
        <ul class="nav-list">
          <li v-for="item in mainNavigation" :key="item.name">
            <router-link
              :to="item.to"
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
        
        <ul class="nav-list nav-secondary">
          <li v-for="item in secondaryNavigation" :key="item.name">
            <router-link
              :to="item.to"
              class="nav-item"
              :class="{ 'active': isActiveRoute(item) }"
              :title="item.text"
            >
              <span class="nav-icon" v-html="item.icon"></span>
              <span v-if="!isSidebarCollapsed" class="nav-text">{{ item.text }}</span>
            </router-link>
          </li>
        </ul>
      </nav>
      
      <div class="sidebar-footer">
        <div class="user-section" @click="toggleUserMenu">
          <div class="user-avatar">
            <span>{{ userInitials }}</span>
          </div>
          <div v-if="!isSidebarCollapsed" class="user-info">
            <div class="user-name">{{ user?.fullName || 'User' }}</div>
            <div class="user-email">{{ user?.email || '' }}</div>
          </div>
        </div>
        
        <transition name="slide-up">
          <div v-if="isUserMenuOpen && !isSidebarCollapsed" class="user-menu">
            <router-link :to="{ name: 'profile' }" class="menu-item" @click="isUserMenuOpen = false">
              Profile
            </router-link>
            <a href="#" class="menu-item" @click="handleSignOut">
              Sign out
            </a>
          </div>
        </transition>
      </div>
    </aside>
    
    <!-- Main Content -->
    <div class="main-wrapper">
      <!-- Top Bar -->
      <header class="top-bar">
        <button class="sidebar-toggle" @click="toggleSidebar">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path d="M3 12H21M3 6H21M3 18H21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </button>
        
        <div class="search-bar">
          <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none">
            <path d="M21 21L15 15M17 10C17 13.866 13.866 17 10 17C6.13401 17 3 13.866 3 10C3 6.13401 6.13401 3 10 3C13.866 3 17 6.13401 17 10Z" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
          <input type="search" placeholder="Search tasks..." v-model="searchQuery" @keyup.enter="handleSearch">
        </div>
        
        <div class="top-bar-actions">
          <button class="notification-btn">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M18 8A6 6 0 1 0 6 8C6 15 3 17 3 17H21C21 17 18 15 18 8Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M13.73 21C13.5542 21.3031 13.3019 21.5547 12.9982 21.7295C12.6946 21.9044 12.3504 21.9965 12 21.9965C11.6496 21.9965 11.3054 21.9044 11.0018 21.7295C10.6982 21.5547 10.4458 21.3031 10.27 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span v-if="unreadNotifications > 0" class="notification-badge">{{ unreadNotifications }}</span>
          </button>
          
          <div class="user-avatar-small" @click="toggleUserMenu">
            <span>{{ userInitials }}</span>
          </div>
        </div>
      </header>
      
      <!-- Page Content -->
      <main class="main-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const chatStore = useChatStore()

const isSidebarCollapsed = ref(false)
const isUserMenuOpen = ref(false)
const searchQuery = ref('')
const unreadNotifications = ref(0)

const user = computed(() => authStore.user)
const totalUnreadCount = computed(() => chatStore.totalUnreadCount)

const mainNavigation = computed(() => [
  { 
    name: 'dashboard', 
    to: { name: 'dashboard' }, 
    text: 'Dashboard',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9L12 2L21 9V20C21 20.5304 20.7893 21.0391 20.4142 21.4142C20.0391 21.7893 19.5304 22 19 22H5C4.46957 22 3.96086 21.7893 3.58579 21.4142C3.21071 21.0391 3 20.5304 3 20V9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>'
  },
  { 
    name: 'tasks', 
    to: { name: 'tasks' }, 
    text: 'Browse Gigs',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M21 21L15 15M17 10C17 13.866 13.866 17 10 17C6.13401 17 3 13.866 3 10C3 6.13401 6.13401 3 10 3C13.866 3 17 6.13401 17 10Z" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>'
  },
  { 
    name: 'create-task', 
    to: { name: 'create-task' }, 
    text: 'Post a Gig',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M12 5V19M5 12H19" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>'
  },
  { 
    name: 'messages', 
    to: { name: 'messages' }, 
    text: 'Messages',
    badge: totalUnreadCount.value > 0 ? totalUnreadCount.value : undefined,
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M21 15C21 15.5304 20.7893 16.0391 20.4142 16.4142C20.0391 16.7893 19.5304 17 19 17H7L3 21V5C3 4.46957 3.21071 3.96086 3.58579 3.58579C3.96086 3.21071 4.46957 3 5 3H19C19.5304 3 20.0391 3.21071 20.4142 3.58579C20.7893 3.96086 21 4.46957 21 5V15Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>'
  }
])

const secondaryNavigation = [
  { 
    name: 'store', 
    to: { name: 'store' }, 
    text: 'Store',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9V21H21V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 9H21L19 3H5L3 9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M12 3V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>'
  },
  { 
    name: 'profile', 
    to: { name: 'profile' }, 
    text: 'Profile',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M20 21V19C20 17.9391 19.5786 16.9217 18.8284 16.1716C18.0783 15.4214 17.0609 15 16 15H8C6.93913 15 5.92172 15.4214 5.17157 16.1716C4.42143 16.9217 4 17.9391 4 19V21" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M12 11C14.2091 11 16 9.20914 16 7C16 4.79086 14.2091 3 12 3C9.79086 3 8 4.79086 8 7C8 9.20914 9.79086 11 12 11Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>'
  }
]

const userInitials = computed(() => {
  if (!user.value?.fullName) return '?'
  return user.value.fullName
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
})

const toggleSidebar = () => {
  isSidebarCollapsed.value = !isSidebarCollapsed.value
}

const toggleUserMenu = () => {
  isUserMenuOpen.value = !isUserMenuOpen.value
}

const handleSignOut = async () => {
  try {
    isUserMenuOpen.value = false
    authStore.logout()
    await router.push({ name: 'login' })
  } catch (error) {
    console.error('Sign out failed:', error)
  }
}

const handleSearch = () => {
  if (searchQuery.value.trim()) {
    router.push({ 
      name: 'tasks', 
      query: { search: searchQuery.value.trim() } 
    })
  }
}

const isActiveRoute = (item: any) => {
  return route.name === item.name
}

onMounted(async () => {
  if (authStore.token) {
    await authStore.checkAuth()
    chatStore.connectSocket()
  }
})
</script>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
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

.nav-secondary {
  margin-top: 2rem;
  padding-top: 2rem;
  border-top: 1px solid #e0e0e0;
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
  background-color: #e3f2fd;
  color: #1976d2;
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
  background-color: #1976d2;
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

.user-menu {
  position: absolute;
  bottom: 100%;
  left: 1rem;
  right: 1rem;
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  margin-bottom: 0.5rem;
  overflow: hidden;
}

.menu-item {
  display: block;
  padding: 0.75rem 1rem;
  color: #212529;
  text-decoration: none;
  font-size: 0.875rem;
  transition: background-color 0.2s;
}

.menu-item:hover {
  background-color: #f8f9fa;
}

/* Main Wrapper */
.main-wrapper {
  flex: 1;
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

.search-bar {
  flex: 1;
  max-width: 600px;
  position: relative;
}

.search-bar input {
  width: 100%;
  padding: 0.5rem 1rem 0.5rem 2.5rem;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.search-bar input:focus {
  outline: none;
  border-color: #1976d2;
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
}

.search-icon {
  position: absolute;
  left: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: #6c757d;
  pointer-events: none;
}

.top-bar-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.notification-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: #6c757d;
  border-radius: 4px;
  transition: all 0.2s;
  position: relative;
}

.notification-btn:hover {
  background-color: #f8f9fa;
  color: #212529;
}

.notification-badge {
  position: absolute;
  top: 0;
  right: 0;
  background-color: #dc3545;
  color: white;
  font-size: 0.625rem;
  padding: 0.125rem 0.375rem;
  border-radius: 10px;
  font-weight: 600;
}

.user-avatar-small {
  width: 32px;
  height: 32px;
  background-color: #1976d2;
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

/* Transitions */
.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.3s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(10px);
  opacity: 0;
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
  }
  
  .main-wrapper {
    margin-left: 0;
  }
  
  .search-bar {
    display: none;
  }
  
  .top-bar {
    padding: 0 1rem;
  }
}
</style>