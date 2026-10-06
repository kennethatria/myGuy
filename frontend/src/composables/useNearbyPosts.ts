import { ref, onMounted, onUnmounted, watch, type Ref } from 'vue'
import config from '@/config'
import { useAuthStore } from '@/stores/auth'
import { nearParam } from '@/composables/useViewerLocation'
import type { RoughLocation } from '@/utils/geoCell'
import { bucketIndex, type PostKind } from '@/utils/radar'

export interface NearbyPost {
  kind: PostKind
  id: number
  title: string
  distance: string
  bucket: number
}

// Posts last 24 hours and distances don't move, so a few minutes is fresh
// enough (and cheap on mobile data)
export const REFRESH_MS = 5 * 60 * 1000
const PER_KIND = 20

/**
 * Other people's nearest live gigs, listings and requests (those with a
 * location), for the dashboard radar. Loads when a location is known, again
 * every 5 minutes while the page is visible, and on return to the tab if the
 * last load is older than that.
 */
export function useNearbyPosts(location: Ref<RoughLocation | null>) {
  const posts = ref<NearbyPost[]>([])
  const loading = ref(false)
  const failed = ref(false)
  let lastLoad = 0
  let timer: ReturnType<typeof setInterval> | undefined
  const authStore = useAuthStore()

  const fetchKind = async (kind: PostKind, url: string, listKey: string): Promise<NearbyPost[]> => {
    const response = await fetch(url, { headers: { Authorization: `Bearer ${authStore.token}` } })
    if (!response.ok) throw new Error(`${kind} ${response.status}`)
    const data = await response.json()
    return ((data[listKey] ?? []) as { id: number; title: string; distance?: string }[])
      .map((post) => ({ kind, id: post.id, title: post.title, distance: post.distance ?? '', bucket: bucketIndex(post.distance) }))
      .filter((post) => post.bucket >= 0)
  }

  const load = async () => {
    if (!location.value) return
    loading.value = true
    failed.value = false
    const me = String(authStore.user?.id ?? '')
    const near = nearParam(location.value)
    const common = `near=${encodeURIComponent(near)}&sort_by=distance&per_page=${PER_KIND}`
    const results = await Promise.allSettled([
      fetchKind('task', `${config.API_URL}/tasks?status=open&exclude_created_by=${me}&${common}`, 'tasks'),
      fetchKind('item', `${config.STORE_API_URL}/items?status=active&exclude_seller_id=${me}&${common}`, 'items'),
      fetchKind('request', `${config.STORE_API_URL}/requests?exclude_requester_id=${me}&${common}`, 'requests')
    ])
    const loaded = results.flatMap((r) => (r.status === 'fulfilled' ? r.value : []))
    failed.value = results.every((r) => r.status === 'rejected')
    // Nearest first; within a ring, keep each board's own order
    posts.value = loaded.map((post, i) => ({ post, i })).sort((a, b) => a.post.bucket - b.post.bucket || a.i - b.i).map(({ post }) => post)
    lastLoad = Date.now()
    loading.value = false
  }

  const onVisible = () => {
    if (document.visibilityState === 'visible' && Date.now() - lastLoad > REFRESH_MS) load()
  }

  onMounted(() => {
    load()
    timer = setInterval(() => {
      if (document.visibilityState === 'visible') load()
    }, REFRESH_MS)
    document.addEventListener('visibilitychange', onVisible)
  })
  onUnmounted(() => {
    if (timer) clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisible)
  })
  watch(location, (now, before) => {
    if (now && (!before || now.lat !== before.lat || now.lng !== before.lng)) load()
  })

  return { posts, loading, failed, reload: load }
}
