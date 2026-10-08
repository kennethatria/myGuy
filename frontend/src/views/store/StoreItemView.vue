<template>
  <div class="store-item-container">
    <div v-if="loading" class="loading">Loading...</div>
    
    <div v-else-if="error" class="error">
      {{ error }}
    </div>
    
    <div v-else-if="item" class="item-details">
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
              <span v-if="noteStatus" class="note-status">{{ noteStatus }}</span>
              <h1 class="note-detail-headline">{{ item.title }}</h1>
              <p class="note-detail-body">{{ item.description }}</p>
            </template>
            <template #footer>
              <router-link :to="{ name: 'user-profile', params: { id: String(item.seller.id) } }" class="note-seller">
                @{{ item.seller.username }}
              </router-link>
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
          
          <!-- Once you've booked, your conversation with the seller (it opens
               for typing when they approve), as "Message …" on a gig -->
          <button
            v-if="item.seller.id !== userId && hasBookingRequest"
            @click="openStoreChat"
            class="btn btn-outline message-btn"
          >
            Message {{ item.seller.username }}
          </button>

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
              
              <!-- Book while it's for sale; afterwards the note shows where your
                   booking stands and the chat takes it from there -->
              <div
                v-if="item.seller.id !== userId && item.status === 'active' && !hasBookingRequest"
                class="booking-section"
              >
                <div class="booking-request">
                  <button
                    @click="sendBookingRequest"
                    :disabled="loadingBookingRequest"
                    class="btn btn-primary btn-large"
                    data-testid="booking-request-btn"
                  >
                    {{ loadingBookingRequest ? 'Sending Request...' : 'Book Now' }}
                  </button>
                  <p v-if="bookingError" class="owner-error" role="alert">{{ bookingError }}</p>
                  <p class="booking-info">Ask to book it, then agree the price and pickup in chat</p>
                </div>
                
              </div>
            </div>
          </div>
          
          <div v-if="item.seller.id === userId" class="owner-section">
            <!-- The seller's own listing: repost or remove it, as with gigs and requests -->
            <p v-if="item.status === 'expired'" class="owner-note">
              Nobody asked to book it within 24 hours. Repost it for another 24 hours, or remove it.
            </p>
            <p v-else-if="item.status === 'reserved'" class="owner-note">
              Reserved for a buyer. To sell it to someone else, release the reservation in your conversation with them.
            </p>
            <!-- Bookings are answered in each buyer's conversation, as on gigs -->
            <p v-if="pendingBookings" class="owner-note">
              {{ pendingBookings }} {{ pendingBookings === 1 ? 'person has' : 'people have' }} asked to book it.
              Approve or decline in Messages.
            </p>
            <div v-if="pendingBookings || reservedFor || messageCount > 0" class="owner-actions">
              <button v-if="pendingBookings" @click="openGeneralStoreChat" class="btn btn-primary btn-sm">
                Answer in Messages
              </button>
              <button
                v-else-if="reservedFor"
                @click="openStoreChatWithUser(reservedFor.id)"
                class="btn btn-primary btn-sm message-approved-btn"
              >
                Message {{ reservedFor.username }}
              </button>
              <button v-else @click="openGeneralStoreChat" class="btn btn-outline btn-sm">
                View messages
              </button>
            </div>
            <p v-if="confirmingRemove" class="owner-note">
              Remove "{{ item.title }}" for good? Anyone waiting on a booking will be told.
            </p>
            <p v-if="ownerError" class="owner-error" role="alert">{{ ownerError }}</p>
            <div v-if="canRemove" class="owner-actions">
              <template v-if="confirmingRemove">
                <button class="btn btn-danger btn-sm" :disabled="ownerBusy" @click="removeItem">
                  {{ ownerBusy ? 'Removing...' : 'Yes, remove' }}
                </button>
                <button class="btn btn-outline btn-sm" :disabled="ownerBusy" @click="confirmingRemove = false">Keep it</button>
              </template>
              <template v-else>
                <button v-if="item.status === 'expired'" class="btn btn-primary btn-sm" :disabled="ownerBusy" @click="repostItem">
                  {{ ownerBusy ? 'Reposting...' : 'Repost for 24 hours' }}
                </button>
                <button class="btn btn-outline-danger btn-sm" :disabled="ownerBusy" @click="confirmingRemove = true">
                  Remove listing
                </button>
              </template>
            </div>
            

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

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { setPageTitle } from '@/utils/pageTitle';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useChatStore } from '@/stores/chat';
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

// Message indicators for owners
const messageCount = ref(0);
// The seller's view of bookings: how many wait for an answer, and who the
// item is reserved for (approved, or picked up but not yet confirmed)
const pendingBookings = computed(() => bookingRequests.value.filter(r => r.status === 'pending').length);
const reservedFor = computed(() =>
  bookingRequests.value.find(r => r.status === 'approved' || r.status === 'item_received')?.requester ?? null);
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

