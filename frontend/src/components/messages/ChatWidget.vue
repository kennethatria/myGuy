<template>
  <div class="chat-widget-container">
    <!-- Widget Button -->
    <button
      v-if="!isExpanded"
      class="chat-widget-button"
      @click="toggleWidget"
      :class="{ 'has-unread': chatStore.totalUnreadCount > 0 }"
    >
      <i class="fas fa-comments"></i>
      <span v-if="chatStore.totalUnreadCount > 0" class="unread-badge">
        {{ chatStore.totalUnreadCount }}
      </span>
    </button>
    
    <!-- Expanded Widget -->
    <div v-if="isExpanded" class="chat-widget-expanded">
      <!-- Widget Header -->
      <div class="widget-header">
        <h3>Messages</h3>
        <div class="header-actions">
          <button @click="openMessageCenter" class="expand-btn" title="Open Message Center">
            <i class="fas fa-expand"></i>
          </button>
          <button @click="toggleWidget" class="close-btn" title="Close">
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>
      
      <!-- Conversation Switcher -->
      <div v-if="!activeConversation" class="conversation-list">
        <div
          v-for="conversation in recentConversations"
          :key="conversation.task_id"
          class="conversation-item"
          @click="selectConversation(conversation)"
        >
          <div class="conversation-info">
            <h4>{{ conversation.task_title }}</h4>
            <p>{{ conversation.other_user_name }}</p>
          </div>
          <span v-if="conversation.unread_count > 0" class="unread-count">
            {{ conversation.unread_count }}
          </span>
        </div>
        
        <div v-if="chatStore.conversations.length === 0" class="no-conversations">
          <p>No conversations yet</p>
        </div>
      </div>
      
      <!-- Active Conversation -->
      <div v-else class="active-conversation">
        <!-- Conversation Header -->
        <div class="conversation-header">
          <button @click="backToList" class="back-btn">
            <i class="fas fa-arrow-left"></i>
          </button>
          <div class="conversation-title">
            <h4>{{ activeConversation.task_title }}</h4>
            <p>{{ activeConversation.other_user_name }}</p>
          </div>
        </div>
        
        <!-- Messages -->
        <div class="messages-area" ref="messagesArea">
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
              @edit="(content: string) => editMessage(message.id, content)"
              @delete="() => deleteMessage(message.id)"
            />
          </template>
          
          <!-- Typing Indicator -->
          <div v-if="typingUsers.length > 0" class="typing-indicator">
            <span class="typing-dots">
              <span></span>
              <span></span>
              <span></span>
            </span>
          </div>
        </div>
        
        <!-- Message Input -->
        <form @submit.prevent="sendMessage" class="message-input-form">
          <input
            v-model="messageText"
            type="text"
            placeholder="Type a message..."
            class="message-input"
            @input="handleTyping"
          />
          <button type="submit" :disabled="!messageText.trim()">
            <i class="fas fa-paper-plane"></i>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue';
import { useRouter } from 'vue-router';
import { useChatStore } from '@/stores/chat';
import { useAuthStore } from '@/stores/auth';
import MessageBubble from './MessageBubble.vue';
import BookingMessageBubble from './BookingMessageBubble.vue';
import type { ConversationSummary, Message } from '@/stores/messages';

const router = useRouter();
const chatStore = useChatStore();
const authStore = useAuthStore();

const isExpanded = ref(false);
const activeConversation = ref<ConversationSummary | null>(null);
const messageText = ref('');
const messagesArea = ref<HTMLElement>();
const isTyping = ref(false);
const typingTimeout = ref<ReturnType<typeof setTimeout>>();

// Get last active conversation from localStorage
const lastConversationId = localStorage.getItem('lastActiveConversation');
if (lastConversationId && chatStore.conversations.length > 0) {
  const conv = chatStore.conversations.find(c => c.task_id === parseInt(lastConversationId));
  if (conv) {
    activeConversation.value = conv;
    chatStore.joinConversation(conv.task_id!);
  }
}

const recentConversations = computed(() => {
  return chatStore.conversations.slice(0, 5);
});

const messages = computed(() => {
  if (!activeConversation.value) return [];
  const convId = activeConversation.value.task_id ?? activeConversation.value.application_id ?? activeConversation.value.item_id;
  if (convId === undefined) return [];
  return chatStore.messages.get(convId) || [];
});

const typingUsers = computed(() => {
  if (!activeConversation.value) return [];
  const convId = activeConversation.value.task_id ?? activeConversation.value.application_id ?? activeConversation.value.item_id;
  if (convId === undefined) return [];
  return chatStore.typingUsers.get(convId) || [];
});

function toggleWidget() {
  isExpanded.value = !isExpanded.value;
  
  if (isExpanded.value && !chatStore.connected) {
    chatStore.connectSocket();
  }
}

function openMessageCenter() {
  router.push('/messages');
  isExpanded.value = false;
}

