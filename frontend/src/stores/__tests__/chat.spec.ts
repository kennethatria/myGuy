import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { Message } from '@/stores/messages'

// Fake socket: records emits and lets tests trigger server events.
const handlers: Record<string, (payload: unknown) => void> = {}
const fakeSocket = {
  connected: true,
  on: vi.fn((event: string, handler: (payload: unknown) => void) => { handlers[event] = handler }),
  emit: vi.fn(),
  disconnect: vi.fn()
}
vi.mock('socket.io-client', () => ({ io: vi.fn(() => fakeSocket) }))

const SELLER = 1
vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({ user: { id: SELLER }, token: 'token' }))
}))
vi.mock('@/stores/user', () => ({
  useUserStore: vi.fn(() => ({ fetchUsers: vi.fn(), getUserById: vi.fn() }))
}))
vi.mock('@/stores/context', () => ({
  useContextStore: vi.fn(() => ({
    fetchTasks: vi.fn(), fetchItems: vi.fn(), getTaskById: vi.fn(), getItemById: vi.fn()
  }))
}))

import { useChatStore, conversationKey } from '@/stores/chat'

let nextId = 1
function message(fields: Partial<Message>): Message {
  return {
    id: nextId++,
    sender_id: 0,
    recipient_id: 0,
    content: 'hi',
    message_type: 'text',
    is_read: false,
    is_edited: false,
    is_deleted: false,
    created_at: new Date().toISOString(),
    ...fields
  } as Message
}

const server = (event: string, payload: unknown) => handlers[event]!(payload)

describe('chat store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    fakeSocket.emit.mockClear()
    const store = useChatStore()
    store.connectSocket()
  })

  it('gives every conversation a distinct key', () => {
    expect(conversationKey({ task_id: 5, other_user_id: 2 })).toBe('task:5:2')
    expect(conversationKey({ application_id: 5, other_user_id: 2 })).toBe('application:5:2')
    expect(conversationKey({ item_id: 5, other_user_id: 2 })).not.toBe(conversationKey({ item_id: 5, other_user_id: 3 }))
  })

  it("keeps two buyers' chats about the same item apart", () => {
    const store = useChatStore()
    server('message:new', message({ store_item_id: 5, sender_id: 2, recipient_id: SELLER, content: 'buyer 2' }))
    server('message:new', message({ store_item_id: 5, sender_id: 3, recipient_id: SELLER, content: 'buyer 3' }))

    expect(store.conversations.map(c => conversationKey(c)).sort()).toEqual(['store:5:2', 'store:5:3'])
    expect(store.conversations.every(c => c.unread_count === 1)).toBe(true)
  })

  it('does not merge a task and an application that share an id', () => {
    const store = useChatStore()
    server('message:new', message({ task_id: 5, sender_id: 4, recipient_id: SELLER }))
    server('message:new', message({ application_id: 5, sender_id: 6, recipient_id: SELLER }))

    expect(store.conversations).toHaveLength(2)
    expect(store.totalUnreadCount).toBe(2)
  })

  it('loads, appends and de-duplicates messages per conversation', () => {
    const store = useChatStore()
    server('conversations:list', [
      { item_id: 5, other_user_id: 2, unread_count: 0, last_message: '', last_message_time: '', other_user_name: 'b2', conversation_type: 'store' }
    ])
    store.joinConversation(store.conversations[0]!)

    expect(fakeSocket.emit).toHaveBeenCalledWith('messages:get', { itemId: 5, otherUserId: 2, limit: 20, offset: 0 })

    server('messages:list', { itemId: 5, otherUserId: 2, offset: 0, messages: [] })
    const incoming = message({ store_item_id: 5, sender_id: 2, recipient_id: SELLER })
    server('message:new', incoming)
    server('message:new', incoming) // delivered twice (e.g. two paths)
    server('message:new', message({ store_item_id: 5, sender_id: 3, recipient_id: SELLER })) // other buyer

    expect(store.activeMessages.map(m => m.id)).toEqual([incoming.id])
  })

  it('marks a message read on the server when its conversation is open', () => {
    const store = useChatStore()
    server('conversations:list', [
      { task_id: 7, other_user_id: 4, unread_count: 0, last_message: '', last_message_time: '', other_user_name: 'u4', conversation_type: 'task' }
    ])
    store.joinConversation(store.conversations[0]!)
    fakeSocket.emit.mockClear()

    server('message:new', message({ task_id: 7, sender_id: 4, recipient_id: SELLER }))

    expect(fakeSocket.emit).toHaveBeenCalledWith('conversation:read', { taskId: 7, otherUserId: 4 })
    expect(store.conversations[0]!.unread_count).toBe(0)
  })

  it('sends typing only to the other participant', () => {
    const store = useChatStore()
    server('conversations:list', [
      { item_id: 5, other_user_id: 3, unread_count: 0, last_message: '', last_message_time: '', other_user_name: 'b3', conversation_type: 'store' }
    ])
    store.joinConversation(store.conversations[0]!)
    store.startTyping()

    expect(fakeSocket.emit).toHaveBeenCalledWith('typing:start', { itemId: 5, recipientId: 3 })
  })
})
