// Grouping and layout for "your network" on the Reviews page: you at the top,
// one node per person you reviewed or who reviewed you, below you in columns,
// most deals first. Each line is labelled with the average rating between you.

/** Where a review came from: a gig, or a marketplace sale. */
export type Via = 'gig' | 'item'

/** One review between you and someone else, either way round. */
export interface Interaction {
  otherId: number
  direction: 'given' | 'received'
  rating: number
  comment: string
  via: Via
  title: string
  /** The gig's or item's id, for a link to it */
  linkId: number
  /** Identifies the deal: both reviews of one gig or sale share it */
  deal: string
  at: string
}

/** Everyone you are connected to, with what passed between you. */
export interface Connection {
  userId: number
  /** Newest first */
  interactions: Interaction[]
  deals: number
  /** Average rating they gave you, or null if they never rated you */
  ratingOfYou: number | null
  /** Average rating you gave them, or null if you never rated them */
  yourRating: number | null
  /** Average of every rating between you, both ways */
  averageRating: number
  via: Via | 'both'
  latest: string
}

function average(ratings: number[]): number | null {
  if (!ratings.length) return null
  return Math.round((ratings.reduce((sum, r) => sum + r, 0) / ratings.length) * 10) / 10
}

/** A gig rating as the APIs send it; your own come with comment and gig. */
export interface GigRating {
  task_id: number
  reviewer_id: number
  reviewed_user_id: number
  rating: number
  created_at: string
  comment?: string
  task?: { title: string }
}

/** A marketplace rating either way round; your own come with review and item. */
export interface StoreRating {
  booking_id: number
  rater_id: number
  rated_id: number
  rating: number
  rated_at: string
  review?: string
  item_id?: number
  item_title?: string
}

/**
 * Every rating between userId and someone else, as interactions with that
 * person: given when userId rated them, received when they rated userId.
 */
export function interactionsFor(userId: number, gigs: GigRating[], store: StoreRating[]): Interaction[] {
  const fromGigs = gigs.map((r): Interaction => {
    const given = r.reviewer_id === userId
    return {
      otherId: given ? r.reviewed_user_id : r.reviewer_id,
      direction: given ? 'given' : 'received',
      rating: r.rating,
      comment: r.comment ?? '',
      via: 'gig',
      title: r.task?.title || 'A gig',
      linkId: r.task_id,
      deal: `gig-${r.task_id}`,
      at: r.created_at
    }
  })
  const fromStore = store.map((r): Interaction => {
    const given = r.rater_id === userId
    return {
      otherId: given ? r.rated_id : r.rater_id,
      direction: given ? 'given' : 'received',
      rating: r.rating,
      comment: r.review ?? '',
      via: 'item',
      title: r.item_title || 'A marketplace item',
      linkId: r.item_id ?? 0,
      deal: `item-${r.booking_id}`,
      at: r.rated_at
    }
  })
  return [...fromGigs, ...fromStore].filter(i => i.otherId && i.otherId !== userId)
}

/** Groups interactions by person: most deals first, then most recent. */
export function connectionsFrom(interactions: Interaction[]): Connection[] {
  const byUser = new Map<number, Interaction[]>()
  for (const interaction of interactions) {
    const list = byUser.get(interaction.otherId) ?? []
    list.push(interaction)
    byUser.set(interaction.otherId, list)
  }

  const connections = [...byUser.entries()].map(([userId, list]): Connection => {
    list.sort((a, b) => b.at.localeCompare(a.at))
    const vias = new Set(list.map(i => i.via))
    return {
      userId,
      interactions: list,
      deals: new Set(list.map(i => i.deal)).size,
      ratingOfYou: average(list.filter(i => i.direction === 'received').map(i => i.rating)),
      yourRating: average(list.filter(i => i.direction === 'given').map(i => i.rating)),
      averageRating: average(list.map(i => i.rating)) ?? 0,
      via: vias.size > 1 ? 'both' : list[0].via,
      latest: list[0].at
    }
  })

  return connections.sort((a, b) => b.deals - a.deals || b.latest.localeCompare(a.latest))
}

