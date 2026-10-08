// Sticky-note rules for gigs and marketplace listings, mirroring the backend
// (internal/services/task_service.go) and store-service (store_service.go).
// The backend is the authority; these only drive counters and the look.

import { timeAgo } from '@/utils/conversationStatus'

export const HEADLINE_MAX_WORDS = 5
export const BODY_MAX_WORDS = 20
export const HEADLINE_MAX_CHARS = 60
export const BODY_MAX_CHARS = 200

const NOTE_COLORS = ['yellow', 'pink', 'blue', 'green', 'orange'] as const
export type NoteColor = (typeof NOTE_COLORS)[number]

/** Words as the backend counts them: runs of non-space characters. */
export function countWords(text: string): number {
  const trimmed = text.trim()
  return trimmed ? trimmed.split(/\s+/).length : 0
}

export function headlineFits(text: string): boolean {
  return countWords(text) <= HEADLINE_MAX_WORDS && text.trim().length <= HEADLINE_MAX_CHARS
}

export function bodyFits(text: string): boolean {
  return countWords(text) <= BODY_MAX_WORDS && text.trim().length <= BODY_MAX_CHARS
}

/** A stable colour per gig, so a note looks the same every visit. */
export function noteColor(id: number): NoteColor {
  return NOTE_COLORS[Math.abs(id) % NOTE_COLORS.length]
}

/** A slight stable tilt (−0.6° to 0.6°) so the board looks hand-pinned. */
export function noteTilt(id: number): number {
  return (((Math.abs(id) * 7) % 5) - 2) * 0.3
}

/**
 * How long an open gig stays on the board. Once the deadline passes the
 * backend expires it unless someone applied, so past-deadline open gigs
 * (which must have applicants) get no countdown.
 */
export function expiryLabel(deadline: string, now: Date = new Date()): string {
  const left = timeLeft(deadline, now)
  return left ? `Expires in ${left}` : ''
}

/** Time left before a note expires, short ("23h", "20m"), or '' once past. */
export function timeLeft(deadline: string, now: Date = new Date()): string {
  const ms = new Date(deadline).getTime() - now.getTime()
  if (Number.isNaN(ms) || ms <= 0) return ''
  const minutes = Math.ceil(ms / 60000)
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h`
}

/**
 * A note's footer line: "~2 km · @ann · 23h". The distance shows as
 * "No location" only where distances are in play (see hasDistances), and
 * the time only while the note is live.
 */
export function noteMeta(
  note: { distance?: string; deadline?: string },
  who: string,
  showUnknown: boolean,
  now: Date = new Date()
): string {
  const distance = note.distance || (showUnknown ? 'No location' : '')
  const left = note.deadline ? timeLeft(note.deadline, now) : ''
  return [distance, who && `@${who}`, left].filter(Boolean).join(' · ')
}

/** A post page's time line: "Posted 2 h ago · Expires in 23h" (expiry only while live). */
export function postedMeta(createdAt: string, deadline: string | undefined, live: boolean, now: Date = new Date()): string {
  const expires = live && deadline ? expiryLabel(deadline, now) : ''
  return [`Posted ${timeAgo(createdAt, now)}`, expires].filter(Boolean).join(' · ')
}
