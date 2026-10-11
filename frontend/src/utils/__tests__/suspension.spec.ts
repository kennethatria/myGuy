import { describe, it, expect } from 'vitest'
import { formatUntil, markText, noticeText, suspendedNow } from '../suspension'

const NOW = Date.parse('2026-10-11T12:00:00Z')
const LATER = '2026-10-14T12:00:00Z'
const EARLIER = '2026-09-02T12:00:00Z'

describe('suspension wording', () => {
  it('knows whether a suspension is still on', () => {
    expect(suspendedNow(LATER, NOW)).toBe(true)
    expect(suspendedNow(EARLIER, NOW)).toBe(false)
    expect(suspendedNow(null, NOW)).toBe(false)
  })

  it('marks a profile while suspended, then with the month it happened', () => {
    expect(markText({ until: LATER }, NOW)).toBe(`Suspended until ${formatUntil(LATER)} for breaking the site rules`)
    const month = new Date(EARLIER).toLocaleString(undefined, { month: 'long', year: 'numeric' })
    expect(markText({ ended: EARLIER }, NOW)).toBe(`Suspended in ${month} for breaking the site rules`)
    // Ran out since the profile was loaded
    expect(markText({ until: EARLIER }, NOW)).toBe(`Suspended in ${month} for breaking the site rules`)
    expect(markText(undefined, NOW)).toBe('')
  })

  it('dates a suspension notice in the viewer\'s time', () => {
    const notice = { content: 'in UTC', metadata: { notice: 'account_suspended', until: LATER } }
    expect(noticeText(notice)).toBe(`This account is suspended until ${formatUntil(LATER)} for breaking the site rules.`)
    expect(noticeText({ content: 'This account is active again.', metadata: { notice: 'account_restored' } }))
      .toBe('This account is active again.')
  })
})
