import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import StoreItemView from '@/views/store/StoreItemView.vue'

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

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(() => 'mock-token'),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn()
}
global.localStorage = localStorageMock

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

  const mockUser = {
    id: 1,
    username: 'buyer1'
  }

  beforeEach(() => {
    vi.clearAllMocks()
    pinia = createPinia()
    
    // Setup auth store mock
    const useAuthStore = () => ({
      user: mockUser,
      isAuthenticated: true
    })
    
    wrapper = mount(StoreItemView, {
      global: {
        plugins: [router, pinia],
        stubs: ['router-link']
      }
    })
  })

  describe('Booking Request Button', () => {
    it('should show "Book Now" button for non-owner users on active fixed-price items', async () => {
      // Mock successful item fetch
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockItem
      })

      // Mock no existing booking request
      fetch.mockResolvedValueOnce({
        ok: false,
        status: 404
      })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      // Should show booking request button
      const bookingButton = wrapper.find('[data-testid="booking-request-btn"]')
      expect(bookingButton.exists()).toBe(true)
      expect(bookingButton.text()).toContain('Book Now')
      expect(bookingButton.attributes('disabled')).toBeUndefined()
    })

    it('should not show booking button for item owner', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ownerItem
      })

      await wrapper.vm.loadItem()

      const bookingSection = wrapper.find('.booking-section')
      expect(bookingSection.exists()).toBe(false)
    })

    it('should not show booking button for inactive items', async () => {
      const inactiveItem = { ...mockItem, status: 'sold' }
      
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => inactiveItem
      })

      await wrapper.vm.loadItem()

      const bookingSection = wrapper.find('.booking-section')
      expect(bookingSection.exists()).toBe(false)
    })
  })

  describe('Booking Request Creation', () => {
    it('should send booking request when button is clicked', async () => {
      // Setup initial item load
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockItem
      })

      // Mock no existing booking request
      fetch.mockResolvedValueOnce({
        ok: false,
        status: 404
      })

      await wrapper.vm.loadItem()

      // Mock successful booking request creation
      const mockBookingRequest = {
        id: 1,
        item_id: 1,
        requester_id: 1,
        status: 'pending',
        message: `I'm interested in booking this item: ${mockItem.title}`,
        created_at: new Date().toISOString()
      }

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockBookingRequest
      })

      // Trigger booking request
      await wrapper.vm.sendBookingRequest()

      // Verify API call was made with correct data
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/v1/items/1/booking-request',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'Authorization': 'Bearer mock-token'
          }),
          body: JSON.stringify({
            message: `I'm interested in booking this item: ${mockItem.title}`
          })
        })
      )

      // Verify state updates
      expect(wrapper.vm.bookingRequest).toEqual(mockBookingRequest)
      expect(wrapper.vm.hasBookingRequest).toBe(true)
    })

    it('should handle booking request errors', async () => {
      // Setup initial item load
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockItem
      })

      fetch.mockResolvedValueOnce({
        ok: false,
        status: 404
      })

      await wrapper.vm.loadItem()

      // Mock error response
      fetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Cannot book your own item' })
      })

      // Spy on alert
      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

      await wrapper.vm.sendBookingRequest()

      expect(alertSpy).toHaveBeenCalledWith('Cannot book your own item')
      expect(wrapper.vm.hasBookingRequest).toBe(false)

      alertSpy.mockRestore()
    })
  })

  describe('Booking Request Status Display', () => {
    it('should show pending status correctly', async () => {
      const pendingBookingRequest = {
        id: 1,
        status: 'pending',
        created_at: new Date().toISOString()
      }

      // Setup component with existing booking request
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockItem
      })

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => pendingBookingRequest
      })

      await wrapper.vm.loadItem()

      // Should show pending status
      expect(wrapper.vm.bookingStatus).toBe('pending')
      expect(wrapper.vm.hasBookingRequest).toBe(true)
    })

    it('should show approved status with enhanced messaging info', async () => {
      const approvedBookingRequest = {
        id: 1,
        status: 'approved',
        created_at: new Date().toISOString()
      }

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockItem
      })

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => approvedBookingRequest
      })

      await wrapper.vm.loadItem()

      expect(wrapper.vm.bookingStatus).toBe('approved')
      expect(wrapper.vm.currentMessageLimit).toBe(10) // Increased limit after approval
    })

    it('should show rejected status', async () => {
      const rejectedBookingRequest = {
        id: 1,
        status: 'rejected',
        created_at: new Date().toISOString()
      }

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockItem
      })

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => rejectedBookingRequest
      })

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
        requester: { username: 'buyer1' },
        created_at: new Date().toISOString()
      }

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ownerItem
      })

      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => pendingBookingRequest
      })

      // Update component to simulate owner view
      wrapper.vm.item = ownerItem
      wrapper.vm.bookingRequest = pendingBookingRequest
      await wrapper.vm.$nextTick()

      // Should show owner management interface
      expect(wrapper.vm.item.seller.id).toBe(wrapper.vm.userId)
      expect(wrapper.vm.bookingRequest.status).toBe('pending')
    })

    it('should approve booking request', async () => {
      const approveBookingRequest = {
        id: 1,
        status: 'pending'
      }

      wrapper.vm.bookingRequest = approveBookingRequest

      // Mock successful approval
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'Booking request approved successfully' })
      })

      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

      await wrapper.vm.approveBookingRequest()

      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/v1/booking-requests/1/approve',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Authorization': 'Bearer mock-token'
          })
        })
      )

      expect(wrapper.vm.bookingRequest.status).toBe('approved')
      expect(alertSpy).toHaveBeenCalledWith(
        'Booking request approved! The requester can now message you with up to 10 messages.'
      )

      alertSpy.mockRestore()
    })

    it('should reject booking request with confirmation', async () => {
      const rejectBookingRequest = {
        id: 1,
        status: 'pending'
      }

      wrapper.vm.bookingRequest = rejectBookingRequest

      // Mock successful rejection
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'Booking request rejected successfully' })
      })

      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

      await wrapper.vm.rejectBookingRequest()

      expect(confirmSpy).toHaveBeenCalledWith('Are you sure you want to decline this booking request?')
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/v1/booking-requests/1/reject',
        expect.objectContaining({
          method: 'POST'
        })
      )

      expect(wrapper.vm.bookingRequest.status).toBe('rejected')
      expect(alertSpy).toHaveBeenCalledWith('Booking request declined.')

      confirmSpy.mockRestore()
      alertSpy.mockRestore()
    })
  })

  describe('Message Limit Integration', () => {
    it('should have 3 message limit before booking approval', () => {
      wrapper.vm.bookingRequest = { status: 'pending' }
      expect(wrapper.vm.currentMessageLimit).toBe(3)
    })

    it('should have 10 message limit after booking approval', () => {
      wrapper.vm.bookingRequest = { status: 'approved' }
      expect(wrapper.vm.currentMessageLimit).toBe(10)
    })

    it('should update message limits when booking status changes', async () => {
      // Start with pending
      wrapper.vm.bookingRequest = { id: 1, status: 'pending' }
      expect(wrapper.vm.currentMessageLimit).toBe(3)

      // Mock approval
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'approved' })
      })

      await wrapper.vm.approveBookingRequest()

      // Should update to 10
      expect(wrapper.vm.currentMessageLimit).toBe(10)
    })
  })

  describe('Owner Messaging Interface', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('should show message button for approved booking requests', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      const approvedBookingRequests = [
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

      // Mock item load
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ownerItem
      })

      // Mock booking requests load
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ booking_requests: approvedBookingRequests })
      })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      // Set booking requests manually for testing
      wrapper.vm.bookingRequests = approvedBookingRequests

      await wrapper.vm.$nextTick()

      // Should show message button for approved request
      const messageButton = wrapper.find('.message-approved-btn')
      expect(messageButton.exists()).toBe(true)
      expect(messageButton.text()).toContain('Message buyer1')
    })

    it('should not show message button for pending booking requests', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      const pendingBookingRequests = [
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

      // Mock item load
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ownerItem
      })

      // Mock booking requests load
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ booking_requests: pendingBookingRequests })
      })

      await router.push('/store/1')
      await wrapper.vm.loadItem()

      // Set booking requests manually for testing
      wrapper.vm.bookingRequests = pendingBookingRequests

      await wrapper.vm.$nextTick()

      // Should not show message button for pending request
      const messageButton = wrapper.find('.message-approved-btn')
      expect(messageButton.exists()).toBe(false)

      // Should show approve/decline buttons instead
      const approveButton = wrapper.find('[data-testid="approve-booking-btn"]')
      const rejectButton = wrapper.find('[data-testid="reject-booking-btn"]')
      expect(approveButton.exists()).toBe(true)
      expect(rejectButton.exists()).toBe(true)
    })

    it('should open chat modal when messaging approved requester', async () => {
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

      // Mock store messages API call
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          messages: [],
          messageCount: 0,
          messageLimit: 10,
          bookingStatus: 'approved'
        })
      })

      // Trigger the chat opening
      await wrapper.vm.openStoreChatWithUser(2)

      // Verify chat state
      expect(wrapper.vm.showChatModal).toBe(true)
      expect(wrapper.vm.chatRecipientId).toBe(2)
      expect(wrapper.vm.chatRecipientName).toContain('User 2')

      // Verify API call was made
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8082/api/v1/store-messages/1',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer mock-token'
          })
        })
      )
    })

    it('should set correct recipient when sending message as owner', async () => {
      const ownerItem = { ...mockItem, seller: { ...mockItem.seller, id: 1 } }
      wrapper.vm.item = ownerItem
      wrapper.vm.chatRecipientId = 2
      wrapper.vm.newMessage = 'Thanks for your interest!'

      // Mock successful message send
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: 1,
          store_item_id: 1,
          sender_id: 1,
          recipient_id: 2,
          content: 'Thanks for your interest!',
          created_at: new Date().toISOString()
        })
      })

      await wrapper.vm.sendMessage()

      // Verify message was sent to correct recipient
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8082/api/v1/store-messages',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            store_item_id: 1,
            recipient_id: 2,
            content: 'Thanks for your interest!'
          })
        })
      )
    })
  })
})