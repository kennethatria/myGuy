<template>
  <div class="app-layout">
    <header class="top-bar">
      <!-- Detail pages go back; every other page opens the menu -->
      <button v-if="backTarget" class="bar-button" aria-label="Back" @click="goBack">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M15 5l-7 7 7 7" />
        </svg>
      </button>
      <button
        v-else
        ref="menuButton"
        class="bar-button"
        aria-label="Menu"
        aria-controls="app-drawer"
        :aria-expanded="isDrawerOpen"
        @click="openDrawer"
      >
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <path d="M4 7h16M4 12h16M4 17h16" />
        </svg>
      </button>
      <h1 v-if="heading" class="bar-title">{{ heading }}</h1>
      <span v-else class="bar-spacer"></span>
      <router-link v-if="postTarget" :to="postTarget" class="post-link">+ Post</router-link>
    </header>

    <!-- The menu slides over the page on every screen size -->
    <div v-if="isDrawerOpen" class="drawer-backdrop" @click="closeDrawer"></div>
    <aside
      id="app-drawer"
      class="drawer"
      :class="{ open: isDrawerOpen }"
      aria-label="Menu"
      @keydown.esc="closeDrawer"
    >
      <div class="drawer-header">
        <router-link :to="{ name: 'dashboard' }" class="logo-link">
          <span class="logo-mark" aria-hidden="true">M</span>
          <span class="logo-text">MyGuy</span>
        </router-link>
      </div>

      <nav class="drawer-nav">
        <section v-for="group in navigation" :key="group.title" class="nav-group">
          <h2 class="nav-group-title">{{ group.title }}</h2>
          <ul class="nav-list">
            <li v-for="item in group.items" :key="item.key">
              <router-link
                :to="item.to"
                class="nav-item"
                :class="{ active: isActiveRoute(item) }"
                :aria-current="isActiveRoute(item) ? 'page' : undefined"
              >
                <span class="nav-icon" aria-hidden="true" v-html="item.icon"></span>
                <span class="nav-text">{{ item.text }}</span>
              </router-link>
            </li>
          </ul>
        </section>
      </nav>

      <div class="drawer-footer">
        <router-link :to="{ name: 'profile' }" class="user-section">
          <span class="user-avatar" aria-hidden="true">{{ userInitial }}</span>
          <span class="user-info">
            <span class="user-name">{{ displayName }}</span>
            <span v-if="ratingLabel" class="user-rating">{{ ratingLabel }}</span>
          </span>
        </router-link>
        <button type="button" class="sign-out" @click="handleSignOut">Sign out</button>
      </div>
    </aside>

    <main :class="['main-content', { 'has-chat-button': !route.meta.hideChatButton }]">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRouter, useRoute, type RouteLocationRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const isDrawerOpen = ref(false)
const menuButton = ref<HTMLButtonElement | null>(null)

const user = computed(() => authStore.user)

interface NavItem {
  key: string
  text: string
  icon: string
  to: RouteLocationRaw
  // Highlighted when on this route (and, for My Gigs, this tab)
  route: string
  tab?: string
}

