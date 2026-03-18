<template>
  <div class="chat-window">
    <!-- Chat Header -->
    <div class="chat-header">
      <h3>{{ headerTitle }}</h3>
      <button v-if="showCloseButton" @click="$emit('close')" class="close-btn">
        <i class="bi bi-x-lg"></i>
      </button>
    </div>

    <!-- Chat Messages -->
    <div class="chat-messages" ref="messagesContainer">
      <!-- Loading State -->
      <div v-if="chatStore.isLoadingMessages" class="loading-state">
        <div class="spinner-border spinner-border-sm" role="status">
          <span class="visually-hidden">Loading messages...</span>
        </div>
        <span class="ms-2">Loading messages...</span>
      </div>

      <!-- Load More Button -->
      <div v-if="chatStore.activeHasMoreMessages && !chatStore.isLoadingMessages" class="load-more-container">
        <button @click="loadMoreMessages" class="btn btn-sm btn-outline-secondary">
          Load older messages
        </button>
      </div>

      <!-- No Messages State -->
      <div v-if="messages.length === 0 && !chatStore.isLoadingMessages" class="no-messages">
        <i class="bi bi-chat-dots"></i>
        <p>No messages yet</p>
        <p class="text-muted small">Start the conversation!</p>
      </div>

      <!-- Message List -->
      <div v-for="message in messages" :key="message.id"
           class="message"
           :class="{ 'own-message': message.sender_id === authStore.user?.id }">
        <div class="message-header">
          <span class="sender">
            {{ message.sender_id === authStore.user?.id ? 'You' : (message.sender?.username || 'User') }}
          </span>
          <span class="timestamp">{{ formatMessageTime(message.created_at) }}</span>
        </div>
        <div class="message-content">
          {{ message.content }}
          <span v-if="message.is_edited" class="edited-badge">(edited)</span>
        </div>
        <div v-if="message.is_read && message.sender_id === authStore.user?.id" class="read-indicator">
          <i class="bi bi-check-all"></i> Read
        </div>
      </div>

      <!-- Typing Indicator -->
      <div v-if="chatStore.activeTypingUsers.length > 0" class="typing-indicator">
        <div class="typing-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
        <span class="typing-text">
          {{ chatStore.activeTypingUsers[0].userName }} is typing...
        </span>
      </div>
    </div>

    <!-- Chat Input -->
    <div v-if="!hideInput" class="chat-input-section">
      <div class="chat-input">
        <textarea
          v-model="newMessage"
          @keydown.enter.prevent="handleSendMessage"
          @input="handleTyping"
          placeholder="Type your message..."
          rows="2"
          :disabled="sending"
        ></textarea>
        <button
          @click="handleSendMessage"
          class="btn btn-primary send-btn"
          :disabled="!canSend">
          <i class="bi bi-send-fill"></i>
          <span class="d-none d-sm-inline ms-1">Send</span>
        </button>
      </div>
      <div v-if="error" class="error-message">
        {{ error }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import { useChatStore } from '@/stores/chat';
import { useAuthStore } from '@/stores/auth';
import { format, formatDistanceToNow } from 'date-fns';

// Props
interface Props {
  conversationId: number;
  conversationType: 'task' | 'application' | 'store';
  recipientId: number;
  recipientName?: string;
  conversationTitle?: string;
  showCloseButton?: boolean;
  hideInput?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showCloseButton: false,
  recipientName: 'User',
  conversationTitle: '',
  hideInput: false
});

// Emits
defineEmits<{
  close: [];
}>();

// Stores
const chatStore = useChatStore();
const authStore = useAuthStore();

// State
const newMessage = ref('');
const sending = ref(false);
const error = ref('');
const messagesContainer = ref<HTMLElement | null>(null);
const typingTimeout = ref<number | null>(null);
const isTyping = ref(false);

// Computed
const messages = computed(() => {
  if (props.conversationType === 'store') {
    return chatStore.getStoreMessages(props.conversationId);
  }
  return chatStore.activeMessages;
});

const canSend = computed(() => {
  return newMessage.value.trim().length > 0 && !sending.value;
});

const headerTitle = computed(() => {
  if (props.conversationTitle) {
    return props.conversationTitle;
  }
  return props.recipientName ? `Conversation with ${props.recipientName}` : 'Messages';
});

// Methods
function formatMessageTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const diffInHours = (now.getTime() - date.getTime()) / (1000 * 60 * 60);

    if (diffInHours < 24) {
      return formatDistanceToNow(date, { addSuffix: true });
    } else {
      return format(date, 'MMM d, h:mm a');
    }
  } catch {
    return timestamp;
  }
}

