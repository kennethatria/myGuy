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

        <!-- Action Buttons (only show for seller if pending) -->
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

        <!-- Status Message (if already decided) -->
        <div v-else-if="message.metadata?.status !== 'pending'" class="booking-status">
          <p v-if="message.metadata?.status === 'approved'" class="approved">
            ✅ Booking approved
          </p>
          <p v-else class="declined">
            ❌ Booking declined
          </p>
        </div>

        <!-- Pending status for sender -->
        <div v-else-if="isOwnMessage" class="booking-status">
          <p class="pending">⏳ Waiting for seller response...</p>
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
import { ref, computed } from 'vue';
import { useUserStore } from '@/stores/user';
import config from '@/config';
import type { Message } from '@/stores/messages';

const props = defineProps<{
  message: Message;
  isOwnMessage: boolean;
}>();

const emit = defineEmits<{
  bookingAction: [bookingId: number, action: 'approve' | 'decline'];
}>();

const userStore = useUserStore();
const isProcessing = ref(false);

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
    default:
      return 'fas fa-info-circle';
  }
});

const statusText = computed(() => {
  const status = props.message.metadata?.status;
  if (status === 'pending') return 'Pending';
  if (status === 'approved') return 'Approved';
  if (status === 'declined') return 'Declined';
  return '';
});

const statusBadgeClass = computed(() => {
  const status = props.message.metadata?.status;
  return `status-badge status-${status}`;
});

function getImageUrl(imagePath: string): string {
  if (imagePath.startsWith('http')) {
    return imagePath;
  }
  return `${config.STORE_API_URL}${imagePath}`;
}

async function handleApprove() {
  if (!props.message.metadata?.booking_id) return;
  isProcessing.value = true;
  emit('bookingAction', props.message.metadata.booking_id, 'approve');
  // Note: isProcessing will be reset when the response comes back via WebSocket
}

async function handleDecline() {
  if (!props.message.metadata?.booking_id) return;
  isProcessing.value = true;
  emit('bookingAction', props.message.metadata.booking_id, 'decline');
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
</script>

<style scoped>
.booking-message {
  margin: 1rem 0;
  padding: 1rem;
  background: #f0f9ff;
  border-left: 4px solid #0284c7;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
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
  font-size: 1.5rem;
  color: #0284c7;
  margin-bottom: 0.75rem;
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
  gap: 0.75rem;
}

.booking-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.booking-header h4 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #111827;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
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

.item-details {
  display: flex;
  gap: 1rem;
  padding: 0.75rem;
  background: white;
  border-radius: 0.375rem;
}

.item-thumbnail {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border-radius: 0.375rem;
  flex-shrink: 0;
}

.item-info {
  flex: 1;
  min-width: 0;
}

.item-title {
  margin: 0 0 0.25rem 0;
  font-weight: 600;
  color: #111827;
  font-size: 0.9375rem;
}

.requester {
  margin: 0;
  font-size: 0.875rem;
  color: #6b7280;
}

.booking-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.booking-actions button {
  flex: 1;
  padding: 0.625rem 1rem;
  border: none;
  border-radius: 0.375rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
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

.booking-status {
  margin-top: 0.5rem;
}

.booking-status p {
  margin: 0;
  padding: 0.625rem;
  border-radius: 0.375rem;
  text-align: center;
  font-weight: 600;
  font-size: 0.875rem;
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

.booking-status-update {
  padding: 0.5rem;
}

.booking-status-update p {
  margin: 0;
  font-size: 0.9375rem;
  color: #111827;
  font-weight: 500;
}

.timestamp {
  font-size: 0.75rem;
  color: #9ca3af;
  text-align: right;
  display: block;
  margin-top: 0.5rem;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .booking-message {
    margin: 0.75rem 0;
    padding: 0.875rem;
  }

  .item-thumbnail {
    width: 56px;
    height: 56px;
  }

  .booking-actions {
    flex-direction: column;
  }

  .booking-actions button {
    width: 100%;
  }
}
</style>
