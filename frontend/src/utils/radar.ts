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

/** Outer radius of each bucket's ring: <1 km innermost, 10+ km outermost. */
export const RING_RADIUS = [11, 18, 25, 32, 40] as const

/**
 * Posts whose poster shared no location sit on one more, dotted ring at the
 * edge (UNKNOWN_RING), labelled "No location".
 */
export const UNKNOWN_RING = RING_RADIUS.length
export const EDGE_RADIUS = 48
export const RING_LABELS = [...BUCKETS, 'No location'] as const

/** Outer radius of a ring, the "No location" ring included. */
export function ringRadius(ring: number): number {
  return ring >= UNKNOWN_RING ? EDGE_RADIUS : RING_RADIUS[ring]
}

/** The ring a post's dot goes on: its distance bucket, or the edge ring. */
export function ringOf(bucket: number): number {
  return bucket < 0 ? UNKNOWN_RING : Math.min(bucket, RING_RADIUS.length - 1)
}

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

/** Radius of the circle a ring's dots sit on: midway through its band. */
export function dotRadius(ring: number): number {
  const r = Math.min(Math.max(ring, 0), UNKNOWN_RING)
  const inner = r === 0 ? 4 : ringRadius(r - 1)
  return (inner + ringRadius(r)) / 2
}

/** Space each dot needs along its circle so neighbours never touch. */
export const DOT_SPACING = 4.2

/** How many dots fit round a circle of radius r, outside the label's arc. */
function capacityAt(r: number): number {
  const usable = (360 - 2 * LABEL_CLEAR) / 360
  return Math.max(1, Math.floor((2 * Math.PI * r * usable) / DOT_SPACING))
}

/** Gap kept between a busy ring's two tracks and its band edges. */
const TRACK_INSET = 1.8

/**
 * The circles a ring's dots sit on: one, midway through its band, or two
 * (near each edge of the band) when there are more than one circle holds.
 */
export function trackRadii(ring: number, count: number): number[] {
  const r = Math.min(Math.max(ring, 0), UNKNOWN_RING)
  if (count <= capacityAt(dotRadius(r))) return [dotRadius(r)]
  const inner = r === 0 ? 4 : ringRadius(r - 1)
  return [inner + TRACK_INSET, ringRadius(r) - TRACK_INSET]
}

/** How many dots a ring can show without overlapping (both tracks). */
export function ringCapacity(ring: number): number {
  return trackRadii(ring, Infinity).reduce((sum, r) => sum + capacityAt(r), 0)
}

/** Where a ring's label goes: just inside the ring, at LABEL_ANGLE. */
export function labelPoint(ring: number): { x: number; y: number } {
  const r = ringRadius(ring) - 2.2
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
export function ringPoint(ring: number, t: number, radius = dotRadius(ring)): { x: number; y: number } {
  const usable = 360 - 2 * LABEL_CLEAR
  const deg = LABEL_ANGLE + LABEL_CLEAR + (((t % 1) + 1) % 1) * usable
  const rad = (deg * Math.PI) / 180
  return { x: CENTRE + radius * Math.sin(rad), y: CENTRE - radius * Math.cos(rad) }
}

/** Where one post's dot goes on its own (no other posts to make room for). */
export function radarPoint(kind: PostKind, id: number, bucket: number): { x: number; y: number } {
  return ringPoint(ringOf(bucket), postT(kind, id))
}

export interface RadarPost {
  kind: PostKind
  id: number
  bucket: number
}

export interface RadarLayout {
  spots: Map<string, { x: number; y: number }>
  // Posts per ring that didn't fit (ring index -> count)
  hidden: Map<number, number>
}

/**
 * Spots for every dot, keyed "kind-id". Posts that share a ring are spaced
 * evenly round it (on two tracks when busy), so dots never pile up; a ring
 * draws at most
 * ringCapacity(ring) of them (the first ones given, i.e. the nearest) and
 * counts the rest in hidden.
 */
export function layoutDots(posts: RadarPost[]): RadarLayout {
  const byRing = new Map<number, RadarPost[]>()
  for (const post of posts) {
    const ring = ringOf(post.bucket)
    byRing.set(ring, [...(byRing.get(ring) ?? []), post])
  }
  const spots = new Map<string, { x: number; y: number }>()
  const hidden = new Map<number, number>()
  for (const [ring, onRing] of byRing) {
    const capacity = ringCapacity(ring)
    if (onRing.length > capacity) hidden.set(ring, onRing.length - capacity)
    const shown = onRing.slice(0, capacity)
    // Fill the inner track first, the rest go on the outer one
    let next = 0
    trackRadii(ring, shown.length).forEach((radius, track) => {
      const onTrack = shown
        .slice(next, next + capacityAt(radius))
        .sort((a, b) => postT(a.kind, a.id) - postT(b.kind, b.id))
      next += onTrack.length
      if (!onTrack.length) return
      // Offset the outer track by half a step so its dots sit between the inner ones
      const start = postT(onTrack[0].kind, onTrack[0].id) + track / (2 * onTrack.length)
      onTrack.forEach((post, i) => {
        spots.set(`${post.kind}-${post.id}`, ringPoint(ring, start + i / onTrack.length, radius))
      })
    })
  }
  return { spots, hidden }
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
