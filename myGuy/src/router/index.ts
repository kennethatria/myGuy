import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomeView.vue')
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { requiresGuest: true }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/auth/RegisterView.vue'),
      meta: { requiresGuest: true }
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/views/tasks/DashboardView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/tasks',
      name: 'tasks',
      component: () => import('@/views/tasks/TaskListView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/tasks/create',
      name: 'create-task',
      component: () => import('@/views/tasks/CreateTaskView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/tasks/:id',
      name: 'task-details',
      component: () => import('@/views/tasks/TaskDetailView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/profile/ProfileView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/profile/:id',
      name: 'user-profile',
      component: () => import('@/views/profile/UserProfileView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/reviews/create/:taskId',
      name: 'create-review',
      component: () => import('@/views/reviews/CreateReviewView.vue'),
      meta: { requiresAuth: true }
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

export default router
