import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, RouterLinkStub } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import RequestOfferMessage from '../RequestOfferMessage.vue'
import type { Message } from '@/stores/messages'

const SELLER = 4
const REQUESTER = 3

// The seller tells the requester that item 9 was listed for their request
const offer = {
  id: 1, store_item_id: 9, sender_id: SELLER, recipient_id: REQUESTER,
  content: '🎁 I listed "Tent" for your request "Camping tent".', message_type: 'system_alert',
  metadata: { event: 'request_answered', request_id: 5 },
  is_read: false, is_edited: false, is_deleted: false, created_at: '2026-10-07T10:00:00Z'
} as Message

// Answers the store API: the item's status and whether the viewer booked it
function api({ status = 'active', booked = false, bookOk = true } = {}) {
  return vi.fn((url: string, init?: RequestInit) => {
    if (init?.method === 'POST') {
      return Promise.resolve(new Response(JSON.stringify(bookOk ? { id: 7 } : { error: 'item is not available for booking' }), { status: bookOk ? 201 : 400 }))
    }
    if (url.endsWith('/booking-request')) {
      return Promise.resolve(new Response(JSON.stringify({ booking_request: booked ? { id: 7 } : null }), { status: 200 }))
    }
    return Promise.resolve(new Response(JSON.stringify({ id: 9, status }), { status: 200 }))
  })
}

const show = async (viewer: number) => {
  const wrapper = mount(RequestOfferMessage, {
    props: { message: offer, currentUserId: viewer },
    global: { stubs: { RouterLink: RouterLinkStub } }
  })
  await flushPromises()
  return wrapper
}

describe('RequestOfferMessage', () => {
  beforeEach(() => setActivePinia(createPinia()))
  afterEach(() => vi.unstubAllGlobals())

  it('lets the requester book it right in the conversation', async () => {
    const fetch = api()
    vi.stubGlobal('fetch', fetch)
    const wrapper = await show(REQUESTER)

    const book = wrapper.findAll('button').find(b => b.text() === 'Book it')!
    await book.trigger('click')
    await flushPromises()

    expect(fetch).toHaveBeenCalledWith(expect.stringMatching(/\/items\/9\/booking-request$/), expect.objectContaining({ method: 'POST', body: '{}' }))
    expect(wrapper.text()).toContain('You booked it')
  })

  it('says so once the requester has booked it', async () => {
    vi.stubGlobal('fetch', api({ booked: true }))
    const wrapper = await show(REQUESTER)
    expect(wrapper.text()).toContain('You booked it')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('says it is no longer available once reserved, sold or removed', async () => {
    vi.stubGlobal('fetch', api({ status: 'reserved' }))
    const wrapper = await show(REQUESTER)
    expect(wrapper.text()).toContain('No longer available.')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('shows why booking failed', async () => {
    vi.stubGlobal('fetch', api({ bookOk: false }))
    const wrapper = await show(REQUESTER)
    await wrapper.findAll('button').find(b => b.text() === 'Book it')!.trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').text()).toBe('item is not available for booking')
  })

  it('offers the seller nothing to do', async () => {
    const fetch = api()
    vi.stubGlobal('fetch', fetch)
    const wrapper = await show(SELLER)
    expect(wrapper.text()).toContain('Waiting for them to book it.')
    expect(wrapper.find('button').exists()).toBe(false)
    expect(fetch).not.toHaveBeenCalled()
  })
})
