import { describe, it, expect } from 'vitest'
import { bucketIndex, radarPoint, layoutDots, countsSentence, dotRadius, labelPoint, ringRadius, ringCapacity, RING_RADIUS, UNKNOWN_RING, EDGE_RADIUS, CENTRE, LABEL_ANGLE, LABEL_CLEAR } from '../radar'

describe('radar', () => {
  it('maps distance tags to rings', () => {
    expect(bucketIndex('<1 km')).toBe(0)
    expect(bucketIndex('10+ km')).toBe(4)
    expect(bucketIndex(undefined)).toBe(-1)
    expect(bucketIndex('far')).toBe(-1)
  })

  it('puts each dot on its own circle, inside the radar', () => {
    for (let bucket = 0; bucket < 5; bucket++) {
      for (let id = 1; id <= 50; id++) {
        const { x, y } = radarPoint('item', id, bucket)
        expect(Math.hypot(x - CENTRE, y - CENTRE)).toBeCloseTo(dotRadius(bucket), 5)
        expect(Math.hypot(x - CENTRE, y - CENTRE)).toBeLessThan(RING_RADIUS[4])
        expect(Math.hypot(x - CENTRE, y - CENTRE)).toBeGreaterThan(bucket === 0 ? 4 : RING_RADIUS[bucket - 1])
      }
    }
  })

  it('keeps a post in the same spot, and spreads posts out', () => {
    expect(radarPoint('task', 7, 2)).toEqual(radarPoint('task', 7, 2))
    expect(radarPoint('task', 7, 2)).not.toEqual(radarPoint('item', 7, 2))
    const spots = new Set(Array.from({ length: 30 }, (_, i) => JSON.stringify(radarPoint('task', i, 1))))
    expect(spots.size).toBeGreaterThan(25)
  })

  it('keeps the arc around each ring label clear of dots', () => {
    for (let bucket = 0; bucket < 5; bucket++) {
      for (let id = 1; id <= 300; id++) {
        const { x, y } = radarPoint('task', id, bucket)
        // clockwise angle from 12 o'clock
        const deg = ((Math.atan2(x - CENTRE, CENTRE - y) * 180) / Math.PI + 360) % 360
        expect(Math.abs(deg - LABEL_ANGLE)).toBeGreaterThanOrEqual(LABEL_CLEAR - 1e-6)
      }
    }
  })

  it('places ring labels inside the radar', () => {
    for (let ring = 0; ring <= UNKNOWN_RING; ring++) {
      const { x, y } = labelPoint(ring)
      expect(Math.hypot(x - CENTRE, y - CENTRE)).toBeLessThan(ringRadius(ring))
    }
  })

  it('spaces dots that share a ring so they never overlap', () => {
    // up to 8 posts on the smallest circle, dots 3.2 units wide
    const posts = Array.from({ length: 8 }, (_, i) => ({ kind: (['task', 'item', 'request'] as const)[i % 3], id: i + 1, bucket: 0 }))
    const spots = [...layoutDots(posts).spots.values()]
    expect(spots).toHaveLength(8)
    for (let a = 0; a < spots.length; a++) {
      for (let b = a + 1; b < spots.length; b++) {
        const d = Math.hypot(spots[a].x - spots[b].x, spots[a].y - spots[b].y)
        expect(d).toBeGreaterThan(3.2)
      }
    }
  })

  it('lays out a lone dot at its own spot, and keeps rings apart', () => {
    const one = layoutDots([{ kind: 'task', id: 7, bucket: 2 }])
    expect(one.spots.get('task-7')).toEqual(radarPoint('task', 7, 2))
    const two = layoutDots([{ kind: 'task', id: 1, bucket: 0 }, { kind: 'item', id: 1, bucket: 4 }])
    expect(two.spots.size).toBe(2)
  })

  it('clamps an out-of-range bucket to the nearest ring', () => {
    expect(radarPoint('request', 1, 9)).toEqual(radarPoint('request', 1, 4))
  })

  it('summarises counts in words', () => {
    expect(countsSentence({ task: 3, item: 1, request: 2 })).toBe('3 gigs, 1 marketplace item and 2 requests')
    expect(countsSentence({ task: 0, item: 2, request: 1 })).toBe('2 marketplace items and 1 request')
    expect(countsSentence({ task: 1, item: 0, request: 0 })).toBe('1 gig')
    expect(countsSentence({ task: 0, item: 0, request: 0 })).toBe('nothing yet')
  })

  it('puts posts without a location on the edge ring', () => {
    const { x, y } = radarPoint('item', 4, -1)
    const r = Math.hypot(x - CENTRE, y - CENTRE)
    expect(r).toBeCloseTo(dotRadius(UNKNOWN_RING), 5)
    expect(r).toBeGreaterThan(RING_RADIUS[4])
    expect(r).toBeLessThan(EDGE_RADIUS)
  })

  it('draws only as many dots as fit on a busy ring, nearest first, and counts the rest', () => {
    const capacity = ringCapacity(0)
    const posts = Array.from({ length: capacity + 12 }, (_, i) => ({ kind: 'task' as const, id: i + 1, bucket: 0 }))
    const layout = layoutDots(posts)
    expect(layout.spots.size).toBe(capacity)
    expect(layout.hidden.get(0)).toBe(12)
    // the first posts given (nearest) are the ones drawn
    expect(layout.spots.has('task-1')).toBe(true)
    expect(layout.spots.has(`task-${capacity + 12}`)).toBe(false)
    const spots = [...layout.spots.values()]
    for (let a = 0; a < spots.length; a++) {
      for (let b = a + 1; b < spots.length; b++) {
        expect(Math.hypot(spots[a].x - spots[b].x, spots[a].y - spots[b].y)).toBeGreaterThan(3.2)
      }
    }
  })

  it('fits more dots on outer rings', () => {
    expect(ringCapacity(0)).toBeGreaterThanOrEqual(6)
    expect(ringCapacity(4)).toBeGreaterThan(ringCapacity(0) * 4)
    expect(ringCapacity(UNKNOWN_RING)).toBeGreaterThan(ringCapacity(4))
  })

  it('always fits the 15 dots the radar can show on any one ring', () => {
    for (let ring = 0; ring <= UNKNOWN_RING; ring++) {
      expect(ringCapacity(ring)).toBeGreaterThanOrEqual(15)
    }
    const posts = Array.from({ length: 15 }, (_, i) => ({ kind: (['task', 'item', 'request'] as const)[i % 3], id: i + 1, bucket: 0 }))
    const layout = layoutDots(posts)
    expect(layout.spots.size).toBe(15)
    expect(layout.hidden.size).toBe(0)
    const spots = [...layout.spots.values()]
    for (let a = 0; a < spots.length; a++) {
      for (let b = a + 1; b < spots.length; b++) {
        expect(Math.hypot(spots[a].x - spots[b].x, spots[a].y - spots[b].y)).toBeGreaterThan(3.2)
      }
      // inside the <1 km band
      const r = Math.hypot(spots[a].x - CENTRE, spots[a].y - CENTRE)
      expect(r).toBeGreaterThan(4)
      expect(r).toBeLessThan(RING_RADIUS[0])
    }
  })
})
