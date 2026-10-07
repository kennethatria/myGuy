import { describe, it, expect, beforeEach } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import MessageThread from '../MessageThread.vue'
import type { ConversationSummary, Message } from '@/stores/messages'

const conversation = {
  task_id: 1, task_title: 'Fix sink', last_message: '', last_message_time: '2026-10-07T10:00:00Z',
  other_user_id: 2, other_user_name: 'ann', unread_count: 0, conversation_type: 'task'
} as ConversationSummary

const completed = {
  id: 9, task_id: 1, sender_id: 1, recipient_id: 2, content: 'Complete', message_type: 'system_alert',
  metadata: { event: 'completed' }, is_read: true, is_edited: false, is_deleted: false, created_at: '2026-10-07T10:00:00Z'
} as Message

const show = (props: Record<string, unknown>) => mount(MessageThread, {
  props: { conversation, messages: [], typingUsers: [], loading: false, hasMore: false, ...props },
  global: { stubs: { RouterLink: RouterLinkStub, TaskEventMessage: true } }
})

describe('MessageThread', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('replaces the message box once the conversation has ended, pointing to the review', () => {
    const wrapper = show({ ended: true, messages: [completed] })
    expect(wrapper.find('.message-input').exists()).toBe(false)
    expect(wrapper.text()).toContain('This conversation has ended. You can still read it and leave your review above.')
  })

  it('keeps the message box while it is going on', () => {
    const wrapper = show({ ended: false })
    expect(wrapper.find('.message-input').exists()).toBe(true)
  })
})
