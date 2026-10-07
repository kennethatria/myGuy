// What the floating chat says about where a conversation stands: its latest
// gig event or booking status, in words, and whether it has ended (the deal
// is done, or closed without one).

const TASK_LABELS: Record<string, string> = {
  application: 'Applied',
  accepted: 'Accepted',
  done: 'Awaiting approval',
  not_done: 'Accepted',
  completed: 'Completed',
  declined: 'Declined',
  cancelled: 'Cancelled'
}

const STORE_LABELS: Record<string, string> = {
  pending: 'Requested',
  approved: 'Approved',
  picked_up: 'Picked up',
  item_received: 'Collected',
  completed: 'Completed',
  rejected: 'Declined',
  released: 'Released'
}

const ENDED: Record<'task' | 'store', Set<string>> = {
  task: new Set(['completed', 'declined', 'cancelled']),
  store: new Set(['completed', 'rejected', 'released'])
}

type Kind = 'task' | 'store'

const kindOf = (c: { task_id?: number | null; item_id?: number | null }): Kind | null =>
  c.task_id ? 'task' : c.item_id ? 'store' : null

/** The state in words ("Picked up"), or '' when there's nothing to say. */
export function statusLabel(c: { task_id?: number | null; item_id?: number | null; state?: string | null }): string {
  const kind = kindOf(c)
  if (!kind || !c.state) return ''
  return (kind === 'task' ? TASK_LABELS : STORE_LABELS)[c.state] ?? ''
}

/** Whether a state ends the conversation for a gig or an item. */
export function endsConversation(kind: Kind | null, state?: string | null): boolean {
  return !!kind && !!state && ENDED[kind].has(state)
}

/**
 * The state a message moves a conversation to, if any: a gig event, or a
 * booking's status (on its request or a step note).
 */
export function stateFromMessage(m: { task_id?: number | null; message_type: string; metadata?: { event?: string; status?: string } | null }): string | null {
  if (m.task_id) return m.metadata?.event ?? null
  if (m.message_type === 'booking_request' || m.message_type.startsWith('booking_') || m.metadata?.status) {
    return m.metadata?.status ?? null
  }
  return null
}

export { kindOf }
