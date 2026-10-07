import { describe, it, expect } from 'vitest'
import { connectionsFrom, interactionsFor, recentNetwork, layoutNetwork, branchPath, formatRating, YOU, NODE_RADIUS, CHILD_RADIUS, type Interaction, type Connection } from '../network'

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

  it('lays out a branching tree, one person per row, children indented under their parent', () => {
    const { placed, height } = layoutNetwork([
      { connection: person(2), children: [person(5), person(6)] },
      { connection: person(3), children: [] }
    ], 1)

    expect(placed.map(n => [n.connection.userId, n.level, n.parentId])).toEqual([
      [2, 1, 1], [5, 2, 2], [6, 2, 2], [3, 1, 1]
    ])
    // One row each, top to bottom
    const ys = placed.map(n => n.y)
    expect([...ys].sort((a, b) => a - b)).toEqual(ys)
    expect(new Set(ys).size).toBe(4)
    // Children are indented further than their parent, and smaller
    expect(placed[1].x).toBeGreaterThan(placed[0].x)
    expect(placed[1].r).toBe(CHILD_RADIUS)
    expect(placed[0].r).toBe(NODE_RADIUS)
    // Each line starts at its parent: the centre, or the person above
    expect(placed[0].from).toEqual({ x: YOU.x, y: YOU.y })
    expect(placed[1].from).toEqual({ x: placed[0].x, y: placed[0].y })
    expect(placed[3].from).toEqual({ x: YOU.x, y: YOU.y })
    // The rating sits on the line's horizontal part, in its own row
    expect(placed[1].label.y).toBe(placed[1].y)
    expect(placed[1].label.x).toBeGreaterThan(placed[0].x)
    expect(placed[1].label.x).toBeLessThan(placed[1].x)
    expect(height).toBeGreaterThan(placed[3].y)
  })

  it('curves each branch from its parent round to just before the person', () => {
    const [node] = layoutNetwork([{ connection: person(2), children: [] }], 1).placed
    const path = branchPath(node)
    expect(path.startsWith(`M ${YOU.x} ${YOU.y + NODE_RADIUS}`)).toBe(true)
    expect(path).toContain(' Q ')
    expect(path.trim().endsWith(`${node.x - node.r} ${node.y}`)).toBe(true)
    // The rating sits on the straight run across, clear of both nodes
    expect(node.label.x - 6).toBeGreaterThan(YOU.x)
    expect(node.label.x + 6).toBeLessThan(node.x - node.r)
  })

  it('is as short as one row with nobody in it', () => {
    expect(layoutNetwork([]).placed).toEqual([])
  })
})