function selectConversation(conversation: ConversationSummary) {
  activeConversation.value = conversation;
  const taskId = conversation.task_id || conversation.application_id || conversation.item_id;
  if (taskId) {
    chatStore.joinConversation(taskId);
    localStorage.setItem('lastActiveConversation', String(taskId));
  }
}

function backToList() {
  activeConversation.value = null;
}

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
  chatStore.handleBookingAction(bookingId, action, rating, review);
}

function sendMessage() {
  if (!messageText.value.trim() || !activeConversation.value) return;
  
  chatStore.sendMessage(messageText.value, activeConversation.value.other_user_id);
  messageText.value = '';
  
  // Stop typing
  if (isTyping.value) {
    isTyping.value = false;
    chatStore.stopTyping();
  }
}

function editMessage(messageId: number, content: string) {
  chatStore.editMessage(messageId, content);
}

function deleteMessage(messageId: number) {
  chatStore.deleteMessage(messageId);
}

function handleTyping() {
  if (!isTyping.value && messageText.value.trim()) {
    isTyping.value = true;
    chatStore.startTyping();
  }
  
  if (typingTimeout.value) {
    clearTimeout(typingTimeout.value);
  }
  
  typingTimeout.value = setTimeout(() => {
    if (isTyping.value) {
      isTyping.value = false;
      chatStore.stopTyping();
    }
  }, 1000);
}

// Auto-scroll to bottom on new messages
watch(() => messages.value.length, () => {
  nextTick(() => {
    if (messagesArea.value) {
      messagesArea.value.scrollTop = messagesArea.value.scrollHeight;
    }
  });
});
</script>

<style scoped>
.chat-widget-container {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  z-index: 1000;
}

/* Widget Button */
.chat-widget-button {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: #4F46E5;
  color: white;
  border: none;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  transition: all 0.2s;
}

.chat-widget-button:hover {
  background: #4338ca;
  transform: scale(1.05);
}

.chat-widget-button.has-unread {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(79, 70, 229, 0.7);
  }
  70% {
    box-shadow: 0 0 0 10px rgba(79, 70, 229, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(79, 70, 229, 0);
  }
}

.chat-widget-button i {
  font-size: 1.5rem;
}

.unread-badge {
  position: absolute;
  top: -5px;
  right: -5px;
  background: #ef4444;
  color: white;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  border-radius: 9999px;
  min-width: 1.25rem;
  text-align: center;
}

/* Expanded Widget */
.chat-widget-expanded {
  width: 400px;
  height: 600px;
  background: white;
  border-radius: 0.75rem;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Widget Header */
.widget-header {
  padding: 1rem;
  background: #4F46E5;
  color: white;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.widget-header h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.expand-btn, .close-btn {
  width: 32px;
  height: 32px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  border-radius: 0.375rem;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.15s;
}

.expand-btn:hover, .close-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* Conversation List */
.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

.conversation-item {
  padding: 0.75rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: background-color 0.15s;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.conversation-item:hover {
  background: #f3f4f6;
}

.conversation-info h4 {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
  margin: 0 0 0.25rem 0;
}

.conversation-info p {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0;
}

.unread-count {
  background: #4F46E5;
  color: white;
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.125rem 0.375rem;
  border-radius: 9999px;
  min-width: 1.25rem;
  text-align: center;
}

.no-conversations {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #9ca3af;
}

/* Active Conversation */
.active-conversation {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.conversation-header {
  padding: 0.75rem;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.back-btn {
  width: 32px;
  height: 32px;
  background: #f3f4f6;
  border: none;
  border-radius: 0.375rem;
  color: #6b7280;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.back-btn:hover {
  background: #e5e7eb;
  color: #111827;
}

.conversation-title {
  flex: 1;
}

.conversation-title h4 {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
  margin: 0 0 0.125rem 0;
}

.conversation-title p {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0;
}

/* Messages Area */
.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

/* Typing Indicator */
.typing-indicator {
  display: flex;
  align-items: center;
  padding: 0.5rem;
}

.typing-dots {
  display: flex;
  gap: 0.25rem;
}

.typing-dots span {
  width: 0.375rem;
  height: 0.375rem;
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

/* Message Input Form */
.message-input-form {
  padding: 0.75rem;
  border-top: 1px solid #e5e7eb;
  display: flex;
  gap: 0.5rem;
}

.message-input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  font-size: 0.875rem;
}

.message-input:focus {
  outline: none;
  border-color: #4F46E5;
}

.message-input-form button {
  padding: 0.5rem 0.75rem;
  background: #4F46E5;
  color: white;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.15s;
}

.message-input-form button:hover:not(:disabled) {
  background: #4338ca;
}

.message-input-form button:disabled {
  background: #e5e7eb;
  cursor: not-allowed;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .chat-widget-container {
    bottom: 1rem;
    right: 1rem;
  }
  
  .chat-widget-expanded {
    width: calc(100vw - 2rem);
    height: calc(100vh - 8rem);
    max-width: 400px;
  }
}
</style>