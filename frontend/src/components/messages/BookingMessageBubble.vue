<template>
  <!-- A booking in the conversation, shown like a gig event: what happened,
       then the step it asks of whoever acts next. -->
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

    <!-- Seller, approved: waiting for the buyer, or release it if the sale
         went nowhere (asks once more first) -->
    <template v-else-if="isSeller && status === 'approved'">
      <p class="event-note">Reserved. Waiting for the buyer to collect it.</p>
      <p v-if="confirmingRelease" class="event-note">Put it back on the board for others?</p>
      <div class="event-actions">
        <button v-if="!confirmingRelease" type="button" class="btn btn-outline btn-sm btn-release" :disabled="isProcessing" @click="confirmingRelease = true">
          Release reservation
        </button>
        <template v-else>
          <button type="button" class="btn btn-danger btn-sm btn-decline" :disabled="isProcessing" @click="release">Yes, release</button>
          <button type="button" class="btn btn-outline btn-sm btn-keep" :disabled="isProcessing" @click="confirmingRelease = false">Keep it</button>
        </template>
      </div>
    </template>

    <!-- Buyer, approved: say when it's collected -->
    <div v-else-if="!isSeller && status === 'approved'" class="event-actions">
      <button type="button" class="btn btn-primary btn-sm btn-confirm-received" :disabled="isProcessing" @click="act('confirm-received')">
        I've collected it
      </button>
    </div>

    <!-- Seller, collected: confirm the handover to finish -->
    <div v-else-if="isSeller && status === 'item_received'" class="event-actions">
      <button type="button" class="btn btn-primary btn-sm btn-confirm-delivery" :disabled="isProcessing" @click="act('confirm-delivery')">
        Confirm handover
      </button>
    </div>

    <!-- Completed: both review each other, comment optional -->
    <form v-else-if="status === 'completed' && !hasRated" class="event-review" @submit.prevent="submitRating">
      <div class="stars" role="radiogroup" :aria-label="isSeller ? 'Rate the buyer' : 'Rate the seller'">
        <button
          v-for="star in 5"
          :key="star"
          type="button"
          role="radio"
          :aria-checked="selectedRating === star"
          :aria-label="`${star} star${star === 1 ? '' : 's'}`"
          :class="['star', { on: star <= selectedRating }]"
          @click="selectedRating = star"
        >★</button>
      </div>
      <textarea v-model="reviewText" rows="2" maxlength="500" placeholder="Add a comment (optional)" class="event-comment"></textarea>
      <button type="submit" class="btn btn-primary btn-sm" :disabled="isProcessing || !selectedRating">Leave review</button>
    </form>

    <p v-else-if="status === 'completed'" class="event-note done">
      You gave ★ {{ displayedRating }}<template v-if="displayedReview">: “{{ displayedReview }}”</template>
    </p>

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
const confirmingRelease = ref(false);
const selectedRating = ref(0);
const reviewText = ref('');

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
    case 'item_received': return 'Collected';
    case 'completed': return 'Completed';
    case 'released': return 'Released';
    default: return '';
  }
});

const waitingNote = computed(() => {
  switch (status.value) {
    case 'pending': return 'Waiting for the seller to answer.';
    case 'item_received': return 'Waiting for the seller to confirm the handover.';
    case 'rejected': return '❌ Booking declined';
    case 'released': return '↩️ Reservation released';
    default: return '';
  }
});

// Whether the viewer already reviewed the other person
const hasRated = computed(() => {
  const rating = props.isOwnMessage ? props.message.metadata?.buyer_rating : props.message.metadata?.seller_rating;
  return rating !== undefined && rating !== null;
});
const displayedRating = computed(() =>
  (props.isOwnMessage ? props.message.metadata?.buyer_rating : props.message.metadata?.seller_rating) || 0);
const displayedReview = computed(() =>
  (props.isOwnMessage ? props.message.metadata?.buyer_review : props.message.metadata?.seller_review) || '');

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

function release() {
  confirmingRelease.value = false;
  act('release');
}

function submitRating() {
  const bookingId = props.message.metadata?.booking_id;
  if (!bookingId || !selectedRating.value) return;
  startProcessing();
  emit('bookingAction', bookingId, props.isOwnMessage ? 'rate-seller' : 'rate-buyer', selectedRating.value, reviewText.value.trim());
}

function formatTime(timestamp: string): string {
  const d = new Date(timestamp);
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' });
}

// The step is done once the message's status or ratings change
watch(
  () => props.message.metadata,
  (next, prev) => {
    if (next?.status !== prev?.status || next?.buyer_rating !== prev?.buyer_rating || next?.seller_rating !== prev?.seller_rating) {
      resetProcessing();
    }
  },
  { deep: true }
);

onUnmounted(resetProcessing);
</script>

<style scoped>
/* Matches the gig event cards (TaskEventMessage) */
.system-message {
  align-self: center;
  max-width: min(90%, 28rem);
  margin: 0.5rem auto;
  padding: 0.5rem 0.75rem;
  border: 1px solid #c7d2fe;
  border-radius: 0.5rem;
  background: #eef2ff;
  color: #3730a3;
  font-size: 0.875rem;
  text-align: center;
}

.event-text,
.event-quote,
.event-note {
  margin: 0;
  overflow-wrap: anywhere;
}

.event-quote {
  margin-top: 0.25rem;
  color: #4338ca;
  font-style: italic;
}

.event-note {
  margin-top: 0.5rem;
}

.event-note.done {
  color: #166534;
}

.event-status {
  display: inline-block;
  margin-top: 0.375rem;
  padding: 0 0.5rem;
  border-radius: 999px;
  background: #e0e7ff;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-approved,
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

.system-message-time {
  display: block;
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: #6366f1;
}

.event-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 0.625rem;
}

/* Thumb-sized on phones */
.event-actions .btn,
.event-review .btn {
  min-height: 44px;
  min-width: 6.5rem;
}

.event-review {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.625rem;
}

.stars {
  display: flex;
  gap: 0.25rem;
}

.star {
  min-width: 44px;
  min-height: 44px;
  border: none;
  background: none;
  font-size: 1.6rem;
  line-height: 1;
  color: #c7d2fe;
  cursor: pointer;
}

.star.on {
  color: #f59e0b;
}

.event-comment {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #c7d2fe;
  border-radius: 0.375rem;
  font: inherit;
  resize: vertical;
}
</style>
