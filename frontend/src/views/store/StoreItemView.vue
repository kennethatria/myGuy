<template>
  <div class="store-item-container">
    <div v-if="loading" class="loading">Loading...</div>
    
    <div v-else-if="error" class="error">
      {{ error }}
    </div>
    
    <div v-else-if="item" class="item-details">
      <div class="item-header">
        <router-link to="/store" class="back-link">
          <i class="fas fa-arrow-left"></i> Back to Store
        </router-link>
      </div>
      
      <div class="item-content">
        <div class="item-image-section">
          <div v-if="item.images && item.images.length > 0" class="image-gallery">
            <div class="main-image">
              <img :src="config.STORE_API_BASE_URL + (selectedImage || item.images[0].url)" :alt="item.title" />
            </div>
            <div v-if="item.images.length > 1" class="image-thumbnails">
              <div 
                v-for="(image, index) in item.images" 
                :key="image.id || index"
                class="thumbnail"
                :class="{ active: selectedImage === image.url || (!selectedImage && index === 0) }"
                @click="selectedImage = image.url"
              >
                <img :src="config.STORE_API_BASE_URL + image.url" :alt="`${item.title} ${index + 1}`" />
              </div>
            </div>
          </div>
          <img v-else :src="'/placeholder.png'" :alt="item.title" />
        </div>
        
        <div class="item-info-section">
          <h1>{{ item.title }}</h1>
          
          <div class="item-meta">
            <span class="category">{{ item.category }}</span>
            <span class="condition">{{ item.condition }}</span>
            <span class="posted-date">Posted {{ formatDate(item.created_at) }}</span>
          </div>
          
          <div class="item-description">
            <h3>Description</h3>
            <p>{{ item.description }}</p>
          </div>
          
          <div class="seller-info">
            <h3>Seller</h3>
            <div class="seller-details">
              <span class="seller-name">{{ item.seller.full_name }}</span>
              <div class="seller-actions">
                <router-link
                  :to="{ name: 'user-profile', params: { id: String(item.seller.id) } }"
                  class="view-profile"
                >
                  View Profile
                </router-link>
                <button 
                  v-if="item.seller.id !== userId"
                  @click="openStoreChat"
                  class="btn btn-outline btn-sm message-btn"
                >
                  <i class="fas fa-comment"></i> Message Seller
                </button>
              </div>
            </div>
          </div>
          

          <div class="price-section">
            <div v-if="item.is_auction" class="auction-info">
              <h3>Auction Details</h3>
              <p class="current-bid">Current Bid: UGX {{ formatCurrency(item.current_bid || item.starting_bid) }}</p>
              <p class="bid-increment">Minimum Increment: UGX {{ formatCurrency(item.bid_increment) }}</p>
              <p class="bid-count">{{ item.bid_count || 0 }} bids</p>
              
              <div v-if="item.seller.id !== userId" class="bid-form">
                <input 
                  v-model="bidAmount" 
                  type="number" 
                  :min="minBidAmount" 
                  :step="item.bid_increment"
                  placeholder="Enter bid amount"
                />
                <button @click="placeBid" class="btn btn-primary">Place Bid</button>
              </div>
            </div>
            
            <div v-else class="fixed-price">
              <h3>Price</h3>
              <p class="price">UGX {{ formatCurrency(item.price) }}</p>
              
              <!-- Booking Request Section -->
              <div v-if="item.seller.id !== userId && item.status === 'active'" class="booking-section">
                <div v-if="!hasBookingRequest" class="booking-request">
                  <button 
                    @click="sendBookingRequest" 
                    :disabled="loadingBookingRequest"
                    class="btn btn-primary btn-large"
                    data-testid="booking-request-btn"
                  >
                    {{ loadingBookingRequest ? 'Sending Request...' : 'Book Now' }}
                  </button>
                  <p class="booking-info">Send a booking request to the item owner</p>
                </div>
                
                <div v-else class="booking-status">
                  <div v-if="bookingStatus === 'pending'" class="status-pending">
                    <i class="fas fa-clock"></i>
                    <div>
                      <p><strong>Booking Request Sent</strong></p>
                      <p>Waiting for the owner to respond</p>
                    </div>
                  </div>
                  
                  <div v-else-if="bookingStatus === 'approved'" class="status-approved">
                    <i class="fas fa-check-circle"></i>
                    <div>
                      <p><strong>Booking Approved!</strong></p>
                      <p>You can now message the owner to coordinate pickup/delivery</p>
                    </div>
                  </div>
                  
                  <div v-else-if="bookingStatus === 'rejected'" class="status-rejected">
                    <i class="fas fa-times-circle"></i>
                    <div>
                      <p><strong>Booking Request Declined</strong></p>
                      <p>The owner has declined your booking request</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-if="item.seller.id === userId" class="owner-section">
            <div class="owner-status">
              <p class="owner-message">This is your listing</p>
              <p class="status-info">Status: {{ item.status }}</p>
            </div>
            
            <!-- General Messages for Owner -->
            <div v-if="messageCount > 0" class="owner-messages">
              <h4>Messages about this item</h4>
              <div class="message-summary">
                <p>{{ messageCount }} message{{ messageCount === 1 ? '' : 's' }} from interested buyers</p>
                <button 
                  @click="openGeneralStoreChat" 
                  class="btn btn-primary btn-sm message-view-btn"
                >
                  <i class="fas fa-comment"></i> View Messages
                </button>
              </div>
            </div>
            
            <!-- Booking Request Management for Owner -->
            <div v-if="bookingRequests.length > 0" class="booking-management">
              <h4>Booking Requests ({{ bookingRequests.length }})</h4>
              
              <div v-for="request in bookingRequests" :key="request.id" class="booking-request-card">
                <div class="requester-info">
                  <p><strong>Request from:</strong> {{ request.requester?.username || 'Unknown User' }}</p>
                  <p class="request-time">{{ formatDate(request.created_at) }}</p>
                  <p v-if="request.message" class="request-message">{{ request.message }}</p>
                  <span :class="`status-badge status-${request.status}`">{{ request.status.toUpperCase() }}</span>
                </div>
                
                <div v-if="request.status === 'pending'" class="booking-actions">
                  <button 
                    @click="approveBookingRequest(request)" 
                    class="btn btn-success btn-sm"
                    :disabled="loadingBookingRequest"
                    data-testid="approve-booking-btn"
                  >
                    Approve
                  </button>
                  <button 
                    @click="rejectBookingRequest(request)" 
                    class="btn btn-danger btn-sm"
                    :disabled="loadingBookingRequest"
                    data-testid="reject-booking-btn"
                  >
                    Decline
                  </button>
                </div>
                
                <div v-else-if="request.status === 'approved'" class="booking-approved">
                  <p class="approved-text">✓ Approved - You can now coordinate via messages</p>
                  <button 
                    @click="openStoreChatWithUser(request.requester.id)" 
                    class="btn btn-primary btn-sm message-approved-btn"
                  >
                    <i class="fas fa-comment"></i> Message {{ request.requester.username }}
                  </button>
                </div>
                
                <div v-else-if="request.status === 'rejected'" class="booking-rejected">
                  <p class="rejected-text">✗ Declined</p>
                </div>
              </div>
            </div>
          </div>
          
          <div v-else-if="item.status !== 'active'" class="item-status">
            <p class="status-message">This item is {{ item.status }}</p>
          </div>
        </div>
      </div>
      
      <!-- Bid History -->
      <div v-if="item.is_auction && bids.length > 0" class="bid-history">
        <h3>Bid History</h3>
        <div class="bid-list">
          <div v-for="bid in bids" :key="bid.id" class="bid-item">
            <span class="bidder">{{ bid.bidder.full_name }}</span>
            <span class="bid-amount">UGX {{ formatCurrency(bid.amount) }}</span>
            <span class="bid-time">{{ formatDate(bid.created_at) }}</span>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Store Chat Modal -->
    <!-- Chat Modal with new ChatWindow component -->
    <div v-if="showChatModal" class="chat-modal-overlay" @click="closeChatModal">
      <div class="chat-modal-container" @click.stop>
        <ChatWindow
          v-if="item"
          :conversation-id="Number(itemId)"
          conversation-type="store"
          :recipient-id="chatRecipientId || item.seller.id"
          :recipient-name="chatRecipientName"
          :conversation-title="`Message about: ${item.title}`"
          :show-close-button="true"
          @close="closeChatModal"
        />
      </div>
    </div>

    <!-- Booking Confirmation Modal -->
    <BookingConfirmationModal
      v-if="item"
      :is-open="showBookingConfirmationModal"
      :item-id="item.id"
      :item-title="item.title"
      :item-image="item.images?.[0]"
      :seller-id="item.seller_id"
      :seller-name="item.seller?.name || item.seller?.username"
      @close="showBookingConfirmationModal = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useChatStore } from '@/stores/chat';
