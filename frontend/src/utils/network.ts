// Grouping and layout for "your network" on the Network page: you in the
// centre, the people you reviewed or who reviewed you on a ring round you, and
// the people they're connected to on an outer ring. Each line is coloured by
// the average rating between the two people it joins.

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

/** The SVG is 100 units wide; its height depends on how many rings it needs. */
export const WIDTH = 100
/** The centre person's circle */
export const CENTRE_RADIUS = 8
/** Their connections, and their connections' connections (a little smaller) */
export const NODE_RADIUS = 5.5
export const CHILD_RADIUS = 4.2
/** The two rings, taller than wide so the picture suits a phone */
const RINGS = [{ rx: 23, ry: 28 }, { rx: 41, ry: 50 }] as const
/** Room round the outer ring for its nodes */
const MARGIN = 7

/** How strong a link is, from the average rating between the two people */
export type Tier = 'strong' | 'fair' | 'weak'

/** Strong 4.5 and up, fair 3 to 4.4, weak below 3 */
export function tierOf(rating: number): Tier {
  return rating >= 4.5 ? 'strong' : rating >= 3 ? 'fair' : 'weak'
}

/** One of your connections, and the people they are connected to in turn */
export interface Branch {
  connection: Connection
  children: Connection[]
}

/** The most people the picture shows, so it stays a size you can take in on a phone */
export const MAX_PEOPLE = 15

const newestFirst = (a: Connection, b: Connection) => b.latest.localeCompare(a.latest)

/**
 * Keeps the network to max people in all: the centre person's most recent
 * connections first, then, in the spots left, the most recent of the people
 * those are connected to. Each branch keeps its own newest first.
 */
export function recentNetwork(branches: Branch[], max = MAX_PEOPLE): Branch[] {
  const kept = [...branches].sort((a, b) => newestFirst(a.connection, b.connection)).slice(0, max)
  const spare = max - kept.length
  const children = kept
    .flatMap(branch => branch.children.map(child => ({ branch, child })))
    .sort((a, b) => newestFirst(a.child, b.child))
    .slice(0, spare)
  return kept.map(branch => ({
    connection: branch.connection,
    children: children.filter(c => c.branch === branch).map(c => c.child)
  }))
}

export interface PlacedConnection {
  connection: Connection
  /** 1: connected to the centre person (inner ring); 2: to one of those (outer ring) */
  level: 1 | 2
  x: number
  y: number
  r: number
  /** Who it is connected to here, and where their node is: the line runs from there */
  parentId: number
  from: { x: number; y: number }
  /** Where the line's rating label sits: halfway along it */
  label: { x: number; y: number }
}

export interface Ring {
  rx: number
  ry: number
}

export interface NetworkLayout {
  placed: PlacedConnection[]
  centre: { x: number; y: number }
  /** The rings in use, drawn as dashed guides */
  rings: Ring[]
  height: number
}

/**
 * Lays the network out on rings round the centre person. Each connection gets
 * a slice of the circle as wide as the people under it (at least one), sits on
 * the inner ring in the middle of its slice, and its own connections share
 * that slice on the outer ring, so they sit beyond it and lines never cross.
 * The first connection is at the top.
 */
export function layoutNetwork(branches: Branch[], centreId = 0): NetworkLayout {
  const hasOuter = branches.some(b => b.children.length > 0)
  const rings: Ring[] = branches.length ? [...RINGS.slice(0, hasOuter ? 2 : 1)] : []
  const outer = rings[rings.length - 1] ?? { rx: 0, ry: 0 }
  const centre = { x: WIDTH / 2, y: outer.ry + MARGIN + (rings.length ? 0 : CENTRE_RADIUS) }
  const at = (ring: Ring, angle: number) => ({
    x: centre.x + ring.rx * Math.cos(angle),
    y: centre.y + ring.ry * Math.sin(angle)
  })
  const node = (connection: Connection, level: 1 | 2, parentId: number, from: { x: number; y: number }, spot: { x: number; y: number }): PlacedConnection => ({
    connection, level, ...spot, r: level === 1 ? NODE_RADIUS : CHILD_RADIUS, parentId, from,
    label: { x: (from.x + spot.x) / 2, y: (from.y + spot.y) / 2 }
  })

  const weights = branches.map(b => Math.max(1, b.children.length))
  const total = weights.reduce((sum, w) => sum + w, 0)
  const placed: PlacedConnection[] = []
  let start = -Math.PI / 2 - (total ? (weights[0] / total) * Math.PI : 0)
  branches.forEach((branch, i) => {
    const span = (2 * Math.PI * weights[i]) / total
    const parent = node(branch.connection, 1, centreId, centre, at(RINGS[0], start + span / 2))
    placed.push(parent)
    const step = span / branch.children.length
    branch.children.forEach((child, j) => {
      placed.push(node(child, 2, branch.connection.userId, { x: parent.x, y: parent.y }, at(RINGS[1], start + (j + 0.5) * step)))
    })
    start += span
  })
  return { placed, centre, rings, height: centre.y * 2 }
}

/** How many different people are in a network, the centre person left out. */
export function networkSize(branches: Branch[], centreId: number): number {
  const ids = new Set<number>()
  for (const branch of branches) {
    ids.add(branch.connection.userId)
    for (const child of branch.children) ids.add(child.userId)
  }
  ids.delete(centreId)
  return ids.size
}
