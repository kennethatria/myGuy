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

  it('opens a task chat with no history so the first message can be sent', () => {
    const store = useChatStore()
    store.openConversationWith({ taskId: 9, otherUserId: 4, otherUserName: 'Ann' })

    expect(fakeSocket.emit).toHaveBeenCalledWith('messages:get', { taskId: 9, otherUserId: 4, limit: 20, offset: 0 })
    store.sendMessage('first message', 4)
    expect(fakeSocket.emit).toHaveBeenCalledWith('message:send', { taskId: 9, recipientId: 4, content: 'first message' })
  })

  it('keeps the open chat when the conversation list arrives afterwards', () => {
    const store = useChatStore()
    store.openConversationWith({ taskId: 9, otherUserId: 4 })
    server('conversations:list', [
      { task_id: 3, other_user_id: 5, unread_count: 0, last_message: '', last_message_time: '', other_user_name: 'x', conversation_type: 'task' }
    ])

    expect(conversationKey(store.activeConversation!)).toBe('task:9:4')
    expect(store.conversations.some(c => conversationKey(c) === 'task:9:4')).toBe(true)
  })

  it("opens the owner's chat with the assignee, not another person who wrote about the task", () => {
    const store = useChatStore()
    server('conversations:list', [
      { task_id: 9, other_user_id: 7, unread_count: 0, last_message: 'question', last_message_time: '', other_user_name: 'asker', conversation_type: 'task' },
      { task_id: 9, other_user_id: 4, unread_count: 0, last_message: 'on it', last_message_time: '', other_user_name: 'assignee', conversation_type: 'task' }
    ])
    store.openConversationWith({ taskId: 9, otherUserId: 4 })

    expect(store.activeConversation?.other_user_id).toBe(4)
  })

  it("opens a seller's chat with the buyer named in the link, and never guesses", async () => {
    const store = useChatStore()
    const storeConv = (buyer: number) => ({ item_id: 5, other_user_id: buyer, unread_count: 0, last_message: '', last_message_time: '', other_user_name: `b${buyer}`, conversation_type: 'store' })
    server('conversations:list', [storeConv(2), storeConv(3)])

    await store.joinStoreConversation(5)
    expect(store.activeConversation).toBeNull()

    await store.joinStoreConversation(5, 3)
    expect(conversationKey(store.activeConversation!)).toBe('store:5:3')
  })

  it('opens the only chat about an item when no user is named', async () => {
    const store = useChatStore()
    server('conversations:list', [
      { item_id: 8, other_user_id: 6, unread_count: 0, last_message: '', last_message_time: '', other_user_name: 'seller', conversation_type: 'store' }
    ])

    await store.joinStoreConversation(8)
    expect(conversationKey(store.activeConversation!)).toBe('store:8:6')
  })
})