import { format } from 'date-fns';
import ChatWindow from '@/components/ChatWindow.vue';
import BookingConfirmationModal from '@/components/BookingConfirmationModal.vue';
import config from '@/config';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const chatStore = useChatStore();

const item = ref(null);
const bids = ref([]);
const loading = ref(true);
const error = ref('');
const bidAmount = ref('');
const selectedImage = ref('');

// Chat-related variables (simplified - ChatWindow handles messaging)
const showChatModal = ref(false);
const chatRecipientId = ref(null);
const chatRecipientName = ref('');

// Booking-related variables
const bookingRequest = ref(null);
const bookingRequests = ref([]);
const hasBookingRequest = ref(false);
const loadingBookingRequest = ref(false);
const showBookingConfirmationModal = ref(false);

// Message indicators for owners
const messageCount = ref(0);
const hasUnreadMessages = ref(false);

const userId = computed(() => authStore.user?.id);
const itemId = computed(() => route.params.id);

const minBidAmount = computed(() => {
  if (!item.value?.is_auction) return 0;
  const currentBid = item.value.current_bid || item.value.starting_bid;
  return currentBid + item.value.bid_increment;
});

// Booking computed properties
const bookingStatus = computed(() => {
  return bookingRequest.value?.status || null;
});

async function loadItem() {
  try {
    loading.value = true;
    
    // Validate itemId
    if (!itemId.value || isNaN(Number(itemId.value))) {
      throw new Error('Invalid item ID');
    }
    
    console.log('Loading item with ID:', itemId.value);
    const apiUrl = `${config.STORE_API_URL}/items/${itemId.value}`;
    console.log('API URL:', apiUrl);
    
    const response = await fetch(apiUrl, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    });
    
    console.log('Response status:', response.status);
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      console.error('API Error:', errorData);
      throw new Error(errorData.error || `HTTP ${response.status}: Failed to load item`);
    }
    
    item.value = await response.json();
    console.log('Item loaded successfully:', item.value);
    
    if (item.value.is_auction) {
      await loadBids();
    }
    
    // Load booking request if user is involved
    await loadBookingRequest();
    
    // Check for messages if user is the owner
    if (item.value.seller.id === userId.value) {
      await checkForMessages();
    }
  } catch (err) {
    console.error('Error loading item:', err);
    error.value = err.message;
  } finally {
    loading.value = false;
  }
}

