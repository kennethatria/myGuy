import { describe, it, expect } from 'vitest'
import { hasDistances } from '../distance'

describe('hasDistances', () => {
  it('is true when any note on the page has a distance', () => {
    expect(hasDistances([{}, { distance: '~2 km' }])).toBe(true)
  })

  it('is false when none do (no viewer location, or the server fell back)', () => {
    expect(hasDistances([{}, { distance: '' }])).toBe(false)
    expect(hasDistances([])).toBe(false)
  })
})
