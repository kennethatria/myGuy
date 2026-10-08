import { describe, it, expect } from 'vitest'
import { countWords, headlineFits, bodyFits, noteColor, noteTilt, expiryLabel, timeLeft, noteMeta, postedMeta, noteFreshness } from '../gigNote'

describe('gigNote', () => {
  it('counts words like the backend', () => {
    expect(countWords('')).toBe(0)
    expect(countWords('   ')).toBe(0)
    expect(countWords('  Paint   my\nfence ')).toBe(3)
  })

  it('applies the headline and body limits', () => {
    expect(headlineFits('one two three four five')).toBe(true)
    expect(headlineFits('one two three four five six')).toBe(false)
    expect(headlineFits('a'.repeat(61))).toBe(false)
    expect(bodyFits(Array(20).fill('word').join(' '))).toBe(true)
    expect(bodyFits(Array(21).fill('word').join(' '))).toBe(false)
  })

  it('gives each gig a stable colour and small tilt', () => {
    expect(noteColor(7)).toBe(noteColor(7))
    expect(noteColor(1)).not.toBe(noteColor(2))
    for (let id = 0; id < 20; id++) {
      expect(Math.abs(noteTilt(id))).toBeLessThanOrEqual(0.6 + 1e-9)
    }
  })

  it('shows time left, and nothing once the deadline has passed', () => {
    const now = new Date('2026-10-05T10:00:00Z')
    expect(expiryLabel('2026-10-06T09:30:00Z', now)).toBe('Expires in 23h')
    expect(expiryLabel('2026-10-05T10:20:00Z', now)).toBe('Expires in 20m')
    expect(expiryLabel('2026-10-05T09:59:00Z', now)).toBe('')
    expect(expiryLabel('not a date', now)).toBe('')
  })

  it('shows short time left for note rows', () => {
    const now = new Date('2026-10-05T10:00:00Z')
    expect(timeLeft('2026-10-06T09:30:00Z', now)).toBe('23h')
    expect(timeLeft('2026-10-05T10:20:00Z', now)).toBe('20m')
    expect(timeLeft('2026-10-05T09:59:00Z', now)).toBe('')
    expect(timeLeft('not a date', now)).toBe('')
  })

  it('writes a note footer from what is known', () => {
    const now = new Date('2026-10-05T10:00:00Z')
    const deadline = '2026-10-06T09:30:00Z'
    expect(noteMeta({ distance: '~2 km', deadline }, 'ann', true, now)).toBe('~2 km · @ann · 23h')
    expect(noteMeta({ deadline }, 'ann', true, now)).toBe('No location · @ann · 23h')
    expect(noteMeta({ deadline }, 'ann', false, now)).toBe('@ann · 23h')
    expect(noteMeta({ distance: '<1 km', deadline: '2026-10-05T09:00:00Z' }, '', false, now)).toBe('<1 km')
  })

  it('writes when a post went up and, while live, when it comes down', () => {
    const now = new Date('2026-10-05T10:00:00Z')
    expect(postedMeta('2026-10-05T08:00:00Z', '2026-10-06T08:00:00Z', true, now)).toBe('Posted 2 h ago · Expires in 22h')
    expect(postedMeta('2026-10-05T10:00:00Z', '2026-10-06T10:00:00Z', false, now)).toBe('Posted just now')
    expect(postedMeta('2026-10-05T08:00:00Z', undefined, true, now)).toBe('Posted 2 h ago')
  })

  it('fades a note as it nears expiry, never below 0.82', () => {
    const now = new Date('2026-10-05T10:00:00Z')
    const inHours = (h: number) => new Date(now.getTime() + h * 3_600_000).toISOString()
    expect(noteFreshness(inHours(24), now)).toBe(1)
    expect(noteFreshness(inHours(23), now)).toBe(1)
    expect(noteFreshness(inHours(13), now)).toBe(0.91)
    expect(noteFreshness(inHours(3), now)).toBe(0.82)
    expect(noteFreshness(inHours(-1), now)).toBe(0.82)
    expect(noteFreshness('not a date', now)).toBe(1)
  })
})
