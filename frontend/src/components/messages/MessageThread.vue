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
        <BookingMessageBubble
          v-if="isBookingMessage(message)"
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
    
    <!-- Message Input -->
    <div class="message-input-container">
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
import type { Message, ConversationSummary } from '@/stores/messages';

const props = defineProps<{
  conversation: ConversationSummary;
  messages: Message[];
  typingUsers: Array<{ userId: number; userName: string }>;
  loading: boolean;
  hasMore: boolean;
}>();

const emit = defineEmits<{
  'send-message': [content: string];
  'edit-message': [messageId: number, content: string];
  'delete-message': [messageId: number];
  'load-more': [];
  'typing-start': [];
  'typing-stop': [];
  'booking-action': [bookingId: number, action: 'approve' | 'decline'];
}>();

const authStore = useAuthStore();
const messagesContainer = ref<HTMLElement>();
const messageText = ref('');
const isTyping = ref(false);
const typingTimeout = ref<NodeJS.Timeout>();

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

const conversationDescription = computed(() => {
  // Only show description for tasks
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

function isBookingMessage(message: Message): boolean {
  return ['booking_request', 'booking_approved', 'booking_declined'].includes(message.message_type);
}

function handleBookingAction(
  bookingId: number,
  action: 'approve' | 'decline' | 'confirm-received' | 'confirm-delivery' | 'rate-seller' | 'rate-buyer',
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
</style>