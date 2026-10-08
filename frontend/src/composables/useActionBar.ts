import { readonly, ref } from 'vue'

// The height of the action bar pinned to the bottom of a detail page (0 when
// there is none), so the floating chat button can sit just above it
const height = ref(0)

export const actionBarHeight = readonly(height)

export function setActionBarHeight(value: number) {
  height.value = value
}
