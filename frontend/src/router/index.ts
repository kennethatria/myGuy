import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { setPageTitle } from '@/utils/pageTitle'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomeView.vue'),
      meta: { requiresGuest: true } // signed-in users go to their dashboard
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { title: 'Sign in', requiresGuest: true }
    },
    {
      // Sign-up happens in the login flow (email code); keep old links working.
      path: '/register',
      name: 'register',
      redirect: { name: 'login' }
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/views/tasks/DashboardView.vue'),
      meta: { title: 'Dashboard', requiresAuth: true }
    },
    {
      path: '/tasks',
      name: 'tasks',
      component: () => import('@/views/tasks/TaskListView.vue'),
      meta: { title: 'Gigs', requiresAuth: true }
    },
    {
      path: '/tasks/create',
      name: 'create-task',
      component: () => import('@/views/tasks/CreateTaskView.vue'),
      meta: { title: 'Post a Gig', requiresAuth: true }
    },
    {
      path: '/tasks/:id',
      name: 'task-detail',
      component: () => import('@/views/tasks/TaskDetailView.vue'),
      meta: { title: 'Gig', requiresAuth: true }
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/profile/ProfileView.vue'),
      meta: { title: 'My Profile', requiresAuth: true }
    },
    {
      path: '/profile/:id',
      name: 'user-profile',
      component: () => import('@/views/profile/UserProfileView.vue'),
      meta: { title: 'Profile', requiresAuth: true }
    },
    {
      path: '/reviews/create/:taskId',
      name: 'create-review',
      component: () => import('@/views/reviews/CreateReviewView.vue'),
      meta: { title: 'Leave a Review', requiresAuth: true }
    },
    {
      path: '/messages',
      name: 'messages',
      component: () => import('@/views/messages/MessageCenter.vue'),
      meta: { title: 'Messages', requiresAuth: true }
    },
    {
      path: '/store',
      name: 'store',
      component: () => import('@/views/store/StoreView.vue'),
      meta: { title: 'Marketplace', requiresAuth: true }
    },
    {
      path: '/store/:id',
      name: 'store-item',
      component: () => import('@/views/store/StoreItemView.vue'),
      meta: { title: 'Marketplace', requiresAuth: true }
    },
    {
      // Anything unmatched (old or mistyped links)
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { title: 'Page not found' }
    }
  ]
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  const isAuthenticated = await authStore.checkAuth()

  if (to.meta.requiresAuth && !isAuthenticated) {
    // Redirect to login if trying to access protected route
    next({
      name: 'login',
      query: { redirect: to.fullPath }
    })
  } else if (to.meta.requiresGuest && isAuthenticated) {
    // Redirect to dashboard if trying to access guest route while authenticated
    next({ name: 'dashboard' })
  } else {
    next()
  }
})

// Detail pages (gig, store item) replace this with the loaded title.
router.afterEach((to) => {
  setPageTitle(to.meta.title as string | undefined)
})

export default router
