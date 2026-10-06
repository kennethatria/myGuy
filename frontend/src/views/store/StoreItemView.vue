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
          <NoPhoto v-else />
        </div>
        
        <div class="item-info-section">
          <StickyNote :seed="item.id" size="large" class="item-note">
            <template #header>
              <span v-if="item.status !== 'active'" class="note-status">{{ statusLabel }}</span>
              <h1 class="note-detail-headline">{{ item.title }}</h1>
              <p class="note-detail-body">{{ item.description }}</p>
            </template>
            <template #footer>
              <span>Posted {{ formatDate(item.created_at) }}</span>
              <span v-if="item.status === 'active' && item.deadline && expiryLabel(item.deadline)">
                {{ expiryLabel(item.deadline) }}
              </span>
            </template>
          </StickyNote>

          <router-link
            v-if="item.request"
            :to="{ name: 'store-request', params: { id: item.request.id } }"
            class="answers-request"
          >
            Listed for the request "{{ item.request.title }}"
          </router-link>
          
          <div class="seller-info">
            <h3>Seller</h3>
            <div class="seller-details">
              <span class="seller-name">{{ item.seller.name || item.seller.full_name || item.seller.username }}</span>
              <div class="seller-actions">
                <router-link
                  :to="{ name: 'user-profile', params: { id: String(item.seller.id) } }"
                  class="view-profile"
                >
                  View Profile
                </router-link>
                <!-- Show transaction complete badge if booking is completed/item received -->
                <div v-if="item.seller.id !== userId && hasCompletedBooking" class="transaction-complete-badge">
                  <i class="fas fa-check-circle"></i>
                  Transaction Complete
                </div>
                <!-- Only show message button if not own item and transaction not complete -->
                <button
                  v-else-if="item.seller.id !== userId"
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
              <p class="bid-increment">Minimum Increment: UGX {{ formatCurrency(item.min_bid_increment) }}</p>
              <p class="bid-count">{{ item.bid_count || 0 }} bids</p>
              <p v-if="biddingClosed" class="bid-closed">Bidding has closed.</p>

              <div v-if="item.seller.id !== userId && item.status === 'active' && !biddingClosed" class="bid-form">
                <input 
                  v-model="bidAmount" 
                  type="number" 
                  :min="minBidAmount" 
                  :step="item.min_bid_increment"
                  placeholder="Enter bid amount"
                />
                <button @click="placeBid" class="btn btn-primary">Place Bid</button>
              </div>
            </div>
            
            <div v-else class="fixed-price">
              <template v-if="listingPriceLabel(item)">
                <h3>Price</h3>
                <p class="price">{{ listingPriceLabel(item) }}</p>
              </template>
              
              <!-- Booking Request Section -->
              <!-- Open to bookings while active; once reserved, only the buyer
                   with a booking still follows it here -->
              <div
                v-if="item.seller.id !== userId && (item.status === 'active' || (item.status === 'reserved' && hasBookingRequest))"
                class="booking-section"
              >
                <div v-if="!hasBookingRequest" class="booking-request">
                  <button
                    @click="sendBookingRequest"
                    :disabled="loadingBookingRequest"
                    class="btn btn-primary btn-large"
                    data-testid="booking-request-btn"
                  >
                    {{ loadingBookingRequest ? 'Sending Request...' : 'Book Now' }}
                  </button>
                  <p class="booking-info">Ask to book it, then agree the price and pickup in chat</p>
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

                  <div v-else-if="bookingStatus === 'item_received'" class="status-item-received">
                    <i class="fas fa-box-open"></i>
                    <div>
                      <p><strong>Item Received</strong></p>
                      <p>Waiting for seller to confirm delivery. Go to Messages to complete the transaction.</p>
                    </div>
                  </div>

                  <div v-else-if="bookingStatus === 'completed'" class="status-completed">
                    <i class="fas fa-check-double"></i>
                    <div>
                      <p><strong>Transaction Completed!</strong></p>
                      <p>This transaction has been completed. You can rate your experience in Messages.</p>
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
              <p v-if="item.status === 'expired'" class="status-info">
                Nobody asked to book it within 24 hours. Repost it for another 24 hours, or remove it.
              </p>
              <p v-else class="status-info">Status: {{ statusLabel }}</p>
              <div v-if="item.status === 'expired'" class="owner-actions">
                <button class="btn btn-primary btn-sm" :disabled="ownerBusy" @click="repostItem">
                  {{ ownerBusy ? 'Reposting...' : 'Repost for 24 hours' }}
                </button>
                <button class="btn btn-outline btn-sm" :disabled="ownerBusy" @click="removeItem">Remove</button>
              </div>
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
                    @click="openStoreChatWithUser(request.requester!.id)"
                    class="btn btn-primary btn-sm message-approved-btn"
                  >
                    <i class="fas fa-comment"></i> Message {{ request.requester?.username }}
                  </button>
                </div>
                
                <div v-else-if="request.status === 'rejected'" class="booking-rejected">
                  <p class="rejected-text">✗ Declined</p>
                </div>
              </div>
            </div>
          </div>
          
          <!-- (the buyer holding the reservation follows it in their booking above) -->
          <div v-else-if="item.status !== 'active' && !(item.status === 'reserved' && hasBookingRequest)" class="item-status">
            <p class="status-message">
              {{ item.status === 'expired' ? 'This listing has come off the board.' : `This item is ${item.status}` }}
            </p>
          </div>
        </div>
      </div>
      
      <!-- Bid History -->
      <div v-if="item.is_auction && bids.length > 0" class="bid-history">
        <h3>Bid History</h3>
        <div class="bid-list">
          <div v-for="bid in bids" :key="bid.id" class="bid-item">
            <span class="bidder">{{ bid.bidder?.name || bid.bidder?.full_name || bid.bidder?.username || 'Unknown Bidder' }}</span>
            <span class="bid-amount">UGX {{ formatCurrency(bid.amount) }}</span>
            <span class="bid-time">{{ formatDate(bid.created_at) }}</span>
          </div>
        </div>
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
import { setPageTitle } from '@/utils/pageTitle';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useChatStore } from '@/stores/chat';
import BookingConfirmationModal from '@/components/BookingConfirmationModal.vue';
import NoPhoto from '@/components/NoPhoto.vue';
import StickyNote from '@/components/StickyNote.vue';
import { expiryLabel } from '@/utils/gigNote';
import { listingPriceLabel } from '@/utils/listingNote';
import config from '@/config';

