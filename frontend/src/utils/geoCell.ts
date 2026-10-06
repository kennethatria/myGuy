// Rough locations: a position is snapped to a cell about 555 m across before
// it leaves the browser, and the proximity service snaps it again. The rule
// is shared with the backend, store-service and proximity service through
// shared/geo-cell-cases.json; change it in all of them.

export const CELL_STEP = 0.005

/** floor(x / step + 0.5): halves round up the same way in every language. */
export function cell(degrees: number): number {
  return Math.floor(degrees / CELL_STEP + 0.5)
}

export interface RoughLocation {
  lat: number
  lng: number
}

/** A position snapped to its cell centre (what gets sent, never the raw one). */
export function roughLocation(lat: number, lng: number): RoughLocation {
  // toFixed keeps 0.35 from going out as 0.35000000000000003
  return {
    lat: Number((cell(lat) * CELL_STEP).toFixed(3)),
    lng: Number((cell(lng) * CELL_STEP).toFixed(3))
  }
}
