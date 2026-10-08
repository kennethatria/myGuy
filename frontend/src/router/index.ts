import { createRouter, createWebHistory, type RouteLocationRaw, type RouteLocationNormalizedLoaded } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { setPageTitle } from '@/utils/pageTitle'
import { reloadOnStaleBuild, clearStaleBuildReload } from '@/utils/staleBuild'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    requiresAuth?: boolean
    requiresGuest?: boolean
    // Detail pages show a back arrow instead of the menu; this is where it
    // goes when there's no page to go back to (a shared link)
    back?: RouteLocationRaw
    // The page's title, shown in the header (pages without one keep their own)
    heading?: string | ((route: RouteLocationNormalizedLoaded) => string)
    // Forms with a button pinned to the bottom, where the chat button would cover it
    hideChatButton?: boolean
  }
}

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
      meta: { title: 'Home', heading: 'Near you', requiresAuth: true }
    },
    {
      path: '/tasks',
      name: 'tasks',
      component: () => import('@/views/tasks/TaskListView.vue'),
      meta: { title: 'Gigs', heading: 'Gigs', requiresAuth: true }
    },
    {
      path: '/tasks/create',
      name: 'create-task',
      component: () => import('@/views/tasks/CreateTaskView.vue'),
      meta: { title: 'Post a Gig', heading: 'New post', hideChatButton: true, requiresAuth: true, back: { name: 'tasks' } }
    },
    {
      // Straight after posting: the note, with Edit and Remove
      path: '/posted/:kind(task|item|request)/:id(\\d+)',
      name: 'posted',
      component: () => import('@/views/PostedView.vue'),
      meta: { title: 'Posted', heading: 'Your post', hideChatButton: true, requiresAuth: true }
    },
    {
      // Change a gig's or listing's headline and note (requests can't be edited)
      path: '/edit/:kind(task|item)/:id(\\d+)',
      name: 'edit-post',
      component: () => import('@/views/EditPostView.vue'),
      meta: { title: 'Edit post', heading: 'Edit post', hideChatButton: true, requiresAuth: true, back: { name: 'my-gigs', params: { tab: 'created' } } }
    },
    {
      path: '/tasks/:id',
      name: 'task-detail',
      component: () => import('@/views/tasks/TaskDetailView.vue'),
      meta: { title: 'Gig', heading: 'Gig', requiresAuth: true, back: { name: 'tasks' } }
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
      meta: { title: 'Profile', requiresAuth: true, back: { name: 'dashboard' } }
    },
    {
      // Someone else's network, ratings only
      path: '/reviews/:userId(\\d+)',
      name: 'user-network',
      component: () => import('@/views/reviews/ReviewsView.vue'),
      meta: { title: 'Network', heading: 'Network', requiresAuth: true, back: { name: 'reviews' } }
    },
    {
      path: '/reviews',
      name: 'reviews',
      component: () => import('@/views/reviews/ReviewsView.vue'),
      meta: { title: 'Network', heading: 'Network', requiresAuth: true }
    },
    {
      // Reviews are left in the gig's conversation now; old links go to the gig
      path: '/reviews/create/:taskId',
      redirect: to => ({ name: 'task-detail', params: { id: to.params.taskId } })
    },
    {
      // Conversations live in the floating chat; old links land on home
      path: '/messages',
      redirect: { name: 'dashboard' }
    },
    {
      // Your own gigs and applications, one list each (side navigation)
      path: '/my-gigs/:tab(created|assigned|applications)',
      name: 'my-gigs',
      component: () => import('@/views/tasks/MyGigsView.vue'),
      meta: {
        title: 'My Gigs',
        heading: (route) => ({ created: 'My stuff', assigned: 'Assignments', applications: 'Applications' })[route.params.tab as string] ?? 'My stuff',
        requiresAuth: true
      }
    },
    {
      path: '/store',
      name: 'store',
      component: () => import('@/views/store/StoreView.vue'),
      meta: { title: 'Marketplace', heading: 'Marketplace', requiresAuth: true }
    },
    {
      path: '/store/new',
      name: 'create-listing',
      component: () => import('@/views/store/CreateListingView.vue'),
      meta: { title: 'Post Item', heading: 'New post', hideChatButton: true, requiresAuth: true, back: { name: 'store' } }
    },
    {
      path: '/store/requests/new',
      name: 'create-request',
      component: () => import('@/views/store/CreateRequestView.vue'),
      meta: { title: 'Post Request', heading: 'New post', hideChatButton: true, requiresAuth: true, back: { name: 'store', query: { tab: 'wanted' } } }
    },
    {
      path: '/store/requests/:id',
      name: 'store-request',
      component: () => import('@/views/store/RequestView.vue'),
      meta: { title: 'Request', heading: 'Wanted', requiresAuth: true, back: { name: 'store', query: { tab: 'wanted' } } }
    },
    {
      path: '/store/:id',
      name: 'store-item',
      component: () => import('@/views/store/StoreItemView.vue'),
      meta: { title: 'Marketplace', heading: 'For sale', requiresAuth: true, back: { name: 'store' } }
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
router.afterEach((to, _from, failure) => {
  setPageTitle(to.meta.title as string | undefined)
  if (!failure) clearStaleBuildReload()
})

// A tab opened before a deploy can't load the new build's pages: load the
// page the user asked for fresh instead of silently staying put.
router.onError((error, to) => {
  if (!reloadOnStaleBuild(error, router.resolve(to).href)) {
    console.error('Navigation failed:', error)
  }
})

export default router
