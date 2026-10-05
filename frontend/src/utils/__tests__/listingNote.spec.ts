import { describe, it, expect } from 'vitest'
import { listingPriceLabel, formatUGX, offersLabel } from '../listingNote'

describe('listingNote', () => {
  it('formats shillings without decimals', () => {
    expect(formatUGX(150000)).toBe('UGX 150,000')
  })

  it('shows no price for a note-only listing', () => {
    expect(listingPriceLabel({ price_type: 'fixed', fixed_price: 0 })).toBe('')
    expect(listingPriceLabel({})).toBe('')
  })

  it('shows a fixed price or the current bid on older listings', () => {
    expect(listingPriceLabel({ price_type: 'fixed', fixed_price: 50000 })).toBe('UGX 50,000')
    expect(listingPriceLabel({ price_type: 'bidding', starting_bid: 10000 })).toBe('Bid UGX 10,000')
    expect(listingPriceLabel({ price_type: 'bidding', starting_bid: 10000, current_bid: 12000 })).toBe('Bid UGX 12,000')
  })

  it('counts offers on a request', () => {
    expect(offersLabel(1)).toBe('1 offer')
    expect(offersLabel(3)).toBe('3 offers')
  })
})