/** A rating as shown on a line: one decimal, whole numbers plain ("4.2", "3") */
export function formatRating(rating: number): string {
  return Number.isInteger(rating) ? String(rating) : rating.toFixed(1)
}

/** The SVG is 100 units wide; its height grows with the rows. */
export const WIDTH = 100
/** Where the centre person sits: top left, the root of the tree */
export const YOU = { x: 9, y: 10 } as const
export const NODE_RADIUS = 5.5
/** Their connections' connections are drawn a little smaller */
export const CHILD_RADIUS = 4.5
/** How far right each level of the tree starts */
const LEVEL_X = [YOU.x, 38, 66] as const
const ROW_GAP = 17
/** Room under the last row */
const BOTTOM = 8

/** One of your connections, and the people they are connected to in turn */
export interface Branch {
  connection: Connection
  children: Connection[]
}

export interface PlacedConnection {
  connection: Connection
  /** 1: connected to the centre person; 2: to one of those */
  level: 1 | 2
  x: number
  y: number
  r: number
  /** The branch it hangs from: who it belongs to and where their node is */
  parentId: number
  from: { x: number; y: number }
  /** Where the line's rating label sits: on its horizontal part */
  label: { x: number; y: number }
}

export interface NetworkLayout {
  placed: PlacedConnection[]
  height: number
  /** Background circles (the pencil sketch): their centre and radii */
  ringCentre: { x: number; y: number }
  rings: number[]
}

/**
 * Lays out a branching tree, one person per row: the centre person at the top
 * left, each of their connections below on its own row, and each of those
 * people's connections indented beneath them. Lines run down from the parent
 * and across to the person, so they never cross.
 */
export function layoutNetwork(branches: Branch[], centreId = 0): NetworkLayout {
  const placed: PlacedConnection[] = []
  let row = 0
  const place = (connection: Connection, level: 1 | 2, parentId: number, from: { x: number; y: number }) => {
    row++
    const x = LEVEL_X[level]
    const y = YOU.y + row * ROW_GAP
    const node: PlacedConnection = {
      connection, level, x, y, r: level === 1 ? NODE_RADIUS : CHILD_RADIUS, parentId, from,
      label: { x: (from.x + x) / 2, y }
    }
    placed.push(node)
    return node
  }
  for (const branch of branches) {
    const parent = place(branch.connection, 1, centreId, { x: YOU.x, y: YOU.y })
    for (const child of branch.children) place(child, 2, branch.connection.userId, { x: parent.x, y: parent.y })
  }
  const height = YOU.y + row * ROW_GAP + BOTTOM + (row ? 0 : NODE_RADIUS)
  // Three whole circles behind the graph, centred in it and inside it
  const ringCentre = { x: WIDTH / 2, y: height / 2 }
  const outer = Math.min(WIDTH, height) / 2 - 1.5
  const rings = [outer / 3, (outer * 2) / 3, outer]
  return { placed, height, ringCentre, rings }
}

/**
 * A branch's line as a sketch: down from the parent, then across to the
 * person, with a slight hand-drawn bow in each part. `wobble` shifts the
 * second pencil stroke so the two don't overlap exactly.
 */
export function branchPath(node: PlacedConnection, wobble = 0): string {
  const { from, x, y, r } = node
  // From just under the parent's node, whose radius is NODE_RADIUS
  const top = from.y + NODE_RADIUS
  const end = x - r
  const bow = 1.1 + wobble
  return [
    `M ${from.x + wobble} ${top}`,
    // down, bowing slightly
    `Q ${from.x + bow} ${(top + y) / 2} ${from.x + wobble * 0.5} ${y}`,
    // across, bowing slightly
    `Q ${(from.x + end) / 2} ${y - bow} ${end} ${y + wobble * 0.5}`
  ].join(' ')
}
