import { describe, it, expect, beforeEach } from 'vitest'
import { readCachedLocation, cacheLocation, nearParam } from '../useViewerLocation'

describe('viewer location cache', () => {
  beforeEach(() => sessionStorage.clear())

  it('keeps only the cell, for 30 minutes', () => {
    const now = 1_000_000
    cacheLocation({ lat: 0.35, lng: 32.585 }, now)
    expect(readCachedLocation(now + 29 * 60 * 1000)).toEqual({ lat: 0.35, lng: 32.585 })
    expect(readCachedLocation(now + 31 * 60 * 1000)).toBeNull()
  })

  it('ignores missing or broken entries', () => {
    expect(readCachedLocation()).toBeNull()
    sessionStorage.setItem('myguy:rough-location', 'not json')
    expect(readCachedLocation()).toBeNull()
    sessionStorage.setItem('myguy:rough-location', JSON.stringify({ lat: 'x', lng: 1, at: Date.now() }))
    expect(readCachedLocation()).toBeNull()
  })

  it('formats the near= value', () => {
    expect(nearParam({ lat: -0.34, lng: 31.73 })).toBe('-0.34,31.73')
  })
})
