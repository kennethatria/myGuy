import { describe, it, expect } from 'vitest'
import { connectionsFrom, layoutNetwork, tierOf, tierCapacity, CENTRE, TIER_RADIUS, type Interaction, type Connection } from '../network'

const review = (over: Partial<Interaction>): Interaction => ({
  otherId: 2,
  direction: 'received',
  rating: 5,
  comment: '',
  via: 'gig',
  title: 'Fix sink',
  linkId: 1,
  deal: 'gig-1',
  at: '2026-10-01T10:00:00Z',
  ...over
})

const person = (userId: number, deals: number): Connection => ({
  userId, deals, interactions: [], ratingOfYou: null, yourRating: null, via: 'gig', latest: '2026-10-01T10:00:00Z'
})

describe('network', () => {
  it('groups reviews by person, counting both reviews of one deal once', () => {
    const [ann] = connectionsFrom([
      review({ direction: 'received', rating: 4 }),
      review({ direction: 'given', rating: 5, at: '2026-10-02T10:00:00Z' }),
      review({ direction: 'received', rating: 5, via: 'item', deal: 'item-9', linkId: 3 })
    ])

    expect(ann.userId).toBe(2)
    expect(ann.deals).toBe(2)
    expect(ann.ratingOfYou).toBe(4.5)
    expect(ann.yourRating).toBe(5)
    expect(ann.via).toBe('both')
    expect(ann.interactions[0].at).toBe('2026-10-02T10:00:00Z')
    expect(ann.latest).toBe('2026-10-02T10:00:00Z')
  })

  it('leaves a rating out until it is given', () => {
    const [ann] = connectionsFrom([review({ direction: 'given', rating: 3 })])
    expect(ann.ratingOfYou).toBeNull()
    expect(ann.yourRating).toBe(3)
    expect(ann.via).toBe('gig')
  })

  it('lists people with more deals first, then the most recent', () => {
    const order = connectionsFrom([
      review({ otherId: 2, deal: 'gig-1', at: '2026-10-05T00:00:00Z' }),
      review({ otherId: 3, deal: 'gig-2', at: '2026-10-01T00:00:00Z' }),
      review({ otherId: 3, deal: 'gig-3', at: '2026-10-02T00:00:00Z' }),
      review({ otherId: 4, deal: 'gig-4', at: '2026-10-06T00:00:00Z' })
    ]).map(c => c.userId)
    expect(order).toEqual([3, 4, 2])
  })

  it('puts more deals on an inner ring', () => {
    expect(tierOf(1)).toBe(2)
    expect(tierOf(2)).toBe(1)
    expect(tierOf(3)).toBe(0)
    expect(tierOf(10)).toBe(0)
  })

  it('places each person on their ring, apart from each other', () => {
    const { placed, hidden } = layoutNetwork([person(1, 3), person(2, 2), person(3, 1), person(4, 1)])
    expect(hidden).toBe(0)
    for (const node of placed) {
      expect(Math.hypot(node.x - CENTRE, node.y - CENTRE)).toBeCloseTo(TIER_RADIUS[tierOf(node.connection.deals)], 5)
    }
    const [a, b] = placed.filter(n => n.tier === 2)
    expect(Math.hypot(a.x - b.x, a.y - b.y)).toBeGreaterThan(8)
  })

  it('passes a full ring outward and counts who does not fit', () => {
    const inner = Array.from({ length: tierCapacity(0) + 2 }, (_, i) => person(i + 1, 5))
    const { placed } = layoutNetwork(inner)
    expect(placed.filter(n => n.tier === 0)).toHaveLength(tierCapacity(0))
    expect(placed.filter(n => n.tier === 1)).toHaveLength(2)

    const crowd = Array.from({ length: tierCapacity(2) + 4 }, (_, i) => person(i + 1, 1))
    expect(layoutNetwork(crowd).hidden).toBe(4)
  })
})
