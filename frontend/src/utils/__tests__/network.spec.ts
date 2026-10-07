import { describe, it, expect } from 'vitest'
import { connectionsFrom, interactionsFor, layoutNetwork, formatRating, COLUMNS, YOU, type Interaction, type Connection } from '../network'

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

const person = (userId: number): Connection => ({
  userId, deals: 1, interactions: [], ratingOfYou: null, yourRating: null, averageRating: 4,
  via: 'gig', latest: '2026-10-01T10:00:00Z'
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
    expect(ann.averageRating).toBe(4.7)
    expect(ann.via).toBe('both')
    expect(ann.interactions[0].at).toBe('2026-10-02T10:00:00Z')
  })

  it('leaves a rating out until it is given', () => {
    const [ann] = connectionsFrom([review({ direction: 'given', rating: 3 })])
    expect(ann.ratingOfYou).toBeNull()
    expect(ann.yourRating).toBe(3)
    expect(ann.averageRating).toBe(3)
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

  it("turns someone's ratings into interactions with each other person", () => {
    const interactions = interactionsFor(5, [
      { task_id: 1, reviewer_id: 5, reviewed_user_id: 6, rating: 4, created_at: '2026-10-01T00:00:00Z' },
      { task_id: 1, reviewer_id: 6, reviewed_user_id: 5, rating: 5, created_at: '2026-10-01T00:00:00Z', comment: 'Great', task: { title: 'Fix sink' } }
    ], [
      { booking_id: 9, rater_id: 7, rated_id: 5, rating: 3, rated_at: '2026-10-02T00:00:00Z' }
    ])

    expect(interactions.map(i => [i.otherId, i.direction, i.rating, i.via, i.deal])).toEqual([
      [6, 'given', 4, 'gig', 'gig-1'],
      [6, 'received', 5, 'gig', 'gig-1'],
      [7, 'received', 3, 'item', 'item-9']
    ])
    // Someone else's network comes without comments or titles
    expect(interactions[0].comment).toBe('')
    expect(interactions[2].title).toBe('A marketplace item')
    expect(interactions[1].comment).toBe('Great')
  })

  it('shows ratings with one decimal, whole numbers plain', () => {
    expect(formatRating(4.2)).toBe('4.2')
    expect(formatRating(3)).toBe('3')
    expect(formatRating(4.25)).toBe('4.3')
  })

  it('hangs the first row from you, and each later person from the one above', () => {
    const { placed, height } = layoutNetwork([1, 2, 3, 4, 5].map(person))

    // First row: three columns, every line starting at you
    expect(placed.slice(0, COLUMNS).map(n => n.from)).toEqual([YOU, YOU, YOU].map(p => ({ x: p.x, y: p.y })))
    expect(new Set(placed.slice(0, COLUMNS).map(n => n.y)).size).toBe(1)
    // Second row: below the first, each line from the person above
    expect(placed[3].x).toBe(placed[0].x)
    expect(placed[3].from).toEqual({ x: placed[0].x, y: placed[0].y })
    expect(placed[4].from).toEqual({ x: placed[1].x, y: placed[1].y })
    expect(placed[3].y).toBeGreaterThan(placed[0].y)
    // Labels halfway along their lines; the drawing is tall enough for every row
    expect(placed[3].label).toEqual({ x: placed[0].x, y: (placed[0].y + placed[3].y) / 2 })
    expect(height).toBeGreaterThan(placed[4].y)
  })

  it('draws three whole background circles, centred in the graph and inside it', () => {
    const { height, ringCentre, rings } = layoutNetwork([1, 2, 3, 4].map(person))
    expect(ringCentre).toEqual({ x: 50, y: height / 2 })
    expect(rings).toHaveLength(3)
    expect(rings[0]).toBeLessThan(rings[1])
    const outer = rings[2]
    expect(ringCentre.y - outer).toBeGreaterThanOrEqual(0)
    expect(ringCentre.y + outer).toBeLessThanOrEqual(height)
    expect(ringCentre.x + outer).toBeLessThanOrEqual(100)
  })

  it('is as short as one row with nobody in it', () => {
    expect(layoutNetwork([]).placed).toEqual([])
  })
})
