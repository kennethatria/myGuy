import { describe, it, expect } from 'vitest'
import { statusLabel, endsConversation, stateFromMessage, timeAgo, statusIcon, postExpired } from '../conversationStatus'

describe('conversationStatus', () => {
  it('names where a gig or a booking stands', () => {
    expect(statusLabel({ task_id: 1, state: 'application' })).toBe('Applied')
    expect(statusLabel({ task_id: 1, state: 'done' })).toBe('Awaiting approval')
    expect(statusLabel({ item_id: 2, state: 'picked_up' })).toBe('Picked up')
    expect(statusLabel({ item_id: 2, state: 'completed' })).toBe('Completed')
    expect(statusLabel({ item_id: 2, state: null })).toBe('')
  })

  it('ends a conversation once the deal is done or closed', () => {
    for (const state of ['completed', 'declined', 'cancelled']) expect(endsConversation('task', state)).toBe(true)
    for (const state of ['completed', 'rejected', 'released']) expect(endsConversation('store', state)).toBe(true)
    for (const state of ['application', 'accepted', 'done']) expect(endsConversation('task', state)).toBe(false)
    for (const state of ['pending', 'approved', 'picked_up']) expect(endsConversation('store', state)).toBe(false)
  })

  it('reads the new state from a gig event or a booking step', () => {
    expect(stateFromMessage({ task_id: 1, message_type: 'system_alert', metadata: { event: 'completed' } })).toBe('completed')
    expect(stateFromMessage({ message_type: 'booking_picked_up', metadata: { status: 'picked_up' } })).toBe('picked_up')
    expect(stateFromMessage({ message_type: 'system_alert', metadata: { status: 'released' } })).toBe('released')
    expect(stateFromMessage({ message_type: 'store', metadata: null })).toBeNull()
  })

  it('says shortly how long ago a deal reached its state', () => {
    const now = new Date('2026-10-08T12:00:00Z')
    expect(timeAgo('2026-10-08T11:59:40Z', now)).toBe('just now')
    expect(timeAgo('2026-10-08T11:55:00Z', now)).toBe('5 min ago')
    expect(timeAgo('2026-10-08T09:00:00Z', now)).toBe('3 h ago')
    expect(timeAgo('2026-10-05T11:00:00Z', now)).toBe('3 d ago')
    // A clock slightly ahead never says "-1 min ago"
    expect(timeAgo('2026-10-08T12:01:00Z', now)).toBe('just now')
  })

  it('gives each state an icon: a green tick once completed', () => {
    expect(statusIcon({ task_id: 1, state: 'completed' })).toEqual({ icon: 'fa-circle-check', tone: 'done' })
    expect(statusIcon({ item_id: 2, state: 'completed' })).toEqual({ icon: 'fa-circle-check', tone: 'done' })
    expect(statusIcon({ item_id: 2, state: 'rejected' })?.tone).toBe('stopped')
    expect(statusIcon({ task_id: 1, state: 'done' })?.tone).toBe('waiting')
    // Every labelled state has one; unlabelled ones have none
    for (const state of ['application', 'accepted', 'done', 'not_done', 'completed', 'declined', 'cancelled']) expect(statusIcon({ task_id: 1, state })).not.toBeNull()
    for (const state of ['pending', 'approved', 'picked_up', 'item_received', 'completed', 'rejected', 'released']) expect(statusIcon({ item_id: 1, state })).not.toBeNull()
    expect(statusIcon({ task_id: 1, state: 'request_answered' })).toBeNull()
  })

  it('knows when the post a conversation is about expired', () => {
    expect(postExpired({ task_id: 1, task_status: 'expired' })).toBe(true)
    expect(postExpired({ item_id: 2, item_status: 'expired' })).toBe(true)
    // Reposted, or never expired
    expect(postExpired({ task_id: 1, task_status: 'open' })).toBe(false)
    expect(postExpired({ item_id: 2, item_status: 'active' })).toBe(false)
    // Not loaded yet
    expect(postExpired({ item_id: 2 })).toBe(false)
  })
})
