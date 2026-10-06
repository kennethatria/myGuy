<template>
  <div class="message-thread">
    <!-- Thread Header -->
    <div class="thread-header">
      <div class="header-info">
        <h2>{{ conversationTitle }}</h2>
        <p v-if="conversationDescription">{{ conversationDescription }}</p>
      </div>
      <div class="header-meta">
        <span v-if="conversation.task_status" class="task-status" :class="`status-${conversation.task_status}`">
          {{ conversation.task_status }}
        </span>
        <router-link v-if="contextLink" :to="contextLink.to" class="context-link">
          {{ contextLink.label }}
        </router-link>
      </div>
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
        <TaskEventMessage
          v-if="message.message_type === 'system_alert' && message.metadata?.event"
          :message="message"
          :current-user-id="authStore.user?.id"
          :latest="message.id === latestEventId"
        />
        <div v-else-if="isNote(message)" class="system-message">
          {{ message.content }}
          <span class="system-message-time">{{ formatTime(message.created_at) }}</span>
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
    
    <!-- Message Input: a gig chat opens once the poster accepts -->
    <div v-if="locked" class="message-locked" role="status">
      {{ lockedNote }}
    </div>
    <div v-else class="message-input-container">
      <form @submit.prevent="sendMessage" class="message-form">
        <input
          v-model="messageText"
          type="text"
          placeholder="Type a message..."
          class="message-input"
          @input="handleTyping"
          maxlength="1000"
        />
        <button type="submit" class="send-button" :disabled="!messageText.trim()">
          <i class="fas fa-paper-plane"></i>
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
import type { Message, ConversationSummary, BookingAction } from '@/stores/messages';

const props = defineProps<{
  conversation: ConversationSummary;
  messages: Message[];
  typingUsers: Array<{ userId: number; userName: string }>;
  loading: boolean;
  hasMore: boolean;
  // Nobody can write here yet (a gig chat before the poster accepts)
  locked?: boolean;
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
    if (props.messages[i].metadata?.event) return props.messages[i].id;
  }
  return null;
});

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
const NOTE_TYPES = ['system_alert', 'booking_approved', 'booking_declined', 'booking_item_received', 'booking_completed', 'booking_status_update'];
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
.context-link {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 0 0.25rem;
  color: var(--color-primary, #4f46e5);
  font-weight: 500;
  white-space: nowrap;
  text-decoration: none;
}

.context-link:hover,
.context-link:focus-visible {
  text-decoration: underline;
}

.message-thread {
  display: flex;
  flex-direction: column;
  height: 100%;
  /* In a flex column (Message Center), take the space left after siblings
     such as the mobile back button, so the reply box stays on screen. */
  flex: 1;
  min-height: 0;
}

/* Thread Header */
.thread-header {
  padding: 1.5rem;
  background: white;
  border-bottom: 1px solid #e5e7eb;
}

.header-info h2 {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0 0 0.25rem 0;
}

.header-info p {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-meta {
  display: flex;
  gap: 1rem;
  margin-top: 0.5rem;
  font-size: 0.875rem;
}

.task-status {
  padding: 0.125rem 0.5rem;
  border-radius: 0.25rem;
  font-weight: 500;
}

.status-open {
  background: #dbeafe;
  color: #1e40af;
}

.status-in_progress {
  background: #fef3c7;
  color: #92400e;
}

/* Marked done, waiting for the poster's approval */
.status-pending_approval {
  background: #ede9fe;
  color: #5b21b6;
}

.status-completed {
  background: #d1fae5;
  color: #065f46;
}

.other-user {
  color: #4F46E5;
  font-weight: 500;
}

/* Messages Container */
.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.load-more {
  text-align: center;
  margin-bottom: 1rem;
}

.load-more button {
  padding: 0.5rem 1rem;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  color: #6b7280;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.15s;
}

.load-more button:hover:not(:disabled) {
  background: #f9fafb;
  color: #111827;
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
  padding: 0.75rem;
  background: white;
  border-radius: 0.5rem;
  width: fit-content;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.typing-dots {
  display: flex;
  gap: 0.25rem;
}

.typing-dots span {
  width: 0.5rem;
  height: 0.5rem;
  background: #6b7280;
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

.typing-text {
  font-size: 0.875rem;
  color: #6b7280;
}

/* Message Input */
.message-input-container {
  padding: 1.5rem;
  background: white;
  border-top: 1px solid #e5e7eb;
}

.message-form {
  display: flex;
  gap: 0.75rem;
}

.message-input {
  flex: 1;
  padding: 0.75rem 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  transition: border-color 0.15s;
}

.message-input:focus {
  outline: none;
  border-color: #4F46E5;
}

.send-button {
  padding: 0.75rem 1.25rem;
  background: #4F46E5;
  color: white;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: background-color 0.15s;
}

.send-button:hover:not(:disabled) {
  background: #4338ca;
}

.send-button:disabled {
  background: #e5e7eb;
  cursor: not-allowed;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .thread-header {
    padding: 1rem;
  }
  
  .messages-container {
    padding: 1rem;
  }
  
  .message-input-container {
    padding: 1rem;
  }
}

/* Task events (new application, accepted, declined): shown as a notice */
.system-message {
  align-self: center;
  max-width: 90%;
  margin: 0.5rem auto;
  padding: 0.625rem 0.875rem;
  background: #eef2ff;
  border: 1px solid #c7d2fe;
  border-radius: 0.5rem;
  color: #3730a3;
  font-size: 0.875rem;
  text-align: center;
  white-space: pre-line;
}

.system-message-time {
  display: block;
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: #6366f1;
}

.message-locked {
  padding: 1rem;
  border-top: 1px solid #e5e7eb;
  background: #f9fafb;
  color: #4b5563;
  font-size: 0.875rem;
  text-align: center;
}
</style>
