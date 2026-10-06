import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import StoreItemView from '@/views/store/StoreItemView.vue'

// Mock stores so they don't call fetch during initialization
vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({
    user: { id: 1, username: 'buyer1' },
    isAuthenticated: true,
    checkAuth: vi.fn().mockResolvedValue(true),
    token: 'mock-token'
  }))
}))

// Shared so a test can check which conversation the floating chat opened
const openChat = vi.fn()

vi.mock('@/stores/chat', () => ({
  useChatStore: vi.fn(() => ({
    getStoreMessages: vi.fn().mockReturnValue([]),
    messages: [],
    sendMessage: vi.fn(),
    openChat
  }))
}))

// Mock fetch globally
global.fetch = vi.fn()

// Create a mock router
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: { template: '<div>Home</div>' } },
    { path: '/store/:id', name: 'store-item', component: StoreItemView }
  ]
})

describe('Store Booking Flow', () => {
  let wrapper
  let pinia

  const mockItem = {
    id: 1,
    title: 'Test Item',
    description: 'A test item for booking',
    price: 50000,
    is_auction: false,
    status: 'active',
    seller: {
      id: 2,
      full_name: 'Test Seller',
      username: 'seller1'
    }
  }

  beforeEach(async () => {
    vi.clearAllMocks()
    pinia = createPinia()

    // Seed token so localStorage.getItem('token') returns the expected value
    localStorage.setItem('token', 'mock-token')

    // Default: initial loadItem (from onMounted on '/') gets no route param → fails gracefully
    global.fetch.mockResolvedValue({ ok: false, status: 404, json: async () => ({}) })

    await router.push('/')

    wrapper = mount(StoreItemView, {
      global: {
        plugins: [router, pinia],
        stubs: {
          'router-link': true,
          ChatWindow: { template: '<div class="chat-window-stub"></div>' },
          BookingConfirmationModal: { template: '<div></div>', props: ['isOpen', 'itemId', 'sellerId', 'itemTitle'] }
        }
      }
    })
  })

  describe('Booking Request Button', () => {
    it('should show "Book Now" button for non-owner users on active fixed-price items', async () => {
      // Mock item fetch (non-owner: seller.id=2, user.id=1)
      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => mockItem })
        // Mock loadBookingRequest → no existing booking
        .mockResolvedValueOnce({ ok: false, status: 404, json: async () => ({}) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()
      await wrapper.vm.$nextTick()

      const bookingButton = wrapper.find('[data-testid="booking-request-btn"]')
      expect(bookingButton.exists()).toBe(true)
      expect(bookingButton.text()).toContain('Book Now')
      expect(bookingButton.attributes('disabled')).toBeUndefined()
    })

    it('should not show booking section for item owner', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }

      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => ownerItem })
        // owner: loadBookingRequest loads all requests
        .mockResolvedValueOnce({ ok: true, json: async () => ({ booking_requests: [] }) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()
      await wrapper.vm.$nextTick()

      const bookingSection = wrapper.find('.booking-section')
      expect(bookingSection.exists()).toBe(false)
    })

    it('should not show booking button for inactive items', async () => {
      const inactiveItem = { ...mockItem, status: 'sold' }

      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => inactiveItem })
        .mockResolvedValueOnce({ ok: false, status: 404, json: async () => ({}) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()
      await wrapper.vm.$nextTick()

      const bookingSection = wrapper.find('.booking-section')
      expect(bookingSection.exists()).toBe(false)
    })
  })

  describe('Booking Request Creation', () => {
    it('should send booking request when button is clicked', async () => {
      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => mockItem })
        .mockResolvedValueOnce({ ok: false, status: 404, json: async () => ({}) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      const mockBookingRequest = {
        id: 1,
        item_id: 1,
        requester_id: 1,
        status: 'pending',
        message: '',
        created_at: new Date().toISOString()
      }

      global.fetch.mockResolvedValueOnce({ ok: true, json: async () => mockBookingRequest })

      await wrapper.vm.sendBookingRequest()

      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/v1/items/1/booking-request',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'Authorization': 'Bearer mock-token'
          }),
          // One tap, like applying for a gig: no message
          body: JSON.stringify({})
        })
      )

      expect(wrapper.vm.bookingRequest).toEqual(mockBookingRequest)
      expect(wrapper.vm.hasBookingRequest).toBe(true)
      // Into the conversation with the seller, in the floating chat
      expect(openChat).toHaveBeenCalledWith(expect.objectContaining({ itemId: mockItem.id, otherUserId: mockItem.seller.id }))
    })

    it('should handle booking request errors', async () => {
      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => mockItem })
        .mockResolvedValueOnce({ ok: false, status: 404, json: async () => ({}) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      global.fetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Cannot book your own item' })
      })

      await wrapper.vm.sendBookingRequest()

      // Shown on the page, not in a browser alert
      expect(wrapper.vm.bookingError).toBe('Cannot book your own item')
      expect(wrapper.vm.hasBookingRequest).toBe(false)
    })
  })

  describe('Booking Request Status Display', () => {
    it('should show pending status correctly', async () => {
      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => mockItem })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ booking_request: { id: 1, status: 'pending', created_at: new Date().toISOString() } }) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      expect(wrapper.vm.bookingStatus).toBe('pending')
      expect(wrapper.vm.hasBookingRequest).toBe(true)
    })

    it('should show approved status correctly', async () => {
      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => mockItem })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ booking_request: { id: 1, status: 'approved', created_at: new Date().toISOString() } }) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      expect(wrapper.vm.bookingStatus).toBe('approved')
    })

    it('should show rejected status', async () => {
      global.fetch
        .mockResolvedValueOnce({ ok: true, json: async () => mockItem })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ booking_request: { id: 1, status: 'rejected', created_at: new Date().toISOString() } }) })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      expect(wrapper.vm.bookingStatus).toBe('rejected')
    })
  })

  describe('Owner Booking Management', () => {
    it('should show booking management interface for item owners', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      const pendingBookingRequest = {
        id: 1,
        status: 'pending',
        requester: { id: 2, username: 'buyer1' },
        created_at: new Date().toISOString()
      }

      wrapper.vm.item = ownerItem
      wrapper.vm.bookingRequest = pendingBookingRequest
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.item.seller.id).toBe(wrapper.vm.userId)
      expect(wrapper.vm.bookingRequest.status).toBe('pending')
    })

    it('sends the seller to Messages to answer waiting bookings', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      wrapper.vm.error = ''
      wrapper.vm.item = ownerItem
      wrapper.vm.bookingRequests = [
        { id: 1, item_id: 1, requester_id: 2, requester: { id: 2, username: 'buyer1' }, status: 'pending', created_at: new Date().toISOString() },
        { id: 2, item_id: 1, requester_id: 3, requester: { id: 3, username: 'buyer2' }, status: 'pending', created_at: new Date().toISOString() }
      ]
      await flushPromises()

      expect(wrapper.text()).toContain('2 people have asked to book it')
      const answer = wrapper.findAll('button').find(b => b.text() === 'Answer in Messages')
      expect(answer).toBeTruthy()
      await answer.trigger('click')
      expect(openChat).toHaveBeenCalledWith()
      // No approving or declining on this page: that happens in the conversation
      expect(wrapper.find('[data-testid="approve-booking-btn"]').exists()).toBe(false)
    })
  })

  describe('Owner Messaging Interface', () => {
    it('should show message button for approved booking requests', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      const approvedRequests = [
        {
          id: 1,
          item_id: 1,
          requester_id: 2,
          requester: { id: 2, username: 'buyer1' },
          status: 'approved',
          message: 'Request message',
          created_at: new Date().toISOString()
        }
      ]

      wrapper.vm.error = ''
      wrapper.vm.item = ownerItem
      wrapper.vm.bookingRequests = approvedRequests
      await flushPromises()
      await wrapper.vm.$nextTick()

      const messageButton = wrapper.find('.message-approved-btn')
      expect(messageButton.exists()).toBe(true)
      expect(messageButton.text()).toContain('buyer1')
    })

    it('should not show message button for pending booking requests', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      const pendingRequests = [
        {
          id: 1,
          item_id: 1,
          requester_id: 2,
          requester: { id: 2, username: 'buyer1' },
          status: 'pending',
          message: 'Request message',
          created_at: new Date().toISOString()
        }
      ]

      wrapper.vm.error = ''
      wrapper.vm.item = ownerItem
      wrapper.vm.bookingRequests = pendingRequests
      await flushPromises()
      await wrapper.vm.$nextTick()

      const messageButton = wrapper.find('.message-approved-btn')
      expect(messageButton.exists()).toBe(false)

      const answer = wrapper.findAll('button').find(b => b.text() === 'Answer in Messages')
      expect(answer).toBeTruthy()
    })

    it('opens the floating chat with an approved requester', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      wrapper.vm.item = ownerItem
      wrapper.vm.bookingRequests = [
        {
          id: 1,
          item_id: 1,
          requester_id: 2,
          requester: { id: 2, username: 'buyer1' },
          status: 'approved',
          message: 'Request message',
          created_at: new Date().toISOString()
        }
      ]

      wrapper.vm.openStoreChatWithUser(2)
      await wrapper.vm.$nextTick()

      // The username comes from the bookingRequests array
      expect(openChat).toHaveBeenCalledWith({ itemId: ownerItem.id, otherUserId: 2, otherUserName: 'buyer1' })
    })
  })
})
