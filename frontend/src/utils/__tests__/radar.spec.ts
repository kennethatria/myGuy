import { describe, it, expect } from 'vitest'
import { bucketIndex, radarPoint, layoutDots, countsSentence, RING_HALF, CENTRE, labelGap } from '../radar'

describe('radar', () => {
  it('maps distance tags to rings', () => {
    expect(bucketIndex('<1 km')).toBe(0)
    expect(bucketIndex('10+ km')).toBe(4)
    expect(bucketIndex(undefined)).toBe(-1)
    expect(bucketIndex('far')).toBe(-1)
  })

  it('puts each dot on its own ring, inside the square', () => {
    for (let bucket = 0; bucket < 5; bucket++) {
      for (let id = 1; id <= 50; id++) {
        const { x, y } = radarPoint('item', id, bucket)
        const half = Math.max(Math.abs(x - CENTRE), Math.abs(y - CENTRE))
        const inner = bucket === 0 ? 4 : RING_HALF[bucket - 1]
        expect(half).toBeCloseTo((inner + RING_HALF[bucket]) / 2, 5)
        expect(x).toBeGreaterThanOrEqual(0)
        expect(x).toBeLessThanOrEqual(100)
        expect(y).toBeGreaterThanOrEqual(0)
        expect(y).toBeLessThanOrEqual(100)
      }
    }
  })

  it('keeps a post in the same spot, and spreads posts out', () => {
    expect(radarPoint('task', 7, 2)).toEqual(radarPoint('task', 7, 2))
    expect(radarPoint('task', 7, 2)).not.toEqual(radarPoint('item', 7, 2))
    const spots = new Set(Array.from({ length: 30 }, (_, i) => JSON.stringify(radarPoint('task', i, 1))))
    expect(spots.size).toBeGreaterThan(25)
  })

  it('keeps each ring label corner clear of dots', () => {
    for (let bucket = 0; bucket < 5; bucket++) {
      const inner = bucket === 0 ? 4 : RING_HALF[bucket - 1]
      const half = (inner + RING_HALF[bucket]) / 2
      const top = CENTRE - half
      const right = CENTRE + half
      for (let id = 1; id <= 300; id++) {
        const { x, y } = radarPoint('task', id, bucket)
        const onTopEdge = Math.abs(y - top) < 1e-9
        expect(onTopEdge && x > right - labelGap(2 * half) + 1e-9).toBe(false)
      }
    }
  })

  it('spaces dots that share a ring so they never overlap', () => {
    // up to 8 posts on the smallest ring, dots 3.2 units wide
    const posts = Array.from({ length: 8 }, (_, i) => ({ kind: (['task', 'item', 'request'] as const)[i % 3], id: i + 1, bucket: 0 }))
    const spots = [...layoutDots(posts).values()]
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
    expect(one.get('task-7')).toEqual(radarPoint('task', 7, 2))
    const two = layoutDots([{ kind: 'task', id: 1, bucket: 0 }, { kind: 'item', id: 1, bucket: 4 }])
    expect(two.size).toBe(2)
  })

  it('clamps an out-of-range bucket to the nearest ring', () => {
    expect(radarPoint('request', 1, 9)).toEqual(radarPoint('request', 1, 4))
  })

  it('summarises counts in words', () => {
    expect(countsSentence({ task: 3, item: 1, request: 2 })).toBe('3 gigs, 1 listing and 2 requests')
    expect(countsSentence({ task: 0, item: 2, request: 1 })).toBe('2 listings and 1 request')
    expect(countsSentence({ task: 1, item: 0, request: 0 })).toBe('1 gig')
    expect(countsSentence({ task: 0, item: 0, request: 0 })).toBe('nothing yet')
  })
})
