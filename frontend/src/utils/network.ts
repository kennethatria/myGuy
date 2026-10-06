// Geometry and grouping for "your network" on the Reviews page: you in the
// middle, one dot per person you reviewed or who reviewed you. People you have
// done more deals with (gigs or marketplace sales) sit on an inner ring.

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
      via: vias.size > 1 ? 'both' : list[0].via,
      latest: list[0].at
    }
  })

  return connections.sort((a, b) => b.deals - a.deals || b.latest.localeCompare(a.latest))
}

/** The SVG is drawn in a 100 x 100 box centred on (50, 50). */
export const CENTRE = 50

/** Ring radii, innermost first: 3+ deals together, 2 deals, 1 deal. */
export const TIER_RADIUS = [17, 29, 41] as const

/** Space each dot needs along its ring so neighbours never touch. */
export const NODE_SPACING = 8

/** The ring a connection belongs on, by how many deals you did together. */
export function tierOf(deals: number): number {
  return deals >= 3 ? 0 : deals === 2 ? 1 : 2
}

/** How many dots fit round a ring. */
export function tierCapacity(tier: number): number {
  return Math.floor((2 * Math.PI * TIER_RADIUS[tier]) / NODE_SPACING)
}

export interface PlacedConnection {
  connection: Connection
  tier: number
  x: number
  y: number
}

/**
 * Places connections on their rings, spread evenly from 12 o'clock (each ring
 * turned a little so dots don't line up). A full ring passes the rest outward;
 * whoever doesn't fit on the outer ring is counted in `hidden`.
 */
export function layoutNetwork(connections: Connection[]): { placed: PlacedConnection[]; hidden: number } {
  const rings: Connection[][] = TIER_RADIUS.map(() => [])
  let hidden = 0
  for (const connection of connections) {
    let tier = tierOf(connection.deals)
    while (tier < TIER_RADIUS.length && rings[tier].length >= tierCapacity(tier)) tier++
    if (tier < TIER_RADIUS.length) rings[tier].push(connection)
    else hidden++
  }

  const placed = rings.flatMap((ring, tier) =>
    ring.map((connection, i) => {
      const degrees = -90 + tier * 25 + (360 / ring.length) * i
      const radians = (degrees * Math.PI) / 180
      return {
        connection,
        tier,
        x: CENTRE + TIER_RADIUS[tier] * Math.cos(radians),
        y: CENTRE + TIER_RADIUS[tier] * Math.sin(radians)
      }
    })
  )
  return { placed, hidden }
}
