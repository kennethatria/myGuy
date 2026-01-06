<template>
  <Teleport to="body">
    <div v-if="isOpen" class="modal-backdrop" @click="handleBackdropClick">
      <div class="modal-container" @click.stop>
        <!-- Header -->
        <div class="modal-header">
          <h2>✅ Request Submitted!</h2>
          <button class="close-button" @click="close" aria-label="Close">
            <i class="fas fa-times"></i>
          </button>
        </div>

        <!-- Body -->
        <div class="modal-body">
          <!-- Item Info Card -->
          <div class="booking-info-card">
            <img
              v-if="itemImage"
              :src="itemImageUrl"
              :alt="itemTitle"
              class="item-thumbnail"
            />
            <div class="item-placeholder" v-else>
              <i class="fas fa-image"></i>
            </div>
            <div class="item-details">
              <h3>{{ itemTitle }}</h3>
              <p>Your booking request has been sent to {{ sellerName }}</p>
            </div>
          </div>

          <!-- Loading State -->
          <div v-if="loading" class="loading-state">
            <div class="spinner"></div>
            <p>Loading conversation...</p>
          </div>

          <!-- Error State -->
          <div v-else-if="error" class="error-state">
            <i class="fas fa-exclamation-circle"></i>
            <p>{{ error }}</p>
            <button @click="retryLoadConversation" class="btn-retry">
              Try Again
            </button>
          </div>

          <!-- Chat Section -->
          <div v-else-if="conversationReady" class="chat-section">
            <h4>Send a message to {{ sellerName }} (optional)</h4>

            <!-- Messages Preview -->
            <div v-if="messages.length > 0" class="messages-preview">
              <div
                v-for="message in messages.slice(-3)"
                :key="message.id"
                class="message-preview-item"
                :class="{ 'own-message': message.sender_id === currentUserId }"
              >
                <div class="message-sender">{{ message.sender?.username || 'Unknown' }}</div>
                <div class="message-content">{{ message.content }}</div>
              </div>
            </div>

            <!-- Empty state message -->
            <div v-else class="no-messages-hint">
              <i class="fas fa-comments"></i>
              <p>Start the conversation! Ask about pickup details, condition, or anything else.</p>
            </div>

            <!-- Message Input -->
            <div class="message-input-container">
              <textarea
                v-model="messageText"
                placeholder="e.g., When can I pick this up?"
                rows="3"
                :disabled="sending"
              ></textarea>
              <button
                @click="sendMessage"
                :disabled="!messageText.trim() || sending"
                class="btn-send"
              >
                <i class="fas fa-paper-plane" v-if="!sending"></i>
                <div class="spinner-small" v-else></div>
                {{ sending ? 'Sending...' : 'Send Message' }}
              </button>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="modal-footer">
          <button @click="goToMessages" class="btn-primary">
            <i class="fas fa-comments"></i>
            View in Messages
          </button>
          <button @click="close" class="btn-secondary">
            Stay on Page
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useChatStore } from '@/stores/chat';
import { useAuthStore } from '@/stores/auth';
import config from '@/config';
import type { Message } from '@/stores/messages';

interface Props {
  isOpen: boolean;
  itemId: number;
  itemTitle: string;
  itemImage?: string | object | null;
  sellerId: number;
  sellerName: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: 'close'): void;
}>();

const router = useRouter();
const chatStore = useChatStore();
const authStore = useAuthStore();

// State
const messageText = ref('');
const messages = ref<Message[]>([]);
const loading = ref(false);
const sending = ref(false);
const conversationReady = ref(false);
const error = ref<string | null>(null);

// Computed
const currentUserId = computed(() => authStore.user?.id);