async function loadBids() {
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/bids`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    });

    if (response.ok) {
      bids.value = await response.json();
      console.log('Bids loaded:', bids.value);
    } else {
      console.error('Failed to load bids, status:', response.status);
    }
  } catch (err) {
    console.error('Error loading bids:', err);
  }
}

async function loadBookingRequest() {
  if (!item.value || !userId.value) return;
  
  try {
    // Check if user is the item owner
    if (item.value.seller.id === userId.value) {
      // Load all booking requests for item owners
      const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/booking-requests`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        bookingRequests.value = data.booking_requests || [];
        // Set the first pending request as the primary one for backwards compatibility
        const pendingRequest = bookingRequests.value.find(req => req.status === 'pending');
        bookingRequest.value = pendingRequest || bookingRequests.value[0] || null;
        hasBookingRequest.value = bookingRequests.value.length > 0;
      } else {
        console.error('Failed to load booking requests, status:', response.status);
      }
    } else {
      // Load user's specific booking request for non-owners
      const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/booking-request`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        bookingRequest.value = data.booking_request;
        hasBookingRequest.value = bookingRequest.value !== null;
      } else if (response.status === 404) {
        // No booking request exists
        bookingRequest.value = null;
        hasBookingRequest.value = false;
      } else {
        console.error('Failed to load booking request, status:', response.status);
      }
    }
  } catch (err) {
    console.error('Error loading booking request:', err);
  }
}

async function placeBid() {
  if (!bidAmount.value || parseFloat(bidAmount.value) < minBidAmount.value) {
    alert(`Minimum bid amount is UGX ${formatCurrency(minBidAmount.value)}`);
    return;
  }
  
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/bids`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ amount: parseFloat(bidAmount.value) })
    });
    
    if (response.ok) {
      await loadItem();
      await loadBids();
      bidAmount.value = '';
    } else {
      const error = await response.json();
      alert(error.error || 'Failed to place bid');
    }
  } catch (err) {
    alert('Error placing bid');
  }
}

