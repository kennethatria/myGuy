// Marketplace listings are sticky notes too, with the gig limits
// (store-service internal/services/store_service.go mirrors them).
// A price is optional: most listings put it in the note or agree it in chat.

export interface ListingPrice {
  price_type?: string
  fixed_price?: number
  starting_bid?: number
  current_bid?: number
}

export function formatUGX(amount: number): string {
  return `UGX ${new Intl.NumberFormat('en-UG', { maximumFractionDigits: 0 }).format(amount)}`
}

/** The price to show on a note, or '' when the listing has none. */
export function listingPriceLabel(item: ListingPrice): string {
  if (item.price_type === 'bidding') {
    const bid = item.current_bid || item.starting_bid || 0
    return bid > 0 ? `Bid ${formatUGX(bid)}` : ''
  }
  return item.fixed_price && item.fixed_price > 0 ? formatUGX(item.fixed_price) : ''
}

/** "1 offer", "3 offers": listings sellers made for a request. */
export function offersLabel(count: number): string {
  return count === 1 ? '1 offer' : `${count} offers`
}