// Type definitions
interface StoreItemImage {
  id: number;
  url: string;
}

interface Seller {
  id: number;
  username: string;
  name?: string;
  full_name?: string;
}

interface StoreItem {
  id: number;
  title: string;
  description: string;
  status: string;
  deadline?: string;
  bid_deadline?: string;
  request?: { id: number; title: string };
  price_type: string;
  fixed_price?: number;
  starting_bid?: number;
  current_bid?: number;
  min_bid_increment?: number;
  bid_count?: number;
  price?: number;
  is_auction?: boolean;
  seller_id: number;
  seller: Seller;
  images?: StoreItemImage[];
  created_at: string;
  updated_at: string;
}

interface Bidder {
  id: number;
  username: string;
  name?: string;
  full_name?: string;
}

interface Bid {
  id: number;
  amount: number;
  bidder_id: number;
  bidder?: Bidder;
  created_at: string;
}

interface Requester {
  id: number;
  username: string;
  name?: string;
}

interface BookingRequest {
  id: number;
  item_id: number;
  requester_id: number;
  requester?: Requester;
  status: string;
  message?: string;
  created_at: string;
  updated_at: string;
}

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const chatStore = useChatStore();

const item = ref<StoreItem | null>(null);
const bids = ref<Bid[]>([]);
const loading = ref(true);
const error = ref('');
const bidAmount = ref('');
const selectedImage = ref('');

// Booking-related variables
const bookingRequest = ref<BookingRequest | null>(null);
const bookingRequests = ref<BookingRequest[]>([]);
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
  // The first bid may match the starting bid; later ones must beat the current bid
  if (!item.value.current_bid) return item.value.starting_bid ?? 0;
  return item.value.current_bid + (item.value.min_bid_increment ?? 0);
});

// Auctions close when the note comes down
const biddingClosed = computed(() =>
  !!item.value?.bid_deadline && new Date(item.value.bid_deadline).getTime() <= Date.now()
);

const statusLabel = computed(() => {
  const status = item.value?.status ?? '';
  return status === 'expired' ? 'Expired' : status.charAt(0).toUpperCase() + status.slice(1);
});

// Seller actions on an expired note
const ownerBusy = ref(false);