const itemImageUrl = computed(() => {
  if (!props.itemImage) return '';

  // Handle object with url property (e.g., { url: '/uploads/...' })
  if (typeof props.itemImage === 'object' && props.itemImage !== null) {
    const imageObj = props.itemImage as any;
    const url = imageObj.url || imageObj.path || '';
    if (!url) return '';

    // Check if it's a full URL
    if (url.startsWith('http')) {
      return url;
    }
    // Use STORE_API_BASE_URL (without /api/v1) for image paths
    return `${config.STORE_API_BASE_URL}${url}`;
  }

  // Handle string (legacy format)
  if (typeof props.itemImage === 'string') {
    if (props.itemImage.startsWith('http')) {
      return props.itemImage;
    }
    // Use STORE_API_BASE_URL (without /api/v1) for image paths
    return `${config.STORE_API_BASE_URL}${props.itemImage}`;
  }

  return '';
});

// Methods
async function loadConversation() {
  loading.value = true;
  error.value = null;

  try {
    // Try to join conversation (best effort - don't fail if it doesn't exist yet)
    await chatStore.joinStoreConversationWithRetry(props.itemId, 2).catch(() => {
      // Ignore errors - conversation might not exist yet, which is fine
      console.log('No existing conversation found - user can still send messages');
    });

    // Try to load any existing messages (optional - won't fail if none exist)
    const allMessages = chatStore.getStoreMessages(props.itemId);

    // Only show messages between current user and seller (privacy filter)
    messages.value = allMessages.filter(msg => {
      const isFromCurrentUser = msg.sender_id === currentUserId.value;
      const isToCurrentUser = msg.recipient_id === currentUserId.value;
      const isFromSeller = msg.sender_id === props.sellerId;
      const isToSeller = msg.recipient_id === props.sellerId;

      // Include message if it's between current user and seller
      return (isFromCurrentUser && isToSeller) || (isFromSeller && isToCurrentUser);
    });

    // Always set conversation as ready - user can send messages even if no history exists
    conversationReady.value = true;
  } catch (err) {
    console.error('Error loading messages:', err);
    // Don't set error - still allow user to send messages
    conversationReady.value = true;
  } finally {
    loading.value = false;
  }
}

async function retryLoadConversation() {
  await loadConversation();
}

async function sendMessage() {
  if (!messageText.value.trim() || sending.value) return;

  sending.value = true;

  try {
    await chatStore.sendStoreMessage(
      messageText.value,
      props.sellerId,
      props.itemId
    );

    // Clear input after sending
    messageText.value = '';

    // Refresh messages with privacy filter
    const allMessages = chatStore.getStoreMessages(props.itemId);
    messages.value = allMessages.filter(msg => {
      const isFromCurrentUser = msg.sender_id === currentUserId.value;
      const isToCurrentUser = msg.recipient_id === currentUserId.value;
      const isFromSeller = msg.sender_id === props.sellerId;
      const isToSeller = msg.recipient_id === props.sellerId;

      return (isFromCurrentUser && isToSeller) || (isFromSeller && isToCurrentUser);
    });
  } catch (err) {
    console.error('Error sending message:', err);
    alert('Failed to send message. Please try again.');
  } finally {
    sending.value = false;
  }
}

function goToMessages() {
  router.push({ path: '/messages', query: { itemId: props.itemId } });
  close();
}

function close() {
  emit('close');
}

function handleBackdropClick() {
  close();
}

// Watch for modal opening
watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    // Reset state
    messageText.value = '';
    messages.value = [];
    conversationReady.value = false;
    error.value = null;

    // Load conversation
    loadConversation();
  }
});
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-container {
  background: white;
  border-radius: 0.75rem;
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
}

.modal-header {
  padding: 1.5rem;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #111827;
}

.close-button {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: #6b7280;
  cursor: pointer;
  padding: 0.5rem;
  border-radius: 0.375rem;
  transition: all 0.2s;
}

.close-button:hover {
  background: #f3f4f6;
  color: #111827;
}

.modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
}

.booking-info-card {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  background: #f0f9ff;
  border: 1px solid #bfdbfe;
  border-radius: 0.5rem;
  margin-bottom: 1.5rem;
}

.item-thumbnail {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border-radius: 0.375rem;
  flex-shrink: 0;
}

