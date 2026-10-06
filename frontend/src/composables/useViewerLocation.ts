import { ref, onMounted } from 'vue'
import { roughLocation, type RoughLocation } from '@/utils/geoCell'

// The viewer's rough position, shared by every board: only the ~500 m cell
// is kept, in this tab's session storage, for 30 minutes.
const CACHE_KEY = 'myguy:rough-location'
const MAX_AGE_MS = 30 * 60 * 1000

export type ViewerLocationState = 'unknown' | 'asking' | 'ready' | 'denied' | 'unavailable'

export function readCachedLocation(now = Date.now()): RoughLocation | null {
  try {
    const raw = sessionStorage.getItem(CACHE_KEY)
    if (!raw) return null
    const cached = JSON.parse(raw) as RoughLocation & { at: number }
    if (typeof cached.lat !== 'number' || typeof cached.lng !== 'number' || now - cached.at > MAX_AGE_MS) return null
    return { lat: cached.lat, lng: cached.lng }
  } catch {
    return null
  }
}

export function cacheLocation(location: RoughLocation, now = Date.now()) {
  try {
    sessionStorage.setItem(CACHE_KEY, JSON.stringify({ ...location, at: now }))
  } catch {
    // Without storage the board just asks again next time
  }
}

/** "lat,lng" for the boards' near= query value. */
export const nearParam = (location: RoughLocation) => `${location.lat},${location.lng}`

/**
 * The viewer's rough location for sorting boards. It never prompts on its
 * own: it uses a recent cached cell, or reads the position silently when
 * the site already has permission; otherwise it waits for request() (a tap).
 */
export function useViewerLocation(onReady?: () => void) {
  const state = ref<ViewerLocationState>('unknown')
  const location = ref<RoughLocation | null>(readCachedLocation())
  if (location.value) state.value = 'ready'

  const read = () => {
    if (!('geolocation' in navigator)) {
      state.value = 'unavailable'
      return
    }
    state.value = 'asking'
    navigator.geolocation.getCurrentPosition(
      (position) => {
        location.value = roughLocation(position.coords.latitude, position.coords.longitude)
        cacheLocation(location.value)
        state.value = 'ready'
        onReady?.()
      },
      (error) => {
        state.value = error.code === error.PERMISSION_DENIED ? 'denied' : 'unavailable'
      },
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: MAX_AGE_MS }
    )
  }

  onMounted(async () => {
    if (state.value === 'ready') return
    try {
      const permission = await navigator.permissions?.query({ name: 'geolocation' as PermissionName })
      if (permission?.state === 'granted') read()
      else if (permission?.state === 'denied') state.value = 'denied'
    } catch {
      // No permissions API (older Safari): wait for a tap
    }
  })

  return { state, location, request: read }
}
