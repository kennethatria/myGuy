import { describe, it, expect } from 'vitest'
import { statusLabel, endsConversation, stateFromMessage } from '../conversationStatus'

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
})
