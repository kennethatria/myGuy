// Geometry for the "near you" radar: a square of concentric square rings,
// one per distance bucket, with "you" in the middle. A dot's place along its
// ring comes from a hash of the post, so it is stable between visits but says
// nothing about real direction (only rough distance is ever shared).

export const BUCKETS = ['<1 km', '~2 km', '~5 km', '~10 km', '10+ km'] as const

export type PostKind = 'task' | 'item' | 'request'

export const KIND_STYLE: Record<PostKind, { fill: string; stroke: string; one: string; many: string }> = {
  task: { fill: '#2563eb', stroke: '#1e3a8a', one: 'gig', many: 'gigs' },
  item: { fill: '#facc15', stroke: '#854d0e', one: 'listing', many: 'listings' },
  request: { fill: '#dc2626', stroke: '#7f1d1d', one: 'request', many: 'requests' }
}

/** The SVG is drawn in a 100 x 100 box centred on (50, 50). */
export const CENTRE = 50

/** Outer half-size of each bucket's ring: <1 km innermost, 10+ km at the edge. */
export const RING_HALF = [10, 18, 27, 36, 46] as const

/** Index of a distance tag in BUCKETS, or -1 when unknown. */
export function bucketIndex(label?: string): number {
  return label ? BUCKETS.indexOf(label as (typeof BUCKETS)[number]) : -1
}

/** FNV-1a: a small, stable hash so a post keeps its spot between visits. */
function hash(text: string): number {
  let h = 0x811c9dc5
  for (let i = 0; i < text.length; i++) {
    h ^= text.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return h >>> 0
}

/**
 * Length of each ring's top edge, at its right end, kept free for its label
 * (about one label wide plus a dot), capped so small rings keep some edge.
 */
export const LABEL_CLEAR = 11

export function labelGap(side: number): number {
  return Math.min(LABEL_CLEAR, 0.8 * side)
}

/**
 * Where a post's dot goes: on the square midway through its bucket's band,
 * at a hashed point along that square's edge, never in the top-right stretch
 * where the ring's distance label sits.
 */
export function radarPoint(kind: PostKind, id: number, bucket: number): { x: number; y: number } {
  const b = Math.min(Math.max(bucket, 0), RING_HALF.length - 1)
  const inner = b === 0 ? 4 : RING_HALF[b - 1]
  const half = (inner + RING_HALF[b]) / 2
  const t = (hash(`${kind}:${id}`) % 10_000) / 10_000
  const side = 2 * half
  const gap = labelGap(side)
  // distance travelled clockwise from the top-left corner, skipping the gap
  let along = t * (8 * half - gap)
  if (along >= side - gap) along += gap
  const left = CENTRE - half
  const top = CENTRE - half
  if (along < side) return { x: left + along, y: top }
  if (along < 2 * side) return { x: left + side, y: top + (along - side) }
  if (along < 3 * side) return { x: left + side - (along - 2 * side), y: top + side }
  return { x: left, y: top + side - (along - 3 * side) }
}

/** "3 gigs, 1 listing and 2 requests" (kinds with none left out). */
export function countsSentence(counts: Record<PostKind, number>): string {
  const parts = (Object.keys(KIND_STYLE) as PostKind[])
    .filter((kind) => counts[kind] > 0)
    .map((kind) => `${counts[kind]} ${counts[kind] === 1 ? KIND_STYLE[kind].one : KIND_STYLE[kind].many}`)
  if (parts.length === 0) return 'nothing yet'
  if (parts.length === 1) return parts[0]
  return `${parts.slice(0, -1).join(', ')} and ${parts[parts.length - 1]}`
}
