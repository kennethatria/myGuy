<template>
  <!-- A booking request at the top of the conversation: who asked, and where
       it stands. The seller answers it here; later steps arrive as messages
       below (BookingStepMessage), like gig events. -->
  <div class="system-message booking-event">
    <p class="event-text">{{ headline }}</p>
    <p v-if="note" class="event-quote">“{{ note }}”</p>
    <span v-if="statusText" :class="['event-status', `status-${status}`]">{{ statusText }}</span>
    <span class="system-message-time">{{ formatTime(message.created_at) }}</span>

    <!-- Seller: answer the request -->
    <div v-if="isSeller && status === 'pending'" class="event-actions">
      <button type="button" class="btn btn-primary btn-sm btn-approve" :disabled="isProcessing" @click="act('approve')">Approve</button>
      <button type="button" class="btn btn-outline btn-sm btn-decline-request" :disabled="isProcessing" @click="act('decline')">Decline</button>
    </div>

    <!-- Waiting on the other person, or closed -->
    <p v-else-if="waitingNote" class="event-note">{{ waitingNote }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue';
import { useUserStore } from '@/stores/user';
import type { Message, BookingAction } from '@/stores/messages';

const props = defineProps<{
  message: Message;
  // The booking request is the buyer's own message
  isOwnMessage: boolean;
}>();

const emit = defineEmits<{
  bookingAction: [bookingId: number, action: BookingAction, rating?: number, review?: string];
}>();

const userStore = useUserStore();
const isProcessing = ref(false);
const processingTimeout = ref<number | null>(null);

// The buyer sent the request; the seller received it
const isSeller = computed(() => !props.isOwnMessage);
const status = computed(() => props.message.metadata?.status);

const senderName = computed(() => {
  if (props.message.sender?.username) return props.message.sender.username;
  const user = props.message.sender_id ? userStore.getUserById(props.message.sender_id) : undefined;
  return user?.username || 'Someone';
});

// What the buyer wrote, if anything. Older bookings kept it as the text.
const note = computed(() => {
  const metadata = props.message.metadata as { note?: string } | undefined;
  return metadata && 'note' in metadata ? metadata.note : props.message.content;
});

const headline = computed(() => {
  const title = props.message.metadata?.item_title || 'this item';
  return props.isOwnMessage
    ? `📩 You asked to book "${title}".`
    : `📩 ${senderName.value} asked to book "${title}".`;
});

const statusText = computed(() => {
  switch (status.value) {
    case 'pending': return 'Pending';
    case 'approved': return 'Approved';
    case 'rejected': return 'Declined';
    case 'picked_up': return 'Picked up';
    case 'item_received': return 'Collected';
    case 'completed': return 'Completed';
    case 'released': return 'Released';
    default: return '';
  }
});

const waitingNote = computed(() => {
  switch (status.value) {
    case 'pending': return 'Waiting for the seller to answer.';
    case 'rejected': return '❌ Booking declined';
    case 'released': return '↩️ Reservation released';
    default: return '';
  }
});

function resetProcessing() {
  isProcessing.value = false;
  if (processingTimeout.value) {
    clearTimeout(processingTimeout.value);
    processingTimeout.value = null;
  }
}

function startProcessing() {
  isProcessing.value = true;
  // Fallback: reset after 10 seconds if no update arrives
  processingTimeout.value = window.setTimeout(resetProcessing, 10000);
}

// Runs a step; its result arrives as an update to this message
function act(action: BookingAction) {
  const bookingId = props.message.metadata?.booking_id;
  if (!bookingId) return;
  startProcessing();
  emit('bookingAction', bookingId, action);
}

function formatTime(timestamp: string): string {
  const d = new Date(timestamp);
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' });
}

// The answer is in once the request's status changes
watch(() => props.message.metadata?.status, resetProcessing);

onUnmounted(resetProcessing);
</script>

<style scoped src="./eventCard.css"></style>

<style scoped>
/* The buyer's own note on the booking */
.event-quote {
  margin: 0.25rem 0 0;
  color: var(--color-primary-dark);
  font-style: italic;
  overflow-wrap: anywhere;
}

.event-status {
  display: inline-block;
  margin-top: 0.375rem;
  padding: 0 0.5rem;
  border-radius: 999px;
  background: var(--accent-tint);
  font-size: 0.75rem;
  font-weight: 600;
}

.status-approved,
.status-picked_up,
.status-item_received,
.status-completed {
  background: #dcfce7;
  color: #166534;
}

.status-rejected,
.status-released {
  background: #fee2e2;
  color: #991b1b;
}
</style>
