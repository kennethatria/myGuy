import { describe, it, expect } from 'vitest'
import { connectionsFrom, interactionsFor, recentNetwork, layoutNetwork, networkSize, tierOf, formatRating, NODE_RADIUS, CHILD_RADIUS, WIDTH, HEIGHT, type Interaction, type Connection } from '../network'

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
  it('keeps the tree to 15 people: the most recent connections, then their most recent', () => {
    const at = (userId: number, day: number) => ({ ...person(userId), latest: `2026-10-${String(day).padStart(2, '0')}T10:00:00Z` })
    const branches = [
      { connection: at(2, 1), children: [at(20, 28), at(21, 2)] },
      ...Array.from({ length: 12 }, (_, i) => ({ connection: at(3 + i, 10 + i), children: [] })),
      { connection: at(30, 5), children: [at(31, 27), at(32, 3)] }
    ]

    const kept = recentNetwork(branches)

    expect(kept.length).toBe(14)
    expect(kept[0].connection.userId).toBe(14)
    expect(kept.reduce((n, b) => n + 1 + b.children.length, 0)).toBe(15)
    expect(kept.find(b => b.connection.userId === 2)!.children.map(c => c.userId)).toEqual([20])
    expect(kept.find(b => b.connection.userId === 30)!.children).toEqual([])

    const many = Array.from({ length: 20 }, (_, i) => ({ connection: at(40 + i, i + 1), children: [at(90, 28)] }))
    const top = recentNetwork(many)
    expect(top.length).toBe(15)
    expect(top.every(b => b.children.length === 0)).toBe(true)
    expect(top[top.length - 1].connection.userId).toBe(45)
  })

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

  it('puts connections on the inner ring and theirs beyond them on the outer ring', () => {
    const { placed, centre, rings } = layoutNetwork([
      { connection: person(2), children: [person(5), person(6)] },
      { connection: person(3), children: [] }
    ], 1)

    expect(placed.map(n => [n.connection.userId, n.level, n.parentId])).toEqual([
      [2, 1, 1], [5, 2, 2], [6, 2, 2], [3, 1, 1]
    ])
    expect(rings).toHaveLength(2)
    // True circles, centred in the square, as on the home radar
    expect(centre).toEqual({ x: WIDTH / 2, y: HEIGHT / 2 })
    for (const n of placed) expect(Math.hypot(n.x - centre.x, n.y - centre.y)).toBeCloseTo(rings[n.level - 1])
    expect(rings[1]).toBeGreaterThan(rings[0])
    expect(placed[0].r).toBe(NODE_RADIUS)
    expect(placed[1].r).toBe(CHILD_RADIUS)
    // The first connection is at the top
    expect(placed[0].x).toBeCloseTo(centre.x)
    expect(placed[0].y).toBeLessThan(centre.y)
    // Each line runs from whom they're connected to, labelled halfway
    expect(placed[0].from).toEqual(centre)
    expect(placed[1].from).toEqual({ x: placed[0].x, y: placed[0].y })
    expect(placed[3].from).toEqual(centre)
    expect(placed[1].label).toEqual({ x: (placed[0].x + placed[1].x) / 2, y: (placed[0].y + placed[1].y) / 2 })
    // Everything fits the picture
    for (const n of placed) {
      expect(n.x - n.r).toBeGreaterThan(0)
      expect(n.x + n.r).toBeLessThan(WIDTH)
      expect(n.y - n.r).toBeGreaterThan(0)
      expect(n.y + n.r).toBeLessThan(HEIGHT)
    }
  })

  it("keeps someone's connections in their own slice, so lines don't cross", () => {
    const { placed, centre } = layoutNetwork([
      { connection: person(2), children: [person(5), person(6)] },
      { connection: person(3), children: [person(7)] },
      { connection: person(4), children: [] }
    ], 1)
    // 2 has half the people under it, so its slice runs from the left (9
    // o'clock) round over the top; measure clockwise from there
    const angle = (n: { x: number; y: number }) => Math.atan2(n.y - centre.y, n.x - centre.x) + Math.PI
    // 2's slice (5, 6), then 3's (7), then 4
    const order = [...placed].sort((a, b) => angle(a) - angle(b)).map(n => n.connection.userId)
    expect(order.indexOf(5)).toBeLessThan(order.indexOf(7))
    expect(order.indexOf(6)).toBeLessThan(order.indexOf(3))
    expect(order.indexOf(7)).toBeLessThan(order.indexOf(4))
  })

  it('needs only the inner ring when nobody has connections of their own', () => {
    const one = layoutNetwork([{ connection: person(2), children: [] }], 1)
    expect(one.rings).toHaveLength(1)
    // ...and takes more of the room
    expect(one.rings[0]).toBeGreaterThan(layoutNetwork([{ connection: person(2), children: [person(3)] }], 1).rings[0])
    expect(layoutNetwork([]).placed).toEqual([])
    expect(layoutNetwork([]).rings).toEqual([])
  })

  it('counts each person in the network once, the centre person left out', () => {
    expect(networkSize([
      { connection: person(2), children: [person(5), person(3)] },
      { connection: person(3), children: [person(1), person(5)] }
    ], 1)).toBe(3)
  })

  it('grades a link by its average rating', () => {
    expect([5, 4.5, 4.4, 3, 2.9, 1].map(tierOf)).toEqual(['strong', 'strong', 'fair', 'fair', 'weak', 'weak'])
  })
})
