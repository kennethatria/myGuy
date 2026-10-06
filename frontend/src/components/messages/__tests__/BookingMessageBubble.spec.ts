import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import BookingMessageBubble from '../BookingMessageBubble.vue'
import type { Message } from '@/stores/messages'

// The buyer's booking request on item 9, as the seller sees it
const booking = (status: NonNullable<Message['metadata']>['status']): Message => ({
  id: 1, store_item_id: 9, sender_id: 2, recipient_id: 1, content: 'I would like it',
  message_type: 'booking_request', metadata: { booking_id: 5, item_title: 'Bike', status },
  is_read: false, is_edited: false, is_deleted: false, created_at: '2026-10-06T10:00:00Z'
} as Message)

describe('BookingMessageBubble', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('asks the seller to approve or decline a new booking', async () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('pending'), isOwnMessage: false } })
    expect(wrapper.text()).toContain('asked to book "Bike"')
    await wrapper.find('.btn-approve').trigger('click')
    expect(wrapper.emitted('bookingAction')).toEqual([[5, 'approve']])
  })

  it('tells the buyer they are waiting on the seller', () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('pending'), isOwnMessage: true } })
    expect(wrapper.text()).toContain('You asked to book "Bike"')
    expect(wrapper.text()).toContain('Waiting for the seller to answer.')
    expect(wrapper.find('.btn-approve').exists()).toBe(false)
  })

  it('lets the buyer say they collected it, then the seller confirm the handover', async () => {
    const buyer = mount(BookingMessageBubble, { props: { message: booking('approved'), isOwnMessage: true } })
    await buyer.find('.btn-confirm-received').trigger('click')
    expect(buyer.emitted('bookingAction')).toEqual([[5, 'confirm-received']])

    const seller = mount(BookingMessageBubble, { props: { message: booking('item_received'), isOwnMessage: false } })
    await seller.find('.btn-confirm-delivery').trigger('click')
    expect(seller.emitted('bookingAction')).toEqual([[5, 'confirm-delivery']])
  })

  it('offers each side a review once completed, comment optional', async () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('completed'), isOwnMessage: true } })
    await wrapper.findAll('.star')[3].trigger('click')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('bookingAction')).toEqual([[5, 'rate-seller', 4, '']])
  })

  it('lets the seller release an approved booking, after asking once more', async () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('approved'), isOwnMessage: false } })

    await wrapper.find('.btn-release').trigger('click')
    expect(wrapper.text()).toContain('Put it back on the board for others?')
    expect(wrapper.emitted('bookingAction')).toBeUndefined()

    await wrapper.find('.btn-decline').trigger('click')
    expect(wrapper.emitted('bookingAction')).toEqual([[5, 'release']])
  })

  it('can keep the reservation instead', async () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('approved'), isOwnMessage: false } })
    await wrapper.find('.btn-release').trigger('click')
    await wrapper.find('.btn-keep').trigger('click')
    expect(wrapper.find('.btn-release').exists()).toBe(true)
    expect(wrapper.emitted('bookingAction')).toBeUndefined()
  })

  it('offers the buyer no release', () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('approved'), isOwnMessage: true } })
    expect(wrapper.find('.btn-release').exists()).toBe(false)
  })

  it('shows a released booking as released', () => {
    const wrapper = mount(BookingMessageBubble, { props: { message: booking('released'), isOwnMessage: true } })
    expect(wrapper.text()).toContain('Reservation released')
    expect(wrapper.text()).toContain('Released')
  })
})
