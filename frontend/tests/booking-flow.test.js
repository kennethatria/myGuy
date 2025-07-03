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
    it('should show "Request Booking" button for non-owner users on active fixed-price items', async () => {
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
      expect(bookingButton.text()).toContain('Request Booking')
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
})