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
export const COLUMNS = 3
/** Where you sit, at the top in the middle */
export const YOU = { x: 50, y: 11 } as const
export const NODE_RADIUS = 7.5
const FIRST_ROW_Y = 50
const ROW_GAP = 36
/** Room under the last row for its names */
const BOTTOM = 18

export interface PlacedConnection {
  connection: Connection
  x: number
  y: number
  /** Where its line starts: you, or the person above it in its column */
  from: { x: number; y: number }
  /** Where the line's rating label sits: halfway along it */
  label: { x: number; y: number }
}

/**
 * Places connections in rows of COLUMNS under you, in the order given (most
 * deals first). The first row hangs from you; each person after that hangs
 * from the one above them, so lines never cross.
 */
export function layoutNetwork(connections: Connection[]): { placed: PlacedConnection[]; height: number } {
  // Fewer people than columns: spread them across the whole width
  const columns = Math.max(1, Math.min(COLUMNS, connections.length))
  const columnX = (col: number) => WIDTH / (columns * 2) * (col * 2 + 1)
  const placed = connections.map((connection, i) => {
    const row = Math.floor(i / columns)
    const col = i % columns
    const x = columnX(col)
    const y = FIRST_ROW_Y + row * ROW_GAP
    const from = row === 0 ? { x: YOU.x, y: YOU.y } : { x, y: y - ROW_GAP }
    return { connection, x, y, from, label: { x: (from.x + x) / 2, y: (from.y + y) / 2 } }
  })
  const rows = Math.ceil(connections.length / columns)
  const height = rows ? FIRST_ROW_Y + (rows - 1) * ROW_GAP + BOTTOM : FIRST_ROW_Y
  return { placed, height }
}
