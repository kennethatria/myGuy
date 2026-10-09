<template>
  <div class="detail-page">
    <p v-if="loading" class="detail-status-text" role="status">Loading...</p>

    <div v-else-if="error" class="detail-error" role="alert">{{ error }}</div>

    <template v-else-if="item">
      <div v-if="item.images && item.images.length > 0" class="gallery">
        <div class="gallery-main">
          <img :src="config.STORE_API_BASE_URL + currentImage" :alt="item.title" />
          <span v-if="item.images.length > 1" class="gallery-count">{{ currentIndex + 1 }} / {{ item.images.length }}</span>
        </div>
        <div v-if="item.images.length > 1" class="gallery-thumbs" role="group" aria-label="Photos">
          <button
            v-for="(image, index) in item.images"
            :key="image.id || index"
            type="button"
            :class="['gallery-thumb', { active: index === currentIndex }]"
            :aria-label="`Photo ${index + 1}`"
            :aria-pressed="index === currentIndex"
            @click="selectedImage = image.url"
          >
            <img :src="config.STORE_API_BASE_URL + image.url" alt="" />
          </button>
        </div>
      </div>
      <NoPhoto v-else />

      <DetailNote
        tone="sell"
        :seed="item.id"
        :status="noteStatus"
        :title="item.title"
        :body="item.description"
        :person="item.seller"
        :meta="postedMeta(item.created_at, item.deadline, item.status === 'active')"
      />

      <router-link
        v-if="item.request"
        :to="{ name: 'store-request', params: { id: item.request.id } }"
        class="detail-line answers-request"
      >
        Listed for the request "{{ item.request.title }}"
      </router-link>

      <!-- Auctions are gone; older ones can't be bid on or booked -->
      <p v-if="item.is_auction" class="detail-line">This older listing was an auction, and bidding has closed.</p>

      <!-- The seller's own listing: where it stands -->
      <template v-if="item.seller.id === userId">
        <p v-if="item.status === 'expired'" class="detail-line">
          Nobody asked to book it within 24 hours. Repost it for another 24 hours, or remove it.
        </p>
        <p v-else-if="item.status === 'reserved'" class="detail-line">
          Reserved for a buyer. To sell it to someone else, release the reservation in your conversation with them.
        </p>
        <!-- Bookings are answered in each buyer's conversation, as on gigs -->
        <p v-if="pendingBookings" class="detail-line">
          {{ pendingBookings }} {{ pendingBookings === 1 ? 'person has' : 'people have' }} asked to book it.
          Approve or decline in Messages.
        </p>
      </template>

      <ActionBar v-if="showBar">
        <!-- Someone else's listing: book it, then the chat takes it from there -->
        <template v-if="item.seller.id !== userId">
          <button v-if="hasBookingRequest" @click="openStoreChat" class="btn btn-outline">
            Message {{ item.seller.username }}
          </button>
          <template v-else>
            <p v-if="bookingError" class="bar-error" role="alert">{{ bookingError }}</p>
            <button
              @click="sendBookingRequest"
              :disabled="loadingBookingRequest"
              class="btn btn-primary"
              data-testid="booking-request-btn"
            >
              {{ loadingBookingRequest ? 'Sending...' : 'Book' }}
            </button>
            <p class="bar-caption">Ask to book it, then agree the price and pickup in chat</p>
          </template>
        </template>

        <!-- Your own listing: answer bookings, repost or remove it -->
        <template v-else>
          <p v-if="confirmingRemove" class="bar-message">
            Remove "{{ item.title }}" for good? Anyone waiting on a booking will be told.
          </p>
          <p v-if="ownerError" class="bar-error" role="alert">{{ ownerError }}</p>
          <div v-if="confirmingRemove" class="bar-row">
            <button class="btn btn-danger" :disabled="ownerBusy" @click="removeItem">
              {{ ownerBusy ? 'Removing...' : 'Yes, remove' }}
            </button>
            <button class="btn btn-outline" :disabled="ownerBusy" @click="confirmingRemove = false">Keep it</button>
          </div>
          <template v-else>
            <button v-if="pendingBookings" @click="openGeneralStoreChat" class="btn btn-primary">
              Answer in Messages
            </button>
            <button v-else-if="reservedFor" @click="openStoreChatWithUser(reservedFor.id)" class="btn btn-primary message-approved-btn">
              Message {{ reservedFor.username }}
            </button>
            <button v-else-if="messageCount > 0" @click="openGeneralStoreChat" class="btn btn-outline">
              View messages
            </button>
            <button v-if="item.status === 'expired'" class="btn btn-primary" :disabled="ownerBusy" @click="repostItem">
              {{ ownerBusy ? 'Reposting...' : 'Repost for 24 hours' }}
            </button>
            <button v-if="canRemove" class="btn btn-outline-danger" :disabled="ownerBusy" @click="confirmingRemove = true">
              Remove listing
            </button>
          </template>
        </template>
      </ActionBar>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { setPageTitle } from '@/utils/pageTitle';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useChatStore } from '@/stores/chat';
import NoPhoto from '@/components/NoPhoto.vue';
import DetailNote from '@/components/DetailNote.vue';
import ActionBar from '@/components/ActionBar.vue';
import { postedMeta } from '@/utils/gigNote';
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
  request?: { id: number; title: string };
  is_auction?: boolean;
  seller_id: number;
  seller: Seller;
  images?: StoreItemImage[];
  created_at: string;
  updated_at: string;
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
const loading = ref(true);
const error = ref('');
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


// Auctions close when the note comes down

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

// The photo shown large: the one tapped, else the first
const currentIndex = computed(() => {
  const index = item.value?.images?.findIndex((image) => image.url === selectedImage.value) ?? -1;
  return index < 0 ? 0 : index;
});
const currentImage = computed(() => item.value?.images?.[currentIndex.value]?.url ?? '');

// The bar shows your next step: book or message the seller, or (yours)
// answer bookings, repost or remove
const showBar = computed(() => {
  if (!item.value) return false;
  if (item.value.seller.id !== userId.value) {
    return hasBookingRequest.value || (item.value.status === 'active' && !item.value.is_auction);
  }
  return !!(pendingBookings.value || reservedFor.value || messageCount.value > 0 || canRemove.value ||
    item.value.status === 'expired' || ownerError.value);
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

<style scoped src="@/assets/detail.css"></style>

<style scoped>
.gallery {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gallery-main {
  position: relative;
  height: 210px;
  border-radius: 12px;
  overflow: hidden;
  background: #F3F4F6;
}

.gallery-main img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.gallery-count {
  position: absolute;
  right: 10px;
  bottom: 10px;
  padding: 3px 10px;
  border-radius: 10px;
  background: rgba(17, 24, 39, 0.55);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.gallery-thumbs {
  display: flex;
  gap: 8px;
}

.gallery-thumb {
  width: 64px;
  height: 48px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
}

.gallery-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.gallery-thumb.active {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.gallery-thumb:focus-visible {
  outline: 3px solid var(--accent-text);
  outline-offset: 2px;
}

.answers-request {
  color: var(--accent-text);
  font-weight: 600;
}









</style>