// Booking request functions
async function sendBookingRequest() {
  if (!item.value || loadingBookingRequest.value) return;
  
  loadingBookingRequest.value = true;
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/booking-request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        message: `I'm interested in booking this item: ${item.value.title}`
      })
    });
    
    if (response.ok) {
      const request = await response.json();
      bookingRequest.value = request;
      hasBookingRequest.value = true;

      // Show confirmation modal instead of redirect
      showBookingConfirmationModal.value = true;
    } else {
      const error = await response.json();
      alert(error.error || 'Failed to send booking request');
    }
  } catch (err) {
    console.error('Error sending booking request:', err);
    alert('Error sending booking request');
  } finally {
    loadingBookingRequest.value = false;
  }
}

async function approveBookingRequest(request = bookingRequest.value) {
  if (!request || loadingBookingRequest.value) return;
  
  loadingBookingRequest.value = true;
  try {
    const response = await fetch(`${config.STORE_API_URL}/booking-requests/${request.id}/approve`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    });
    
    if (response.ok) {
      request.status = 'approved';
      // Update both arrays
      if (bookingRequest.value && bookingRequest.value.id === request.id) {
        bookingRequest.value.status = 'approved';
      }
      alert('Booking request approved! The requester can now message you.');
    } else {
      const error = await response.json();
      alert(error.error || 'Failed to approve booking request');
    }
  } catch (err) {
    console.error('Error approving booking request:', err);
    alert('Error approving booking request');
  } finally {
    loadingBookingRequest.value = false;
  }
}

async function rejectBookingRequest(request = bookingRequest.value) {
  if (!request || loadingBookingRequest.value) return;
  
  if (!confirm('Are you sure you want to decline this booking request?')) {
    return;
  }
  
  loadingBookingRequest.value = true;
  try {
    const response = await fetch(`${config.STORE_API_URL}/booking-requests/${request.id}/reject`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    });
    
    if (response.ok) {
      request.status = 'rejected';
      // Update both arrays
      if (bookingRequest.value && bookingRequest.value.id === request.id) {
        bookingRequest.value.status = 'rejected';
      }
      alert('Booking request declined.');
    } else {
      const error = await response.json();
      alert(error.error || 'Failed to decline booking request');
    }
  } catch (err) {
    console.error('Error declining booking request:', err);
    alert('Error declining booking request');
  } finally {
    loadingBookingRequest.value = false;
  }
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-UG', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  if (diff < 86400000) { // Less than 24 hours
    const hours = Math.floor(diff / 3600000);
    if (hours < 1) return 'just now';
    return `${hours}h ago`;
  }
  
  if (diff < 604800000) { // Less than 7 days
    const days = Math.floor(diff / 86400000);
    return `${days}d ago`;
  }
  
  return date.toLocaleDateString();
}

function formatMessageTime(dateString: string): string {
  try {
    const date = new Date(dateString);
    return format(date, 'MMM d, h:mm a');
  } catch (error) {
    return 'Unknown time';
  }
}

// Chat functions (simplified - ChatWindow component now handles messaging logic)
function openStoreChat() {
  // Set recipient info and show modal - ChatWindow handles socket connection and loading messages
  chatRecipientId.value = item.value.seller.id;
  chatRecipientName.value = item.value.seller.full_name || item.value.seller.username;
  showChatModal.value = true;
}

function closeChatModal() {
  showChatModal.value = false;
}

function openStoreChatWithUser(recipientId: number) {
  // For sellers messaging a specific buyer
  const requester = bookingRequests.value.find(req => req.requester.id === recipientId);
  chatRecipientId.value = recipientId;
  chatRecipientName.value = requester?.requester?.username || `User ${recipientId}`;
  showChatModal.value = true;
}