async function handleSendMessage() {
  if (!canSend.value) return;

  const content = newMessage.value.trim();
  sending.value = true;
  error.value = '';

  try {
    if (props.conversationType === 'store') {
      await chatStore.sendStoreMessage(content, props.recipientId, props.conversationId);
    } else {
      await chatStore.sendMessage(content, props.recipientId);
    }

    newMessage.value = '';
    stopTyping();

    // Scroll to bottom after sending
    await nextTick();
    scrollToBottom();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to send message';
    console.error('Error sending message:', e);
  } finally {
    sending.value = false;
  }
}

function handleTyping() {
  if (!isTyping.value) {
    isTyping.value = true;
    chatStore.startTyping();
  }

  // Clear existing timeout
  if (typingTimeout.value) {
    clearTimeout(typingTimeout.value);
  }

  // Set new timeout to stop typing after 2 seconds of inactivity
  typingTimeout.value = window.setTimeout(() => {
    stopTyping();
  }, 2000);
}

function stopTyping() {
  if (isTyping.value) {
    isTyping.value = false;
    chatStore.stopTyping();
  }
  if (typingTimeout.value) {
    clearTimeout(typingTimeout.value);
    typingTimeout.value = null;
  }
}

function loadMoreMessages() {
  chatStore.loadMoreMessages();
}

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
  }
}

// Lifecycle
onMounted(async () => {
  // Connect socket if not connected
  if (!chatStore.connected) {
    await chatStore.connectSocket();
  }

  // Join the conversation
  if (props.conversationType === 'store') {
    await chatStore.joinStoreConversation(props.conversationId);
  } else {
    chatStore.joinConversation(props.conversationId);
  }

  // Scroll to bottom initially
  await nextTick();
  scrollToBottom();
});

onUnmounted(() => {
  stopTyping();
});

// Watch for new messages and scroll to bottom
watch(() => messages.value.length, async () => {
  await nextTick();
  scrollToBottom();
});
</script>

<style scoped>
.chat-window {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: white;
  border-radius: 8px;
  overflow: hidden;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.chat-header h3 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  color: white;
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  opacity: 0.8;
  transition: opacity 0.2s;
}

.close-btn:hover {
  opacity: 1;
}

.chat-messages {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
  background: #f8f9fa;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  color: #6c757d;
}

.load-more-container {
  display: flex;
  justify-content: center;
  padding: 0.5rem 0;
}

.no-messages {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  text-align: center;
  color: #6c757d;
}

.no-messages i {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.3;
}

.message {
  display: flex;
  flex-direction: column;
  max-width: 70%;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  background: white;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  animation: slideIn 0.2s ease-out;
}

.message.own-message {
  align-self: flex-end;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.25rem;
  font-size: 0.85rem;
}

.sender {
  font-weight: 600;
}

.own-message .sender {
  color: rgba(255, 255, 255, 0.9);
}

.timestamp {
  color: #6c757d;
  font-size: 0.75rem;
}

.own-message .timestamp {
  color: rgba(255, 255, 255, 0.7);
}

.message-content {
  word-wrap: break-word;
  line-height: 1.4;
}

.edited-badge {
  font-size: 0.75rem;
  opacity: 0.7;
  margin-left: 0.5rem;
}

.read-indicator {
  font-size: 0.75rem;
  text-align: right;
  margin-top: 0.25rem;
  opacity: 0.7;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.25rem;
}

.typing-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  background: white;
  border-radius: 12px;
  max-width: 200px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.typing-dots {
  display: flex;
  gap: 0.25rem;
}

.typing-dots span {
  width: 6px;
  height: 6px;
  background: #667eea;
  border-radius: 50%;
  animation: typingDot 1.4s infinite;
}

.typing-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

.typing-text {
  font-size: 0.85rem;
  color: #6c757d;
}

.chat-input-section {
  padding: 1rem;
  background: white;
  border-top: 1px solid #dee2e6;
}

.chat-input {
  display: flex;
  gap: 0.5rem;
  align-items: flex-end;
}

.chat-input textarea {
  flex: 1;
  padding: 0.75rem;
  border: 1px solid #dee2e6;
  border-radius: 8px;
  resize: none;
  font-family: inherit;
  font-size: 0.95rem;
  transition: border-color 0.2s;
}

.chat-input textarea:focus {
  outline: none;
  border-color: #667eea;
}

.chat-input textarea:disabled {
  background: #f8f9fa;
  cursor: not-allowed;
}

.send-btn {
  padding: 0.75rem 1.25rem;
  border-radius: 8px;
  white-space: nowrap;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.error-message {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: #f8d7da;
  color: #721c24;
  border-radius: 4px;
  font-size: 0.875rem;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes typingDot {
  0%, 60%, 100% {
    transform: translateY(0);
  }
  30% {
    transform: translateY(-8px);
  }
}

/* Scrollbar styling */
.chat-messages::-webkit-scrollbar {
  width: 8px;
}

.chat-messages::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: #888;
  border-radius: 4px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: #555;
}
</style>
