import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BookingStepMessage from '../BookingStepMessage.vue'
import type { Message } from '@/stores/messages'

const BUYER = 2
const SELLER = 4
type Status = NonNullable<Message['metadata']>['status']

// The booking's request (buyer → seller) at a status, with ratings so far
const request = (status: Status, ratings: Partial<NonNullable<Message['metadata']>> = {}): Message => ({
  id: 1, store_item_id: 9, sender_id: BUYER, recipient_id: SELLER, content: '', message_type: 'booking_request',
  metadata: { booking_id: 5, item_title: 'Bike', status, ...ratings },
  is_read: false, is_edited: false, is_deleted: false, created_at: '2026-10-07T10:00:00Z'
} as Message)

// A step note for that booking
const step = (type: Message['message_type'], status: Status): Message => ({
  id: 2, store_item_id: 9, sender_id: SELLER, recipient_id: BUYER, content: `note ${status}`, message_type: type,
  metadata: { booking_id: 5, status },
  is_read: false, is_edited: false, is_deleted: false, created_at: '2026-10-07T11:00:00Z'
} as Message)

const show = (note: Message, req: Message, viewer: number, latest = true) =>
  mount(BookingStepMessage, { props: { message: note, request: req, currentUserId: viewer, latest } })
const buttons = (w: ReturnType<typeof show>) => w.findAll('button').map(b => b.text())

describe('BookingStepMessage', () => {
  it('asks the seller to mark it picked up once approved, or release it', async () => {
    const wrapper = show(step('booking_approved', 'approved'), request('approved'), SELLER)
    expect(buttons(wrapper)).toEqual(['Picked up', 'Release reservation'])

    await wrapper.find('.btn-picked-up').trigger('click')
    expect(wrapper.emitted('bookingAction')).toEqual([[5, 'confirm-delivery', undefined, undefined]])
  })

  it('asks once more before releasing, and can keep it', async () => {
    const wrapper = show(step('booking_approved', 'approved'), request('approved'), SELLER)
    await wrapper.find('.btn-release').trigger('click')
    expect(wrapper.text()).toContain('Put it back on the board for others?')
    await wrapper.find('.btn-keep').trigger('click')
    expect(wrapper.find('.btn-release').exists()).toBe(true)

    await wrapper.find('.btn-release').trigger('click')
    await wrapper.find('.btn-decline').trigger('click')
    expect(wrapper.emitted('bookingAction')).toEqual([[5, 'release', undefined, undefined]])
  })

  it('tells the buyer they wait for the seller', () => {
    const wrapper = show(step('booking_approved', 'approved'), request('approved'), BUYER)
    expect(wrapper.text()).toContain('Waiting for the seller to mark it picked up.')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('asks the buyer to confirm once the seller marks it picked up', async () => {
    const buyer = show(step('booking_picked_up', 'picked_up'), request('picked_up'), BUYER)
    await buyer.find('.btn-confirm-received').trigger('click')
    expect(buyer.emitted('bookingAction')).toEqual([[5, 'confirm-received', undefined, undefined]])

    const seller = show(step('booking_picked_up', 'picked_up'), request('picked_up'), SELLER)
    expect(seller.text()).toContain('Waiting for the buyer to confirm')
    expect(seller.find('button').exists()).toBe(false)
  })

  it('lets the seller finish an older booking the buyer already confirmed', async () => {
    const wrapper = show(step('booking_item_received', 'item_received'), request('item_received'), SELLER)
    await wrapper.find('.btn-confirm-delivery').trigger('click')
    expect(wrapper.emitted('bookingAction')).toEqual([[5, 'confirm-delivery', undefined, undefined]])
  })

  it('offers each side a review once complete, comment optional', async () => {
    const buyer = show(step('booking_completed', 'completed'), request('completed'), BUYER)
    await buyer.findAll('.star')[3].trigger('click')
    await buyer.find('form').trigger('submit')
    expect(buyer.emitted('bookingAction')).toEqual([[5, 'rate-seller', 4, '']])

    const seller = show(step('booking_completed', 'completed'), request('completed'), SELLER)
    await seller.findAll('.star')[4].trigger('click')
    await seller.find('form').trigger('submit')
    expect(seller.emitted('bookingAction')).toEqual([[5, 'rate-buyer', 5, '']])
  })

  it('shows the review given instead of the form', () => {
    const wrapper = show(step('booking_completed', 'completed'), request('completed', { buyer_rating: 4, buyer_review: 'Smooth' }), BUYER)
    expect(wrapper.text()).toContain('You gave ★ 4: “Smooth”')
    expect(wrapper.find('form').exists()).toBe(false)
  })

  it('offers nothing once the booking moved on, or on an older step', () => {
    // The request is already picked up: the approval step is history
    expect(show(step('booking_approved', 'approved'), request('picked_up'), SELLER).find('button').exists()).toBe(false)
    // Not the newest step of this booking
    expect(show(step('booking_approved', 'approved'), request('approved'), SELLER, false).find('button').exists()).toBe(false)
  })
})
