import { describe, it, expect } from 'vitest'
import { offersLabel } from '../listingNote'

describe('listingNote', () => {
  it('counts offers on a request', () => {
    expect(offersLabel(1)).toBe('1 offer')
    expect(offersLabel(3)).toBe('3 offers')
  })
})
