<template>
  <div class="message-thread">
    <!-- Thread Header -->
    <div class="thread-header">
      <div class="header-info">
        <h2>{{ conversationTitle }}</h2>
        <p v-if="conversationDescription">{{ conversationDescription }}</p>
      </div>
      <router-link v-if="contextLink" :to="contextLink.to" class="context-link">
        {{ contextLink.label }}
      </router-link>
    </div>
    
    <!-- Messages Container -->
    <div class="messages-container" ref="messagesContainer" @scroll="handleScroll">
      <!-- Load More Button -->
      <div v-if="props.hasMore" class="load-more">
        <button @click="$emit('load-more')" :disabled="loading">
          {{ loading ? 'Loading...' : 'Load earlier messages' }}
        </button>
      </div>
      
      <!-- Messages -->
      <template v-for="message in messages" :key="message.id">
        <RequestOfferMessage
          v-if="message.message_type === 'system_alert' && message.metadata?.event === 'request_answered'"
          :message="message"
          :current-user-id="authStore.user?.id"
        />
        <TaskEventMessage
          v-else-if="message.message_type === 'system_alert' && message.task_id && message.metadata?.event"
          :message="message"
          :current-user-id="authStore.user?.id"
          :latest="message.id === latestEventId"
        />
        <BookingStepMessage
          v-else-if="isBookingStep(message)"
          :message="message"
          :request="bookingRequests.get(Number(message.metadata?.booking_id))"
          :current-user-id="authStore.user?.id"
          :latest="message.id === latestStepIds.get(Number(message.metadata?.booking_id))"
          @booking-action="handleBookingAction"
        />
        <div v-else-if="isNote(message)" class="thread-note">
          {{ message.content }}
          <span class="thread-note-time">{{ formatTime(message.created_at) }}</span>
        </div>
        <BookingMessageBubble
          v-else-if="isBookingMessage(message)"
          :message="message"
          :is-own-message="isOwnMessage(message)"
          @booking-action="handleBookingAction"
        />
        <MessageBubble
          v-else
          :message="message"
          :is-own-message="isOwnMessage(message)"
          @edit="$emit('edit-message', message.id, $event)"
          @delete="$emit('delete-message', message.id)"
        />
      </template>
      
      <!-- Typing Indicators -->
      <div v-if="typingUsers.length > 0" class="typing-indicator">
        <span class="typing-dots">
          <span></span>
          <span></span>
          <span></span>
        </span>
        <span class="typing-text">
          {{ typingText }}
        </span>
      </div>
    </div>
    
    <!-- Message Input: a gig chat opens once the poster accepts. Until then,
         and once the deal has ended, a strip says why instead -->
    <p v-if="ended || locked" class="message-closed" role="status">
      <template v-if="ended">This conversation has ended. You can still read it{{ hasReviewStep ? ' and leave your review above' : '' }}.</template>
      <template v-else>{{ lockedNote }}</template>
    </p>
    <div v-else class="message-input-container">
      <form @submit.prevent="sendMessage" class="message-form">
        <input
          v-model="messageText"
          type="text"
          placeholder="Type a message..."
          aria-label="Message"
          class="message-input"
          @input="handleTyping"
          maxlength="1000"
        />
        <button type="submit" class="send-button" :disabled="!messageText.trim()" aria-label="Send">
          <i class="fas fa-paper-plane" aria-hidden="true"></i>
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue';
import { useAuthStore } from '@/stores/auth';
import MessageBubble from './MessageBubble.vue';
import BookingMessageBubble from './BookingMessageBubble.vue';
import TaskEventMessage from './TaskEventMessage.vue';
import RequestOfferMessage from './RequestOfferMessage.vue';
import BookingStepMessage from './BookingStepMessage.vue';
import type { Message, ConversationSummary, BookingAction } from '@/stores/messages';

const props = defineProps<{
  conversation: ConversationSummary;
  messages: Message[];
  typingUsers: Array<{ userId: number; userName: string }>;
  loading: boolean;
  hasMore: boolean;
  // Nobody can write here yet (a gig chat before the poster accepts)
  locked?: boolean;
  // The deal is done or closed: readable, not writable
  ended?: boolean;
}>();

const emit = defineEmits<{
  'send-message': [content: string];
  'edit-message': [messageId: number, content: string];
  'delete-message': [messageId: number];
  'load-more': [];
  'typing-start': [];
  'typing-stop': [];
  'booking-action': [bookingId: number, action: BookingAction, rating?: number, review?: string];
}>();