// What the note says about where things stand: your booking, if you have
// one, else the item itself (nothing while it's simply for sale)
const BOOKING_LABELS: Record<string, string> = {
  pending: 'Requested',
  approved: 'Booked for you',
  picked_up: 'Picked up',
  item_received: 'Collected',
  completed: 'Bought',
  rejected: 'Declined',
  released: 'Released'
};
const noteStatus = computed(() => {
  if (!item.value) return '';
  if (item.value.seller.id !== userId.value && bookingStatus.value) return BOOKING_LABELS[bookingStatus.value] ?? '';
  return item.value.status === 'active' ? '' : statusLabel.value;
});

const statusLabel = computed(() => {
  const status = item.value?.status ?? '';
  return status === 'expired' ? 'Expired' : status.charAt(0).toUpperCase() + status.slice(1);
});

// Seller actions on an expired note
const ownerBusy = ref(false);

// A live or expired listing can be removed (a reserved one is released first)
const canRemove = computed(() => item.value?.status === 'active' || item.value?.status === 'expired');
const confirmingRemove = ref(false);
const ownerError = ref('');

async function repostItem() {
  ownerBusy.value = true;
  ownerError.value = '';
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/repost`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || 'Could not repost the listing. Please try again.');
    await loadItem();
  } catch (err) {
    ownerError.value = err instanceof Error ? err.message : 'Could not repost the listing. Please try again.';
  } finally {
    ownerBusy.value = false;
  }
}

async function removeItem() {
  if (!item.value) return;
  ownerBusy.value = true;
  ownerError.value = '';
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || 'Could not remove the listing. Please try again.');
    router.push({ name: 'store' });
  } catch (err) {
    ownerError.value = err instanceof Error ? err.message : 'Could not remove the listing. Please try again.';
    confirmingRemove.value = false;
  } finally {
    ownerBusy.value = false;
  }
}

// Booking computed properties
const bookingStatus = computed(() => {
  return bookingRequest.value?.status || null;
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
const bookingError = ref('');

async function sendBookingRequest() {
  if (!item.value || loadingBookingRequest.value) return;

  loadingBookingRequest.value = true;
  bookingError.value = '';
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/booking-request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      // Booking is one tap, like applying for a gig: no message
      body: JSON.stringify({})
    });

    if (response.ok) {
      const request = await response.json();
      bookingRequest.value = request;
      hasBookingRequest.value = true;

      // Into the conversation, as applying for a gig does: the booking is
      // there, and the chat opens once the seller approves
      openStoreChat();
    } else {
      const error = await response.json().catch(() => ({}));
      bookingError.value = error.error || 'Failed to send booking request';
    }
  } catch (err) {
    console.error('Error sending booking request:', err);
    bookingError.value = 'Could not send the booking request. Please try again.';
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
  border-color: var(--color-primary);
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
  color: var(--color-primary);
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

.price-section h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #111827;
}

/* An outline button, as "Message …" on the gig page */
.message-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.2s;
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
  color: var(--color-primary);
  margin-bottom: 1rem;
}

.btn {
  padding: 0.75rem 1.5rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  /* Width and style only: the shared .btn and .btn-outline* give the colour */
  border-width: 1px;
  border-style: solid;
  font-size: 1rem;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover {
  background: var(--color-primary-dark);
}

.btn-large {
  padding: 1rem 2rem;
  font-size: 1.125rem;
}

.owner-note {
  margin: 0 0 0.75rem;
  color: #4b5563;
  font-size: 0.9rem;
}

.owner-error {
  margin: 0 0 0.75rem;
  color: #dc2626;
  font-size: 0.9rem;
}

/* The remove question sits apart from the buttons above it */
.owner-actions + .owner-note {
  margin-top: 0.75rem;
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

.owner-section {
  background: #e0f2fe;
  padding: 1rem;
  border-radius: 0.375rem;
  border: 1px solid #b3e5fc;
}

.message-approved-btn {
  margin-top: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-danger {
  background: #ef4444;
  color: white;
  border: none;
}

.btn-danger:hover {
  background: #dc2626;
}

@media (max-width: 768px) {
  .booking-request-card {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
  }
}

/* The seller, on the note: their profile is one tap away */
.note-seller {
  color: inherit;
  font-weight: 600;
  text-decoration: none;
}

.note-seller:hover,
.note-seller:focus-visible {
  text-decoration: underline;
}

.message-btn {
  width: 100%;
  margin-top: 1rem;
  min-height: 44px;
}
</style>
