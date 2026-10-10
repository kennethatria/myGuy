import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useChatStore } from '@/stores/chat'
import { useReviewsStore } from '@/stores/reviews'
import type { ConversationSummary } from '@/stores/messages'
import ChatWidget from '../ChatWidget.vue'

vi.mock('vue-router', () => ({ useRoute: () => ({ meta: {} }) }))

const conv = (item_id: number, time: string, extra: Partial<ConversationSummary> = {}): ConversationSummary => ({
  item_id, item_title: `Item ${item_id}`, other_user_id: 2, other_user_name: 'bea', last_message: '',
  last_message_time: time, unread_count: 0, conversation_type: 'store', ...extra
})

const show = () => mount(ChatWidget, { global: { stubs: { MessageThread: true, DeletionWarningBanner: true } } })
const titles = (wrapper: ReturnType<typeof show>) => wrapper.findAll('.conversation-title').map(t => t.text())

describe('ChatWidget list', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(useReviewsStore(), 'fetchInteractions').mockResolvedValue([])
    const chat = useChatStore()
    chat.widgetOpen = true
    chat.conversations = [
      conv(1, '2026-10-10T08:00:00Z', { unread_count: 2 }),
      conv(2, '2026-10-10T10:00:00Z', { state: 'completed', ended: true }),
      conv(3, '2026-10-10T11:00:00Z', { item_status: 'expired' }),
      conv(4, '2026-10-10T09:00:00Z', { state: 'approved' })
    ]
  })

  it('opens on Active, latest message first', () => {
    const wrapper = show()
    expect(wrapper.find('[aria-selected="true"]').text()).toContain('Active')
    expect(titles(wrapper)).toEqual(['Item 4', 'Item 1'])
  })

  it('lists ended deals and expired posts under Done, with what happened', async () => {
    const wrapper = show()
    await wrapper.findAll('[role="tab"]')[1].trigger('click')
    expect(titles(wrapper)).toEqual(['Item 3', 'Item 2'])
    const chips = wrapper.findAll('.conversation-item').map(i => i.find('.status-chip').text())
    expect(chips).toEqual(['Expired', 'Completed'])
  })
})
