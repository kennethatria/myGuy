<template>
  <div class="booking-message" :class="messageTypeClass">
    <div class="booking-icon">
      <i :class="iconClass"></i>
    </div>

    <div class="booking-content">
      <!-- Booking Request -->
      <div v-if="message.message_type === 'booking_request'" class="booking-request">
        <div class="booking-header">
          <h4>Booking Request</h4>
          <span :class="statusBadgeClass">{{ statusText }}</span>
        </div>

        <div class="item-details">
          <img
            v-if="message.metadata?.item_image"
            :src="getImageUrl(message.metadata.item_image)"
            :alt="message.metadata?.item_title"
            class="item-thumbnail"
          />
          <div class="item-info">
            <p class="item-title">{{ message.metadata?.item_title || 'Store Item' }}</p>
            <p class="requester">
              {{ isOwnMessage ? 'You requested this item' : `${senderName} wants to book this item` }}
            </p>
          </div>
        </div>

        <!-- Display the user's message -->
        <div v-if="message.content" class="booking-message-text">
          <p>{{ message.content }}</p>
        </div>

        <!-- Action Buttons for Seller: Approve/Decline (pending status) -->
        <div
          v-if="!isOwnMessage && message.metadata?.status === 'pending'"
          class="booking-actions"
        >
          <button
            @click="handleApprove"
            class="btn-approve"
            :disabled="isProcessing"
          >
            <i class="fas fa-check"></i> Approve
          </button>
          <button
            @click="handleDecline"
            class="btn-decline"
            :disabled="isProcessing"
          >
            <i class="fas fa-times"></i> Decline
          </button>
        </div>

        <!-- Action Button for Buyer: Confirm Receipt (approved status) -->
        <div
          v-else-if="isOwnMessage && message.metadata?.status === 'approved'"
          class="booking-actions"
        >
          <button
            @click="handleConfirmReceived"
            class="btn-confirm-received"
            :disabled="isProcessing"
          >
            <i class="fas fa-box-check"></i> I Received Item
          </button>
        </div>

        <!-- Action Button for Seller: Confirm Delivery (item_received status) -->
        <div
          v-else-if="!isOwnMessage && message.metadata?.status === 'item_received'"
          class="booking-actions"
        >
          <button
            @click="handleConfirmDelivery"
            class="btn-confirm-delivery"
            :disabled="isProcessing"
          >
            <i class="fas fa-check-circle"></i> Confirm Delivery
          </button>
        </div>

        <!-- Status Messages -->
        <div v-else class="booking-status">
          <!-- Pending: Waiting for seller -->
          <p v-if="message.metadata?.status === 'pending' && isOwnMessage" class="pending">
            ⏳ Waiting for seller response...
          </p>

          <!-- Approved: Waiting for buyer to receive -->
          <p v-else-if="message.metadata?.status === 'approved' && !isOwnMessage" class="approved">
            ✅ Booking approved - Waiting for buyer to confirm receipt
          </p>

          <!-- Item Received: Waiting for seller confirmation -->
          <p v-else-if="message.metadata?.status === 'item_received' && isOwnMessage" class="item-received">
            📦 Item received - Waiting for seller to confirm delivery
          </p>

          <!-- Completed -->
          <p v-else-if="message.metadata?.status === 'completed'" class="completed">
            ✅ Transaction completed!
          </p>

          <!-- Declined -->
          <p v-else-if="message.metadata?.status === 'rejected'" class="declined">
            ❌ Booking declined
          </p>
        </div>

        <!-- Rating Section (only show when completed) -->
        <div v-if="message.metadata?.status === 'completed'" class="rating-section">
          <!-- Buyer's Rating of Seller -->
          <div v-if="isOwnMessage && !hasRated" class="rating-input">
            <h5>Rate your experience with the seller</h5>
            <div class="star-rating">
              <span
                v-for="star in 5"
                :key="star"
                @click="selectRating(star)"
                @mouseenter="hoverRating = star"
                @mouseleave="hoverRating = 0"
                class="star"
                :class="{ filled: star <= (hoverRating || selectedRating) }"
              >
                ★
              </span>
            </div>
            <textarea
              v-model="reviewText"
              placeholder="Share your experience (optional)"
              class="review-input"
              rows="2"
            ></textarea>
            <button
              @click="submitRating"
              :disabled="!selectedRating || isProcessing"
              class="btn-submit-rating"
            >
              Submit Rating
            </button>
          </div>

          <!-- Seller's Rating of Buyer -->
          <div v-else-if="!isOwnMessage && !hasRated" class="rating-input">
            <h5>Rate your experience with the buyer</h5>
            <div class="star-rating">
              <span
                v-for="star in 5"
                :key="star"
                @click="selectRating(star)"
                @mouseenter="hoverRating = star"
                @mouseleave="hoverRating = 0"
                class="star"
                :class="{ filled: star <= (hoverRating || selectedRating) }"
              >
                ★
              </span>
            </div>
            <textarea
              v-model="reviewText"
              placeholder="Share your experience (optional)"
              class="review-input"
              rows="2"
            ></textarea>
            <button
              @click="submitRating"
              :disabled="!selectedRating || isProcessing"
              class="btn-submit-rating"
            >
              Submit Rating
            </button>
          </div>

          <!-- Display Submitted Rating -->
          <div v-else-if="hasRated" class="rating-display">
            <div class="rating-submitted">
              <span class="rating-label">{{ isOwnMessage ? 'You rated:' : 'They rated you:' }}</span>
              <div class="star-display">
                <span v-for="star in 5" :key="star" class="star" :class="{ filled: star <= displayedRating }">
                  ★
                </span>
              </div>
              <p v-if="displayedReview" class="review-text">{{ displayedReview }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Status Update Messages -->
      <div v-else class="booking-status-update">
        <p>{{ message.content }}</p>
      </div>

      <span class="timestamp">{{ formatTime(message.created_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue';
import { useUserStore } from '@/stores/user';
import config from '@/config';
import type { Message } from '@/stores/messages';

const props = defineProps<{
  message: Message;
  isOwnMessage: boolean;
}>();

const emit = defineEmits<{
  bookingAction: [bookingId: number, action: 'approve' | 'decline' | 'confirm-received' | 'confirm-delivery' | 'rate-seller' | 'rate-buyer', rating?: number, review?: string];
}>();

const userStore = useUserStore();
const isProcessing = ref(false);
const processingTimeout = ref<number | null>(null);

// Rating state
const selectedRating = ref(0);
const hoverRating = ref(0);
const reviewText = ref('');

const senderName = computed(() => {
  if (props.message.sender?.username) {
    return props.message.sender.username;
  }
  if (props.message.sender_id) {
    const user = userStore.getUserById(props.message.sender_id);
    if (user) return user.username;
  }
  return 'Unknown User';
});

const messageTypeClass = computed(() => {
  return `message-type-${props.message.message_type}`;
});

const iconClass = computed(() => {
  switch (props.message.message_type) {
    case 'booking_request':
      return 'fas fa-calendar-check';
    case 'booking_approved':
      return 'fas fa-check-circle';
    case 'booking_declined':
      return 'fas fa-times-circle';
    case 'booking_item_received':
      return 'fas fa-box-check';
    case 'booking_completed':
      return 'fas fa-check-double';
    default:
      return 'fas fa-info-circle';
  }
});

const statusText = computed(() => {
  const status = props.message.metadata?.status;
  if (status === 'pending') return 'Pending';
  if (status === 'approved') return 'Approved';
  if (status === 'rejected') return 'Declined';
  if (status === 'item_received') return 'Item Received';
  if (status === 'completed') return 'Completed';
  return '';
});

const statusBadgeClass = computed(() => {
  const status = props.message.metadata?.status;
  return `status-badge status-${status}`;
});

// Check if user has already rated
const hasRated = computed(() => {
  if (props.isOwnMessage) {
    // Buyer checking if they rated the seller
    return props.message.metadata?.buyer_rating !== undefined && props.message.metadata?.buyer_rating !== null;
  } else {
    // Seller checking if they rated the buyer
    return props.message.metadata?.seller_rating !== undefined && props.message.metadata?.seller_rating !== null;
  }
});

// Get the rating to display
const displayedRating = computed(() => {
  if (props.isOwnMessage) {
    return props.message.metadata?.buyer_rating || 0;
  } else {
    return props.message.metadata?.seller_rating || 0;
  }
});

// Get the review to display
const displayedReview = computed(() => {
  if (props.isOwnMessage) {
    return props.message.metadata?.buyer_review || '';
  } else {
    return props.message.metadata?.seller_review || '';
  }
});

// Processing state management
function resetProcessing() {
  isProcessing.value = false;
  if (processingTimeout.value) {
    clearTimeout(processingTimeout.value);
    processingTimeout.value = null;
  }
}

function startProcessing() {
  isProcessing.value = true;

  // Fallback: reset after 10 seconds if no response
  processingTimeout.value = window.setTimeout(() => {
    console.warn('Booking action timeout - resetting processing state');
    resetProcessing();
  }, 10000);
}

function getImageUrl(imagePath: string): string {
  if (imagePath.startsWith('http')) {
    return imagePath;
  }
  // Use STORE_API_BASE_URL (without /api/v1) since uploads are served at root level
  return `${config.STORE_API_BASE_URL}${imagePath}`;
}

async function handleApprove() {
  if (!props.message.metadata?.booking_id) return;
  startProcessing();
  emit('bookingAction', props.message.metadata.booking_id, 'approve');
}

async function handleDecline() {
  if (!props.message.metadata?.booking_id) return;
  startProcessing();
  emit('bookingAction', props.message.metadata.booking_id, 'decline');
}

async function handleConfirmReceived() {
  if (!props.message.metadata?.booking_id) return;
  startProcessing();
  emit('bookingAction', props.message.metadata.booking_id, 'confirm-received');
}

async function handleConfirmDelivery() {
  if (!props.message.metadata?.booking_id) return;
  startProcessing();
  emit('bookingAction', props.message.metadata.booking_id, 'confirm-delivery');
}

function selectRating(rating: number) {
  selectedRating.value = rating;
}

async function submitRating() {
  if (!props.message.metadata?.booking_id || !selectedRating.value) return;

  startProcessing();
  const action = props.isOwnMessage ? 'rate-seller' : 'rate-buyer';

  emit(
    'bookingAction',
    props.message.metadata.booking_id,
    action,
    selectedRating.value,
    reviewText.value
  );
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();

  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return `Yesterday ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  }

  return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

// Watch for message metadata changes to detect completion
watch(
  () => props.message.metadata,
  (newMetadata, oldMetadata) => {
    // If action completed (status changed or rating added), reset processing state
    if (isProcessing.value) {
      const statusChanged = newMetadata?.status !== oldMetadata?.status;
      const buyerRatingAdded = newMetadata?.buyer_rating && !oldMetadata?.buyer_rating;
      const sellerRatingAdded = newMetadata?.seller_rating && !oldMetadata?.seller_rating;

      if (statusChanged || buyerRatingAdded || sellerRatingAdded) {
        console.log('Booking action completed - resetting processing state');
        resetProcessing();
      }
    }
  },
  { deep: true }
);

// Cleanup on unmount
onUnmounted(() => {
  if (processingTimeout.value) {
    clearTimeout(processingTimeout.value);
  }
});
</script>

<style scoped>
.booking-message {
  margin: 0.5rem 0;
  padding: 0.75rem;
  background: #f0f9ff;
  border-left: 3px solid #0284c7;
  border-radius: 0.375rem;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.message-type-booking_approved {
  background: #f0fdf4;
  border-left-color: #10b981;
}

.message-type-booking_declined {
  background: #fef2f2;
  border-left-color: #ef4444;
}

.booking-icon {
  font-size: 1.125rem;
  color: #0284c7;
  margin-bottom: 0.5rem;
}

.message-type-booking_approved .booking-icon {
  color: #10b981;
}

.message-type-booking_declined .booking-icon {
  color: #ef4444;
}

.booking-content {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.booking-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.25rem;
}

.booking-header h4 {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
}

.status-badge {
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.6875rem;
  font-weight: 600;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-approved {
  background: #d1fae5;
  color: #065f46;
}

.status-declined {
  background: #fee2e2;
  color: #991b1b;
}

.status-rejected {
  background: #fee2e2;
  color: #991b1b;
}

.status-item_received {
  background: #dbeafe;
  color: #1e40af;
}

.status-completed {
  background: #d1fae5;
  color: #065f46;
}

.item-details {
  display: flex;
  gap: 0.625rem;
  padding: 0.5rem;
  background: white;
  border-radius: 0.25rem;
}

.item-thumbnail {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 0.25rem;
  flex-shrink: 0;
}

.item-info {
  flex: 1;
  min-width: 0;
}

.item-title {
  margin: 0 0 0.125rem 0;
  font-weight: 600;
  color: #111827;
  font-size: 0.8125rem;
}

.requester {
  margin: 0;
  font-size: 0.75rem;
  color: #6b7280;
}

.booking-message-text {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: white;
  border-radius: 0.25rem;
  border-left: 2px solid #0284c7;
}

.booking-message-text p {
  margin: 0;
  font-size: 0.875rem;
  color: #374151;
  line-height: 1.5;
}

.booking-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.25rem;
}

.booking-actions button {
  padding: 0.375rem 0.625rem;
  border: none;
  border-radius: 0.25rem;
  font-weight: 600;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  min-width: 90px;
}

.booking-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-approve {
  background: #10b981;
  color: white;
}

.btn-approve:hover:not(:disabled) {
  background: #059669;
}

.btn-decline {
  background: #ef4444;
  color: white;
}

.btn-decline:hover:not(:disabled) {
  background: #dc2626;
}

.btn-confirm-received {
  background: #3b82f6;
  color: white;
}

.btn-confirm-received:hover:not(:disabled) {
  background: #2563eb;
}

.btn-confirm-delivery {
  background: #10b981;
  color: white;
}

.btn-confirm-delivery:hover:not(:disabled) {
  background: #059669;
}

/* Rating Section */
.rating-section {
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: #f8fafc;
  border-radius: 0.375rem;
  border: 1px solid #e2e8f0;
}

.rating-input h5 {
  margin: 0 0 0.5rem 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.star-rating {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
}

.star {
  font-size: 2rem;
  color: #d1d5db;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.star.filled {
  color: #fbbf24;
}

.star:hover {
  transform: scale(1.1);
}

.review-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-family: inherit;
  resize: vertical;
  margin-bottom: 0.5rem;
}

.review-input:focus {
  outline: none;
  border-color: #3b82f6;
  ring: 2px;
  ring-color: #3b82f6;
}

.btn-submit-rating {
  padding: 0.5rem 1rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.15s;
  width: 100%;
}

.btn-submit-rating:hover:not(:disabled) {
  background: #2563eb;
}

.btn-submit-rating:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.rating-display {
  padding: 0.5rem;
}

.rating-submitted {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.rating-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #6b7280;
}

.star-display {
  display: flex;
  gap: 0.125rem;
}

.star-display .star {
  font-size: 1.25rem;
  color: #d1d5db;
  cursor: default;
}

.star-display .star.filled {
  color: #fbbf24;
}

.review-text {
  margin: 0;
  padding: 0.5rem;
  background: white;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  color: #374151;
  border: 1px solid #e5e7eb;
  font-style: italic;
}

.booking-status {
  margin-top: 0.25rem;
}

.booking-status p {
  margin: 0;
  padding: 0.375rem 0.5rem;
  border-radius: 0.25rem;
  text-align: center;
  font-weight: 600;
  font-size: 0.75rem;
}

.booking-status .approved {
  background: #d1fae5;
  color: #065f46;
}

.booking-status .declined {
  background: #fee2e2;
  color: #991b1b;
}

.booking-status .pending {
  background: #fef3c7;
  color: #92400e;
}

.booking-status .item-received {
  background: #dbeafe;
  color: #1e40af;
}

.booking-status .completed {
  background: #d1fae5;
  color: #065f46;
}

.booking-status-update {
  padding: 0.25rem;
}

.booking-status-update p {
  margin: 0;
  font-size: 0.8125rem;
  color: #111827;
  font-weight: 500;
}

.timestamp {
  font-size: 0.6875rem;
  color: #9ca3af;
  text-align: right;
  display: block;
  margin-top: 0.25rem;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .booking-message {
    margin: 0.5rem 0;
    padding: 0.625rem;
  }

  .item-thumbnail {
    width: 40px;
    height: 40px;
  }

  .booking-actions {
    flex-direction: column;
  }

  .booking-actions button {
    width: 100%;
  }
}
</style>
