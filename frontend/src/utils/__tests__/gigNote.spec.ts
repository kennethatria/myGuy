import { describe, it, expect } from 'vitest'
import { countWords, headlineFits, bodyFits, noteColor, noteTilt, expiryLabel, timeLeft } from '../gigNote'

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
})