.item-placeholder {
  width: 64px;
  height: 64px;
  background: #e5e7eb;
  border-radius: 0.375rem;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
  font-size: 1.5rem;
  flex-shrink: 0;
}

.item-details h3 {
  margin: 0 0 0.5rem 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
}

.item-details p {
  margin: 0;
  color: #6b7280;
  font-size: 0.875rem;
}

.loading-state,
.error-state {
  text-align: center;
  padding: 2rem;
  color: #6b7280;
}

.spinner {
  border: 3px solid #f3f4f6;
  border-top: 3px solid #3b82f6;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.spinner-small {
  border: 2px solid #f3f4f6;
  border-top: 2px solid white;
  border-radius: 50%;
  width: 16px;
  height: 16px;
  animation: spin 1s linear infinite;
  display: inline-block;
}

.error-state i {
  font-size: 2rem;
  color: #ef4444;
  margin-bottom: 0.5rem;
}

.btn-retry {
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-weight: 500;
  transition: background 0.2s;
}

.btn-retry:hover {
  background: #2563eb;
}

.chat-section h4 {
  margin: 0 0 1rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: #374151;
}

.no-messages-hint {
  text-align: center;
  padding: 2rem 1rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  margin-bottom: 1rem;
}

.no-messages-hint i {
  font-size: 2rem;
  color: #9ca3af;
  margin-bottom: 0.5rem;
}

.no-messages-hint p {
  margin: 0;
  color: #6b7280;
  font-size: 0.875rem;
  line-height: 1.5;
}

.messages-preview {
  max-height: 200px;
  overflow-y: auto;
  margin-bottom: 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  padding: 0.75rem;
  background: #f9fafb;
}

.message-preview-item {
  margin-bottom: 0.75rem;
  padding: 0.5rem;
  background: white;
  border-radius: 0.375rem;
  border-left: 3px solid #3b82f6;
}

.message-preview-item.own-message {
  border-left-color: #10b981;
}

.message-preview-item:last-child {
  margin-bottom: 0;
}

.message-sender {
  font-size: 0.75rem;
  font-weight: 600;
  color: #6b7280;
  margin-bottom: 0.25rem;
}

.message-content {
  font-size: 0.875rem;
  color: #111827;
}

.message-input-container {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.message-input-container textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-family: inherit;
  font-size: 0.875rem;
  resize: vertical;
  transition: border-color 0.2s;
}

.message-input-container textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.message-input-container textarea:disabled {
  background: #f3f4f6;
  cursor: not-allowed;
}

.btn-send {
  padding: 0.75rem 1.5rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  font-weight: 500;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  transition: background 0.2s;
}

.btn-send:hover:not(:disabled) {
  background: #2563eb;
}

.btn-send:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}

.modal-footer {
  padding: 1.5rem;
  border-top: 1px solid #e5e7eb;
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
}

.btn-primary,
.btn-secondary {
  padding: 0.75rem 1.5rem;
  border-radius: 0.5rem;
  font-weight: 500;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-primary {
  background: #3b82f6;
  color: white;
  border: none;
}

.btn-primary:hover {
  background: #2563eb;
}

.btn-secondary {
  background: white;
  color: #374151;
  border: 1px solid #d1d5db;
}

.btn-secondary:hover {
  background: #f9fafb;
}

/* Mobile Responsive */
@media (max-width: 640px) {
  .modal-backdrop {
    padding: 0;
  }

  .modal-container {
    width: 100%;
    max-width: none;
    max-height: 100vh;
    border-radius: 0;
  }

  .modal-header {
    padding: 1rem;
  }

  .modal-header h2 {
    font-size: 1.25rem;
  }

  .modal-body {
    padding: 1rem;
  }

  .modal-footer {
    padding: 1rem;
    flex-direction: column-reverse;
  }

  .btn-primary,
  .btn-secondary {
    width: 100%;
    justify-content: center;
  }
}
</style>