const authStore = useAuthStore();
const messagesContainer = ref<HTMLElement>();
const messageText = ref('');
const isTyping = ref(false);
const typingTimeout = ref<ReturnType<typeof setTimeout>>();

// The newest gig event: the only one that offers the next step
const latestEventId = computed(() => {
  for (let i = props.messages.length - 1; i >= 0; i--) {
    if (props.messages[i].task_id && props.messages[i].metadata?.event) return props.messages[i].id;
  }
  return null;
});

// A completed deal still offers its review in the conversation
const hasReviewStep = computed(() => props.messages.some(m =>
  m.metadata?.event === 'completed' || m.message_type === 'booking_completed'));

// Before a match, the poster or seller is the one who can open the chat
const lockedNote = computed(() => {
  const me = authStore.user?.id;
  if (props.conversation.item_id || props.conversation.conversation_type === 'store') {
    const booking = [...props.messages].reverse().find(m => m.message_type === 'booking_request');
    if (booking && booking.recipient_id === me) return 'Approve the booking to start chatting.';
    if (booking) return 'You can chat once the seller approves your booking.';
    return 'Book the item to ask the seller. You can chat once they approve.';
  }
  const latest = props.messages.find(m => m.id === latestEventId.value);
  const isPoster = latest?.metadata?.event === 'application' && latest.recipient_id === me;
  return isPoster
    ? 'Accept the application to start chatting.'
    : 'You can chat once the poster accepts the application.';
});

// Computed properties for conversation display
const conversationTitle = computed(() => {
  // Priority order: task > application > item
  if (props.conversation.task_title) {
    return props.conversation.task_title;
  }
  if (props.conversation.application_title) {
    return props.conversation.application_title;
  }
  if (props.conversation.item_title) {
    return props.conversation.item_title;
  }
  return 'Conversation';
});

// Where the conversation's gig or item lives, to act on it (book, apply)
const contextLink = computed(() => {
  const c = props.conversation;
  if (c.conversation_type === 'store' && c.item_id) {
    return { to: { name: 'store-item', params: { id: c.item_id } }, label: 'View item' };
  }
  if (c.conversation_type === 'task' && c.task_id) {
    return { to: { name: 'task-detail', params: { id: c.task_id } }, label: 'View gig' };
  }
  return null;
});

const conversationDescription = computed(() => {
  // Who this conversation is with matters most: a gig or an item can have
  // several conversations, one per person
  if (props.conversation.other_user_name) {
    return `with ${props.conversation.other_user_name}`;
  }
  if (props.conversation.task_description) {
    return props.conversation.task_description;
  }
  return '';
});

const typingText = computed(() => {
  if (props.typingUsers.length === 0) return '';
  if (props.typingUsers.length === 1) {
    return `${props.typingUsers[0].userName} is typing...`;
  }
  if (props.typingUsers.length === 2) {
    return `${props.typingUsers[0].userName} and ${props.typingUsers[1].userName} are typing...`;
  }
  return `${props.typingUsers[0].userName} and ${props.typingUsers.length - 1} others are typing...`;
});

function isOwnMessage(message: Message): boolean {
  return message.sender_id === authStore.user?.id;
}

function formatTime(date: string): string {
  const d = new Date(date);
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' });
}

// Notes recording what happened: shown centred, never editable
const NOTE_TYPES = ['system_alert', 'booking_declined', 'booking_status_update'];

// Booking steps after the request: each offers the next step, like gig events
const STEP_TYPES = ['booking_approved', 'booking_picked_up', 'booking_item_received', 'booking_completed'];
function isBookingStep(message: Message): boolean {
  return STEP_TYPES.includes(message.message_type) && !!message.metadata?.booking_id;
}

// Each booking's request (buyer, seller, current status, ratings) and its
// newest step: only that one offers a step
const bookingRequests = computed(() => {
  const requests = new Map<number, Message>();
  for (const m of props.messages) {
    if (m.message_type === 'booking_request' && m.metadata?.booking_id) requests.set(Number(m.metadata.booking_id), m);
  }
  return requests;
});
const latestStepIds = computed(() => {
  const latest = new Map<number, number>();
  for (const m of props.messages) {
    if (isBookingStep(m)) latest.set(Number(m.metadata!.booking_id), m.id);
  }
  return latest;
});
function isNote(message: Message): boolean {
  return NOTE_TYPES.includes(message.message_type);
}

