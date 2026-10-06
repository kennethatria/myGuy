import { ref } from 'vue'
import { roughLocation, type RoughLocation } from '@/utils/geoCell'

export type LocationState = 'idle' | 'asking' | 'added' | 'denied' | 'unavailable'

/**
 * The poster's rough location, asked for only when they tap (never on page
 * load). Low accuracy: phones answer from Wi-Fi and cell towers, which is
 * quicker and kinder to the battery; a reading up to 30 minutes old is fine.
 */
export function useRoughLocation() {
  const state = ref<LocationState>('idle')
  const location = ref<RoughLocation | null>(null)

  const request = () => {
    if (!('geolocation' in navigator)) {
      state.value = 'unavailable'
      return
    }
    state.value = 'asking'
    navigator.geolocation.getCurrentPosition(
      (position) => {
        location.value = roughLocation(position.coords.latitude, position.coords.longitude)
        state.value = 'added'
      },
      (error) => {
        location.value = null
        state.value = error.code === error.PERMISSION_DENIED ? 'denied' : 'unavailable'
      },
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 30 * 60 * 1000 }
    )
  }

  const clear = () => {
    location.value = null
    state.value = 'idle'
  }

  return { state, location, request, clear }
}
