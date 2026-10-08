import { afterEach, describe, expect, it, vi } from 'vitest'
import { trackEvent } from '../analytics'

type UmamiWindow = Window & { umami?: { track: (event: string, data?: object) => void } }

describe('trackEvent', () => {
  afterEach(() => {
    delete (window as UmamiWindow).umami
  })

  it('reports the event to Umami', () => {
    const track = vi.fn()
    ;(window as UmamiWindow).umami = { track }
    trackEvent('item-booked', { from: 'item-page' })
    expect(track).toHaveBeenCalledWith('item-booked', { from: 'item-page' })
  })

  it('does nothing without Umami', () => {
    expect(() => trackEvent('sign-in')).not.toThrow()
  })

  it('never throws when Umami fails', () => {
    ;(window as UmamiWindow).umami = {
      track: () => {
        throw new Error('blocked')
      },
    }
    expect(() => trackEvent('gig-posted')).not.toThrow()
  })
})
