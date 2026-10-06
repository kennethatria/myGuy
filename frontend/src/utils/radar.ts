// Geometry for the "near you" radar: concentric circles, one per distance
// bucket, with "you" in the middle. A dot's angle on its circle comes from a
// hash of the post, so it is stable between visits but says nothing about
// real direction (only rough distance is ever shared).

export const BUCKETS = ['<1 km', '~2 km', '~5 km', '~10 km', '10+ km'] as const

export type PostKind = 'task' | 'item' | 'request'

export const KIND_STYLE: Record<PostKind, { fill: string; stroke: string; one: string; many: string }> = {
  task: { fill: '#2563eb', stroke: '#1e3a8a', one: 'gig', many: 'gigs' },
  item: { fill: '#facc15', stroke: '#854d0e', one: 'marketplace item', many: 'marketplace items' },
  request: { fill: '#dc2626', stroke: '#7f1d1d', one: 'request', many: 'requests' }
}

/** The SVG is drawn in a 100 x 100 box centred on (50, 50). */
export const CENTRE = 50

/** Outer radius of each bucket's ring: <1 km innermost, 10+ km at the edge. */
export const RING_RADIUS = [10, 18, 27, 36, 46] as const

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
 * Each ring's label sits at the top right (LABEL_ANGLE, measured clockwise
 * from 12 o'clock); dots keep LABEL_CLEAR degrees either side of it free.
 */
export const LABEL_ANGLE = 45
export const LABEL_CLEAR = 22

/** Radius of the circle a bucket's dots sit on: midway through its band. */
export function dotRadius(bucket: number): number {
  const b = Math.min(Math.max(bucket, 0), RING_RADIUS.length - 1)
  const inner = b === 0 ? 4 : RING_RADIUS[b - 1]
  return (inner + RING_RADIUS[b]) / 2
}

/** Where a ring's label goes: just inside the ring, at LABEL_ANGLE. */
export function labelPoint(bucket: number): { x: number; y: number } {
  const r = RING_RADIUS[bucket] - 2.2
  const rad = (LABEL_ANGLE * Math.PI) / 180
  return { x: CENTRE + r * Math.sin(rad), y: CENTRE - r * Math.cos(rad) + 1 }
}

/** A post's own spot along its ring, 0 to 1: stable between visits. */
export function postT(kind: PostKind, id: number): number {
  return (hash(`${kind}:${id}`) % 10_000) / 10_000
}

/**
 * A point a fraction t (0 to 1) of the way round a bucket's circle,
 * clockwise from just past its label, never in the arc around the label.
 */
export function ringPoint(bucket: number, t: number): { x: number; y: number } {
  const usable = 360 - 2 * LABEL_CLEAR
  const deg = LABEL_ANGLE + LABEL_CLEAR + (((t % 1) + 1) % 1) * usable
  const rad = (deg * Math.PI) / 180
  const r = dotRadius(bucket)
  return { x: CENTRE + r * Math.sin(rad), y: CENTRE - r * Math.cos(rad) }
}

/** Where one post's dot goes on its own (no other posts to make room for). */
export function radarPoint(kind: PostKind, id: number, bucket: number): { x: number; y: number } {
  return ringPoint(bucket, postT(kind, id))
}

export interface RadarPost {
  kind: PostKind
  id: number
  bucket: number
}

/**
 * Spots for every dot, keyed "kind-id". Posts that share a circle are spaced
 * evenly round it (in the order of their own spots, starting from the
 * first one's), so dots never pile up however many are close by.
 */
export function layoutDots(posts: RadarPost[]): Map<string, { x: number; y: number }> {
  const byRing = new Map<number, RadarPost[]>()
  for (const post of posts) {
    const ring = Math.min(Math.max(post.bucket, 0), RING_RADIUS.length - 1)
    byRing.set(ring, [...(byRing.get(ring) ?? []), post])
  }
  const spots = new Map<string, { x: number; y: number }>()
  for (const [ring, onRing] of byRing) {
    const ordered = [...onRing].sort((a, b) => postT(a.kind, a.id) - postT(b.kind, b.id))
    const start = postT(ordered[0].kind, ordered[0].id)
    ordered.forEach((post, i) => {
      spots.set(`${post.kind}-${post.id}`, ringPoint(ring, start + i / ordered.length))
    })
  }
  return spots
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