function openGeneralStoreChat() {
  // For owners to view all messages about their item - use seller as recipient
  chatRecipientId.value = item.value.seller.id;
  chatRecipientName.value = item.value.seller.full_name || item.value.seller.username;
  showChatModal.value = true;
}

async function checkForMessages() {
  if (!item.value) return;

  try {
    // Use chatStore to get store messages
    const messages = chatStore.getStoreMessages(Number(itemId.value));
    messageCount.value = messages.length;
    hasUnreadMessages.value = messages.some(msg => !msg.is_read && msg.sender_id !== userId.value);
  } catch (error) {
    console.error('Error checking for messages:', error);
  }
}

onMounted(() => {
  loadItem();
});
</script>

<style scoped>
.store-item-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.loading, .error {
  text-align: center;
  padding: 4rem;
  font-size: 1.125rem;
  color: #6b7280;
}

.error {
  color: #ef4444;
}

.item-header {
  margin-bottom: 2rem;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  color: #4F46E5;
  text-decoration: none;
  font-weight: 500;
}

.back-link:hover {
  color: #4338CA;
}

.item-content {
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
}

.item-image-section {
  background: #f3f4f6;
  min-height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.item-image-section > img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.image-gallery {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.main-image {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.main-image img {
  max-width: 100%;
  max-height: 400px;
  object-fit: contain;
}

.image-thumbnails {
  display: flex;
  gap: 0.5rem;
  justify-content: center;
}

.thumbnail {
  width: 80px;
  height: 80px;
  border: 2px solid transparent;
  border-radius: 0.375rem;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s;
}

.thumbnail:hover {
  border-color: #e5e7eb;
}

.thumbnail.active {
  border-color: #4F46E5;
}

.thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.item-info-section {
  padding: 2rem;
}

.item-info-section h1 {
  font-size: 2rem;
  font-weight: 600;
  margin-bottom: 1rem;
}

.item-meta {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.category, .condition, .posted-date {
  padding: 0.25rem 0.75rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  background: #f3f4f6;
  color: #374151;
}

.item-description {
  margin-bottom: 2rem;
}

.item-description h3,
.seller-info h3,
.price-section h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #111827;
}

.item-description p {
  color: #6b7280;
  line-height: 1.6;
}

.seller-info {
  margin-bottom: 2rem;
  padding-bottom: 2rem;
  border-bottom: 1px solid #e5e7eb;
}

.seller-details {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.seller-name {
  font-weight: 500;
  color: #374151;
}

.seller-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.view-profile {
  color: #4F46E5;
  text-decoration: none;
  font-size: 0.875rem;
  padding: 0.5rem 1rem;
  border: 1px solid #4F46E5;
  border-radius: 0.375rem;
  transition: all 0.2s;
}

.view-profile:hover {
  color: #4338CA;
  border-color: #4338CA;
  background-color: #f8fafc;
}

.message-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: #10b981;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.message-btn:hover {
  background: #059669;
}

.message-btn i {
  font-size: 0.875rem;
}

.price-section {
  margin-bottom: 2rem;
}

.auction-info p {
  margin-bottom: 0.5rem;
  color: #374151;
}

.current-bid {
  font-size: 1.5rem;
  font-weight: 600;
  color: #059669;
}

.bid-form {
  display: flex;
  gap: 1rem;
  margin-top: 1rem;
}

.bid-form input {
  flex: 1;
  padding: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  font-size: 1rem;
}

.fixed-price .price {
  font-size: 2rem;
  font-weight: 600;
  color: #4F46E5;
  margin-bottom: 1rem;
}

.btn {
  padding: 0.75rem 1.5rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
  font-size: 1rem;
}

.btn-primary {
  background: #4F46E5;
  color: white;
}

.btn-primary:hover {
  background: #4338CA;
}

.btn-large {
  padding: 1rem 2rem;
  font-size: 1.125rem;
}

.item-status {
  background: #fef3c7;
  padding: 1rem;
  border-radius: 0.375rem;
  text-align: center;
}

.status-message {
  color: #92400e;
  font-weight: 500;
}

.owner-status {
  background: #e0f2fe;
  padding: 1rem;
  border-radius: 0.375rem;
  text-align: center;
  border: 1px solid #b3e5fc;
}

.owner-message {
  color: #0277bd;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.status-info {
  color: #0288d1;
  font-size: 0.875rem;
  margin: 0;
}

.bid-history {
  margin-top: 2rem;
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  padding: 2rem;
}

.bid-history h3 {
  margin-bottom: 1rem;
}

.bid-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.bid-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  background: #f9fafb;
  border-radius: 0.375rem;
}

.bidder {
  font-weight: 500;
  color: #374151;
}

.bid-amount {
  font-weight: 600;
  color: #059669;
}

.bid-time {
  font-size: 0.875rem;
  color: #6b7280;
}

/* Chat Modal Styles */
.chat-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.chat-modal {
  background: white;
  border-radius: 0.5rem;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e5e7eb;
}

.chat-header h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #6b7280;
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.25rem;
  transition: background-color 0.2s;
}

.close-btn:hover {
  background-color: #f3f4f6;
  color: #374151;
}

.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.chat-messages {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
  max-height: 400px;
}

.no-messages {
  text-align: center;
  color: #6b7280;
  padding: 2rem;
}

.no-messages p {
  margin: 0.5rem 0;
}

.message-limit {
  font-size: 0.875rem;
  color: #059669;
  font-weight: 500;
}

.message {
  margin-bottom: 1rem;
  padding: 0.75rem;
  border-radius: 0.5rem;
  background: #f9fafb;
}

.message.own-message {
  background: #dbeafe;
  margin-left: 2rem;
}

.message.own-message .message-content {
  color: #1e40af;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.sender {
  font-weight: 600;
  color: #374151;
  font-size: 0.875rem;
}

.timestamp {
  font-size: 0.75rem;
  color: #6b7280;
}

.message-content {
  color: #111827;
  line-height: 1.5;
  white-space: pre-wrap;
}

.chat-input-section {
  border-top: 1px solid #e5e7eb;
  padding: 1rem;
}

.chat-input {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.chat-input textarea {
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  padding: 0.75rem;
  font-size: 0.875rem;
  resize: vertical;
  min-height: 80px;
  font-family: inherit;
}

.chat-input textarea:focus {
  outline: none;
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.input-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.message-count {
  font-size: 0.75rem;
  color: #6b7280;
}

.btn-sm {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
}

.message-limit-reached {
  text-align: center;
  padding: 1.5rem;
  color: #059669;
}

.message-limit-reached i {
  margin-right: 0.5rem;
}

.suggestion {
  font-size: 0.875rem;
  color: #6b7280;
  margin-top: 0.5rem;
}

@media (max-width: 768px) {
  .item-content {
    grid-template-columns: 1fr;
  }
  
  .bid-form {
    flex-direction: column;
  }

  .chat-modal {
    width: 95%;
    max-height: 90vh;
  }
  
  .message.own-message {
    margin-left: 1rem;
  }

  .seller-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .view-profile,
  .message-btn {
    text-align: center;
    justify-content: center;
  }
}

/* Booking Functionality Styles */
.booking-section {
  margin-top: 1rem;
}

.booking-request {
  text-align: center;
}

.booking-info {
  font-size: 0.875rem;
  color: #6b7280;
  margin-top: 0.5rem;
}

.booking-status {
  margin-top: 1rem;
  padding: 1rem;
  border-radius: 0.5rem;
  border: 1px solid;
}

.status-pending {
  background: #fef3c7;
  border-color: #fbbf24;
  color: #92400e;
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.status-approved {
  background: #d1fae5;
  border-color: #10b981;
  color: #065f46;
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.status-rejected {
  background: #fee2e2;
  border-color: #f87171;
  color: #991b1b;
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.status-pending i,
.status-approved i,
.status-rejected i {
  font-size: 1.25rem;
  margin-top: 0.125rem;
}

.message-limit-info {
  font-size: 0.75rem;
  color: #065f46;
  font-weight: 500;
  margin-top: 0.25rem;
}

.owner-section {
  background: #e0f2fe;
  padding: 1rem;
  border-radius: 0.375rem;
  border: 1px solid #b3e5fc;
}

.booking-management {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #b3e5fc;
}

.booking-management h4 {
  margin: 0 0 0.75rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: #0277bd;
}

.booking-request-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  padding: 1rem;
  margin-bottom: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.requester-info {
  flex: 1;
}

.requester-info p {
  margin: 0 0 0.25rem 0;
}

.request-time {
  font-size: 0.75rem;
  color: #6b7280;
}

.request-message {
  font-size: 0.875rem;
  color: #4b5563;
  font-style: italic;
  margin: 0.5rem 0;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  margin-top: 0.5rem;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-approved {
  background: #d1fae5;
  color: #065f46;
}

.status-rejected {
  background: #fee2e2;
  color: #991b1b;
}

.booking-approved {
  margin-top: 1rem;
}

.approved-text {
  color: #059669;
  font-weight: 500;
  margin: 0;
}

.booking-rejected {
  margin-top: 1rem;
}

.rejected-text {
  color: #dc2626;
  font-weight: 500;
  margin: 0;
}

.message-approved-btn {
  margin-top: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.booking-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-success {
  background: #10b981;
  color: white;
  border: none;
}

.btn-success:hover {
  background: #059669;
}

.btn-danger {
  background: #ef4444;
  color: white;
  border: none;
}

.btn-danger:hover {
  background: #dc2626;
}

.booking-approved-owner {
  margin-top: 1rem;
  padding: 1rem;
  background: #d1fae5;
  border: 1px solid #10b981;
  border-radius: 0.375rem;
  color: #065f46;
}

.booking-approved-owner h4 {
  margin: 0 0 0.5rem 0;
  color: #065f46;
}

.limit-info {
  font-size: 0.75rem;
  color: #059669;
}

/* Owner Messages Styles */
.owner-messages {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #b3e5fc;
}

.owner-messages h4 {
  margin: 0 0 0.75rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: #0277bd;
}

.message-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: white;
  padding: 0.75rem;
  border-radius: 0.375rem;
  border: 1px solid #e5e7eb;
}

.message-summary p {
  margin: 0;
  color: #374151;
  font-size: 0.875rem;
}

.message-view-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: #4f46e5;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.message-view-btn:hover {
  background: #4338ca;
}

.message-view-btn i {
  font-size: 0.875rem;
}

@media (max-width: 768px) {
  .booking-request-card {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
  }

  .booking-actions {
    justify-content: center;
  }
}

/* Success Message Styles */
.success-message {
  background: #f0f9f0;
  border: 1px solid #c3e6c3;
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1rem;
}

.success-content {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.success-content i {
  color: #28a745;
  font-size: 1.25rem;
  flex-shrink: 0;
  margin-top: 0.125rem;
}

.success-content p {
  margin: 0 0 0.5rem 0;
  color: #155724;
}

.success-content p:last-of-type {
  margin-bottom: 0;
}

/* Conversation Starter Styles */
.conversation-starters {
  margin-top: 1.5rem;
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.starter-label {
  font-size: 0.875rem;
  color: #6c757d;
  margin-bottom: 0.75rem;
  font-weight: 500;
}

.starter-buttons {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.starter-btn {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  padding: 0.75rem;
  text-align: left;
  color: #495057;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.starter-btn:hover {
  background: #e9ecef;
  border-color: #adb5bd;
  transform: translateY(-1px);
}

.starter-btn:active {
  transform: translateY(0);
}

/* Enhanced No Messages Styling */
.no-messages {
  text-align: center;
  padding: 2rem 1rem;
  color: #6c757d;
}

.no-messages i {
  font-size: 1.5rem;
  color: #adb5bd;
  margin-right: 0.5rem;
}

.no-messages p:first-child {
  font-size: 1.1rem;
  font-weight: 500;
  color: #495057;
  margin-bottom: 0.5rem;
}

.message-limit {
  font-size: 0.875rem;
  color: #6c757d;
  margin-bottom: 1rem !important;
}

/* Button Link Style */
.btn-link {
  color: #007bff;
  text-decoration: none;
  background: none;
  border: none;
  padding: 0;
  font-size: 0.875rem;
  cursor: pointer;
}

.btn-link:hover {
  color: #0056b3;
  text-decoration: underline;
}

.btn-link i {
  margin-right: 0.375rem;
}
</style>