<template>
  <div class="chat-widget-container">
    <!-- Widget Button -->
    <button
      v-if="!chatStore.widgetOpen"
      class="chat-widget-button"
      @click="toggleWidget"
      :class="{ 'has-unread': chatStore.totalUnreadCount > 0 }"
      aria-label="Open messages"
    >
      <i class="fas fa-comments"></i>
      <span v-if="chatStore.totalUnreadCount > 0" class="unread-badge">
        {{ chatStore.totalUnreadCount }}
      </span>
    </button>

    <!-- Expanded Widget -->
    <div v-if="chatStore.widgetOpen" class="chat-widget-expanded">
      <!-- Widget Header -->
      <div class="widget-header">
        <h3>Messages</h3>
        <div class="header-actions">
          <!-- Inline SVG: crisp at any size and shown immediately, without
               waiting for the icon font -->
          <button
            @click="openMessageCenter"
            class="expand-btn"
            title="Open full Messages page"
            aria-label="Open full Messages page"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button @click="toggleWidget" class="close-btn" title="Close" aria-label="Close messages">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Conversation Switcher -->
      <div v-if="!chatStore.activeConversation" class="conversation-list">
        <div
          v-for="conversation in recentConversations"
          :key="conversationKey(conversation) ?? undefined"
          class="conversation-item"
          @click="chatStore.joinConversation(conversation)"
        >
          <div class="conversation-info">
            <h4>{{ conversation.task_title || conversation.item_title }}</h4>
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

      <!-- Active Conversation: the same thread as on the Messages page -->
      <div v-else class="active-conversation">
        <button @click="chatStore.activeConversation = null" class="back-btn" aria-label="All conversations">
          <i class="fas fa-arrow-left" aria-hidden="true"></i>
          All conversations
        </button>
        <MessageThread
          :conversation="chatStore.activeConversation"
          :messages="chatStore.activeMessages"
          :typing-users="chatStore.activeTypingUsers"
          :loading="chatStore.isLoadingMessages"
          :has-more="chatStore.activeHasMoreMessages"
          @send-message="sendMessage"
          @edit-message="chatStore.editMessage"
          @delete-message="chatStore.deleteMessage"
          @load-more="chatStore.loadMoreMessages"
          @typing-start="chatStore.startTyping"
          @typing-stop="chatStore.stopTyping"
          @booking-action="chatStore.handleBookingAction"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useChatStore, conversationKey } from '@/stores/chat';
import MessageThread from './MessageThread.vue';

const router = useRouter();
const chatStore = useChatStore();

const recentConversations = computed(() => {
  return chatStore.conversations.slice(0, 5);
});

function toggleWidget() {
  chatStore.widgetOpen = !chatStore.widgetOpen;

  if (chatStore.widgetOpen && !chatStore.connected) {
    chatStore.connectSocket();
  }
}

function openMessageCenter() {
  router.push('/messages');
  chatStore.widgetOpen = false;
}

function sendMessage(content: string) {
  const conversation = chatStore.activeConversation;
  if (conversation) chatStore.sendMessage(content, conversation.other_user_id);
}
</script>

<style scoped>
.chat-widget-container {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  z-index: 900; /* below modals and the mobile nav drawer */
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
  color: white; /* global heading colour is dark, unreadable on the purple header */
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.expand-btn, .close-btn {
  width: 36px;
  height: 36px;
  background: rgba(255, 255, 255, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.45);
  border-radius: 0.375rem;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.15s;
}

.expand-btn:hover, .close-btn:hover,
.expand-btn:focus-visible, .close-btn:focus-visible {
  background: rgba(255, 255, 255, 0.4);
  outline: none;
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
  flex: 1;
  min-height: 0;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: #f9fafb;
  border: none;
  border-bottom: 1px solid #e5e7eb;
  color: #4b5563;
  font-size: 0.875rem;
  cursor: pointer;
  text-align: left;
}

.back-btn:hover,
.back-btn:focus-visible {
  background: #f3f4f6;
  color: #111827;
}

/* The thread's own header is roomy for the Messages page; tighten it here */
.active-conversation :deep(.thread-header) {
  padding: 0.75rem 1rem;
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
    height: calc(100dvh - 8rem);
    max-width: 400px;
  }
}
</style>