function isBookingMessage(message: Message): boolean {
  return message.message_type === 'booking_request';
}

function handleBookingAction(
  bookingId: number,
  action: BookingAction,
  rating?: number,
  review?: string
) {
  emit('booking-action', bookingId, action, rating, review);
}

function sendMessage() {
  if (!messageText.value.trim()) return;
  
  emit('send-message', messageText.value);
  messageText.value = '';
  
  // Stop typing indicator
  if (isTyping.value) {
    isTyping.value = false;
    emit('typing-stop');
  }
}

function handleTyping() {
  if (!isTyping.value && messageText.value.trim()) {
    isTyping.value = true;
    emit('typing-start');
  }
  
  // Clear existing timeout
  if (typingTimeout.value) {
    clearTimeout(typingTimeout.value);
  }
  
  // Set new timeout
  typingTimeout.value = setTimeout(() => {
    if (isTyping.value) {
      isTyping.value = false;
      emit('typing-stop');
    }
  }, 1000);
}

function handleScroll() {
  if (!messagesContainer.value) return;
  
  // Check if scrolled to top
  if (messagesContainer.value.scrollTop === 0 && props.hasMore && !props.loading) {
    emit('load-more');
  }
}

// Auto-scroll to bottom on new messages
watch(() => props.messages.length, () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
    }
  });
});
</script>

<style scoped>
.message-thread {
  display: flex;
  flex-direction: column;
  height: 100%;
  flex: 1;
  min-height: 0;
  background: var(--bg);
}

/* Thread Header: back, what it's about and with whom, a link to it */
.thread-header {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px 14px;
  border-bottom: 1px solid #EEF0F3;
  background: var(--surface);
}

.header-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.header-info h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-info p {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.context-link {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 14px;
  border-radius: 16px;
  background: var(--accent-tint);
  color: var(--accent-text);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.context-link:hover,
.context-link:focus-visible {
  color: var(--color-primary-dark);
}

/* Messages Container */
.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 20px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.load-more {
  text-align: center;
  margin-bottom: 1rem;
}

.load-more button {
  min-height: 36px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--surface);
  color: var(--text-muted);
  font-size: 13px;
  cursor: pointer;
}

.load-more button:hover:not(:disabled) {
  color: var(--text);
}

.load-more button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Typing Indicator */
.typing-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 16px 16px 16px 4px;
  background: var(--surface);
  width: fit-content;
}

.typing-dots {
  display: flex;
  gap: 0.25rem;
}

.typing-dots span {
  width: 0.5rem;
  height: 0.5rem;
  background: var(--text-muted);
  border-radius: 50%;
  animation: typing 1.4s infinite;
}

.typing-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes typing {
  0%, 60%, 100% {
    opacity: 0.3;
  }
  30% {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .typing-dots span {
    animation: none;
  }
}

.typing-text {
  font-size: 13px;
  color: var(--text-muted);
}

/* Composer: a rounded box and a round send button */
.message-input-container {
  flex: none;
  padding: 12px 16px 16px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.message-form {
  display: flex;
  align-items: center;
  gap: 10px;
}

.message-input {
  flex: 1;
  min-width: 0;
  height: 44px;
  padding: 0 16px;
  border: 1px solid transparent;
  border-radius: 22px;
  background: #F3F4F6;
  color: var(--text);
  font-size: 15px;
}

.message-input:focus {
  outline: none;
  border-color: var(--accent);
  background: var(--surface);
}

.send-button {
  flex: none;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 22px;
  background: var(--accent);
  color: var(--on-accent);
  cursor: pointer;
}

.send-button:disabled {
  background: #E5E7EB;
  color: #fff;
  cursor: not-allowed;
}

/* Closed to typing: the reason, in a strip where you'd type */
.message-closed {
  flex: none;
  margin: 0;
  padding: 14px 20px;
  border-top: 1px solid #EEF0F3;
  background: #F8F9FA;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.4;
  text-align: center;
}

/* Notes in the conversation (not events): small and centred. Not named
   .system-message: scoped styles reach child roots, and the event cards use it. */
.thread-note {
  align-self: center;
  max-width: 90%;
  margin: 6px auto;
  color: var(--text-muted);
  font-size: 12px;
  text-align: center;
  white-space: pre-line;
}

.thread-note-time {
  display: block;
  margin-top: 2px;
  font-size: 11px;
}
</style>