// Messages are not here: they open from the floating chat button. Your
// requests and listings are under Marketplace → Yours.
const navigation: { title: string; items: NavItem[] }[] = [
  {
    title: 'Browse',
    items: [
      { key: 'home', route: 'dashboard', to: { name: 'dashboard' }, text: 'Home', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9L12 2L21 9V20C21 20.5304 20.7893 21.0391 20.4142 21.4142C20.0391 21.7893 19.5304 22 19 22H5C4.46957 22 3.96086 21.7893 3.58579 21.4142C3.21071 21.0391 3 20.5304 3 20V9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
      { key: 'tasks', route: 'tasks', to: { name: 'tasks' }, text: 'Gigs', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M21 21L15 15M17 10C17 13.866 13.866 17 10 17C6.13401 17 3 13.866 3 10C3 6.13401 6.13401 3 10 3C13.866 3 17 6.13401 17 10Z" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>' },
      { key: 'store', route: 'store', to: { name: 'store' }, text: 'Marketplace', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9V21H21V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 9H21L19 3H5L3 9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M12 3V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
      { key: 'reviews', route: 'reviews', to: { name: 'reviews' }, text: 'Network', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2"/><circle cx="4" cy="5" r="2" stroke="currentColor" stroke-width="2"/><circle cx="20" cy="6" r="2" stroke="currentColor" stroke-width="2"/><circle cx="19" cy="19" r="2" stroke="currentColor" stroke-width="2"/><path d="M5.6 6.3L9.7 10M18.3 7.1L14.6 10.4M17.6 17.6L14.2 14.2" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>' }
    ]
  },
  {
    title: 'Mine',
    items: [
      { key: 'created', route: 'my-gigs', tab: 'created', to: { name: 'my-gigs', params: { tab: 'created' } }, text: 'My stuff', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M14 2H6C4.9 2 4 2.9 4 4V20C4 21.1 4.9 22 6 22H18C19.1 22 20 21.1 20 20V8L14 2Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/><path d="M14 2V8H20M8 13H16M8 17H16" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
      { key: 'assigned', route: 'my-gigs', tab: 'assigned', to: { name: 'my-gigs', params: { tab: 'assigned' } }, text: 'Assignments', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M16 21V19C16 16.8 14.2 15 12 15H5C2.8 15 1 16.8 1 19V21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><circle cx="8.5" cy="7" r="4" stroke="currentColor" stroke-width="2"/><path d="M17 11L19 13L23 9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
      { key: 'applications', route: 'my-gigs', tab: 'applications', to: { name: 'my-gigs', params: { tab: 'applications' } }, text: 'Applications', icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M22 2L11 13M22 2L15 22L11 13L2 9L22 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' }
    ]
  }
]

const displayName = computed(() => user.value?.fullName || user.value?.username || 'You')
const userInitial = computed(() => displayName.value.charAt(0).toUpperCase())
const ratingLabel = computed(() => {
  const rating = user.value?.averageRating
  return rating ? `⭐ ${rating.toFixed(1)}` : ''
})

const backTarget = computed(() => route.meta.back)
const heading = computed(() => {
  const value = route.meta.heading
  return typeof value === 'function' ? value(route) : value
})

// Back to where the user came from; a shared link has nowhere to go back
// to, so it goes to the page's parent list instead
const goBack = () => {
  if (window.history.state?.back) router.back()
  else if (backTarget.value) router.push(backTarget.value)
}

// "+ Post" posts what the page is about; hidden while posting or editing
const postTarget = computed<RouteLocationRaw | null>(() => {
  if (['create-task', 'create-listing', 'create-request', 'posted', 'edit-post'].includes(route.name as string)) return null
  if (route.name === 'store' || route.name === 'store-item') {
    return route.query.tab === 'wanted' ? { name: 'create-request' } : { name: 'create-listing' }
  }
  if (route.name === 'store-request') return { name: 'create-request' }
  return { name: 'create-task' }
})

const openDrawer = async () => {
  isDrawerOpen.value = true
  await nextTick()
  document.querySelector<HTMLElement>('#app-drawer .nav-item')?.focus()
}

const closeDrawer = () => {
  if (!isDrawerOpen.value) return
  isDrawerOpen.value = false
  menuButton.value?.focus()
}

// Close the drawer once the user has picked a page
watch(() => route.fullPath, () => {
  isDrawerOpen.value = false
})

const handleSignOut = async () => {
  try {
    authStore.logout()
    await router.push({ name: 'login' })
  } catch (error) {
    console.error('Sign out failed:', error)
  }
}

// My Gigs keeps its tab in the path; the Marketplace item stays
// unhighlighted on the tab with your own listings
const currentTab = computed(() => route.params.tab ?? route.query.tab)
const isActiveRoute = (item: NavItem) =>
  route.name === item.route &&
  (item.tab ? currentTab.value === item.tab : !(item.key === 'store' && currentTab.value === 'mine'))

onMounted(async () => {
  if (authStore.token) {
    await authStore.checkAuth()
  }
})
</script>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh; /* phones: the visible height, excluding browser toolbars */
  background-color: var(--bg);
}

/* Header: menu or back, then "+ Post" */
.top-bar {
  flex: none;
  display: flex;
  align-items: center;
  gap: 4px;
  height: 56px;
  padding: 0 8px;
  background: var(--bg);
}

.bar-button {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  border-radius: 22px;
  background: transparent;
  color: #374151;
  cursor: pointer;
}

.bar-button:hover {
  background: rgba(17, 24, 39, 0.05);
}

.bar-spacer,
.bar-title {
  flex: 1;
}

.bar-title {
  min-width: 0;
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-link {
  display: inline-flex;
  align-items: center;
  height: 44px;
  padding: 0 12px;
  color: var(--accent-text);
  font-size: 15px;
  font-weight: 600;
}

.post-link:hover {
  color: var(--color-primary-dark);
}

/* Drawer */
.drawer-backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
  background: rgba(17, 24, 39, 0.4);
}

.drawer {
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  z-index: 1000;
  width: 300px;
  max-width: 85vw;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  transform: translateX(-100%);
  visibility: hidden; /* out of the tab order and screen readers when shut */
  transition: transform 0.25s ease, visibility 0s linear 0.25s;
}

.drawer.open {
  transform: translateX(0);
  visibility: visible;
  box-shadow: 0 0 24px rgba(17, 24, 39, 0.2);
  transition: transform 0.25s ease;
}

@media (prefers-reduced-motion: reduce) {
  .drawer,
  .drawer.open {
    transition: none;
  }
}

.drawer-header {
  flex: none;
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  border-bottom: 1px solid #EEF0F3;
}

.logo-link {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text);
}

.logo-mark {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--accent);
  color: var(--on-accent);
  font-size: 15px;
  font-weight: 700;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
}

.drawer-nav {
  flex: 1;
  overflow-y: auto;
  padding: 12px 12px 0;
}

.nav-group + .nav-group .nav-group-title {
  padding-top: 20px;
}

.nav-group-title {
  margin: 0;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.5;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #9CA3AF;
}

.nav-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  height: 44px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 12px;
  border-radius: 10px;
  color: #374151;
  font-size: 16px;
}

.nav-item:hover {
  color: var(--text);
  background: #F7F7F5;
}

.nav-item.active {
  background: #F5F6FF;
  color: var(--accent-text);
  font-weight: 600;
}

.nav-icon {
  display: flex;
  color: var(--text-muted);
}

.nav-item.active .nav-icon {
  color: var(--accent);
}

.drawer-footer {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px 24px;
  border-top: 1px solid #EEF0F3;
}

.user-section {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text);
}

.user-avatar {
  flex: none;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 22px;
  background: var(--accent);
  color: var(--on-accent);
  font-size: 17px;
  font-weight: 600;
}

.user-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.user-name {
  font-size: 16px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-rating {
  font-size: 13px;
  color: var(--text-muted);
}

.sign-out {
  border: 0;
  background: transparent;
  padding: 10px 0;
  color: var(--text-muted);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.sign-out:hover {
  color: var(--text);
}

/* Page */
.main-content {
  flex: 1;
  overflow-y: auto;
  background-color: var(--bg);
}

/* Room to scroll the last buttons above the floating chat button */
.main-content.has-chat-button {
  padding-bottom: 5.5rem;
}
</style>
