import { describe, it, expect } from 'vitest'
import { connectionsFrom, layoutNetwork, formatRating, COLUMNS, YOU, type Interaction, type Connection } from '../network'

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

  it('is as short as one row with nobody in it', () => {
    expect(layoutNetwork([]).placed).toEqual([])
  })
})