async function repostItem() {
  ownerBusy.value = true;
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/repost`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || 'Could not repost the listing. Please try again.');
    await loadItem();
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not repost the listing. Please try again.');
  } finally {
    ownerBusy.value = false;
  }
}

async function removeItem() {
  if (!item.value || !confirm(`Remove "${item.value.title}" for good?`)) return;
  ownerBusy.value = true;
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || 'Could not remove the listing. Please try again.');
    router.push({ name: 'store' });
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Could not remove the listing. Please try again.');
  } finally {
    ownerBusy.value = false;
  }
}

// Booking computed properties
const bookingStatus = computed(() => {
  return bookingRequest.value?.status || null;
});

const hasCompletedBooking = computed(() => {
  if (!bookingRequest.value) return false;
  const status = bookingRequest.value.status;
  // Consider booking complete when item is received or fully completed
  return status === 'completed' || status === 'item_received';
});

async function loadItem() {
  try {
    loading.value = true;
    error.value = '';

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
    setPageTitle(item.value?.title);
    console.log('Item loaded successfully:', item.value);
    
    if (item.value?.is_auction) {
      await loadBids();
    }

    // Load booking request if user is involved
    await loadBookingRequest();

    // Check for messages if user is the owner
    if (item.value?.seller.id === userId.value) {
      await checkForMessages();
    }
  } catch (err) {
    console.error('Error loading item:', err);
    error.value = err instanceof Error ? err.message : 'Failed to load item';
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
  } catch {
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

function formatCurrency(amount: number | undefined): string {
  return new Intl.NumberFormat('en-UG', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount ?? 0);
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

// Every conversation opens in the floating chat. A buyer talks to the
// seller; the seller picks a buyer, or sees all their conversations.
function openStoreChat() {
  if (!item.value) return;
  chatStore.openChat({
    itemId: item.value.id,
    otherUserId: item.value.seller.id,
    otherUserName: item.value.seller.name || item.value.seller.full_name || item.value.seller.username
  });
}

function openStoreChatWithUser(recipientId: number) {
  if (!item.value) return;
  const requester = bookingRequests.value.find(req => req.requester?.id === recipientId);
  chatStore.openChat({ itemId: item.value.id, otherUserId: recipientId, otherUserName: requester?.requester?.username });
}

function openGeneralStoreChat() {
  chatStore.openChat();
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

.item-note {
  margin-bottom: 2rem;
}

.note-status {
  align-self: flex-start;
  padding: 0.15rem 0.6rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(31, 41, 55, 0.12);
}

.note-detail-headline {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1.2;
}

.note-detail-body {
  margin: 0;
  font-size: 1.15rem;
  line-height: 1.5;
  flex: 1;
}

.answers-request {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  margin: -1rem 0 1.5rem;
  color: var(--color-primary, #4f46e5);
  font-weight: 500;
  text-decoration: none;
}

.answers-request:hover,
.answers-request:focus-visible {
  text-decoration: underline;
}

.owner-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.bid-closed {
  font-weight: 600;
  color: #6b7280;
}

.seller-info h3,
.price-section h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #111827;
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

.transaction-complete-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #d1fae5;
  color: #065f46;
  border: 1px solid #10b981;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
}

.transaction-complete-badge i {
  font-size: 0.875rem;
  color: #10b981;
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

.btn-sm {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
}


/* Phones: thumb-sized targets */
@media (max-width: 768px) {
  .back-link {
    min-height: 44px;
  }

  .view-profile,
  .message-btn,
  .owner-actions .btn,
  .bid-form input,
  .bid-form button {
    min-height: 44px;
  }

  .owner-actions .btn {
    flex: 1;
  }
}

@media (max-width: 768px) {
  .item-content {
    grid-template-columns: 1fr;
  }

  .store-item-container {
    padding: 1rem;
  }

  /* The photo shouldn't push the details off the first screen */
  .item-image-section {
    min-height: 0;
    padding: 1rem;
  }

  .main-image {
    min-height: 200px;
  }

  .item-info-section {
    padding: 1.25rem;
  }

  .note-detail-headline {
    font-size: 1.4rem;
  }
  
  .bid-form {
    flex-direction: column;
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
  text-align: center;
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

.status-item-received {
  background: #dbeafe;
  border-color: #3b82f6;
  color: #1e40af;
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.status-completed {
  background: #d1fae5;
  border-color: #10b981;
  color: #065f46;
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.status-pending i,
.status-approved i,
.status-rejected i,
.status-item-received i,
.status-completed i {
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
  color: var(--color-primary);
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