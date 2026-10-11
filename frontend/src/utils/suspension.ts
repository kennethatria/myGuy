// How the app words suspensions: a suspended account is read-only until a
// date, its conversations are paused meanwhile, and its profile says so
// (and, for 90 days after, that it was).

export interface SuspensionMark {
  until?: string // suspended now, until then
  ended?: string // was, and ended then
}

/** "Wed 14 Oct, 14:00" in the viewer's own time. */
export function formatUntil(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit'
  })
}

/** Whether a suspension ending at until is still on. */
export function suspendedNow(until?: string | null, now = Date.now()): boolean {
  return !!until && new Date(until).getTime() > now
}

/** The profile mark, or '' for none. */
export function markText(mark?: SuspensionMark | null, now = Date.now()): string {
  if (!mark) return ''
  if (suspendedNow(mark.until, now)) return `Suspended until ${formatUntil(mark.until!)} for breaking the site rules`
  const ended = mark.ended ?? mark.until
  if (!ended) return ''
  const month = new Date(ended).toLocaleString(undefined, { month: 'long', year: 'numeric' })
  return `Suspended in ${month} for breaking the site rules`
}

/** A notice in a conversation, with a suspension's end in the viewer's time. */
export function noticeText(message: { content: string; metadata?: { notice?: string; until?: string } }): string {
  const { notice, until } = message.metadata ?? {}
  return notice === 'account_suspended' && until
    ? `This account is suspended until ${formatUntil(until)} for breaking the site rules.`
    : message.content
}
