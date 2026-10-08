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
          <button @click="toggleWidget" class="close-btn" title="Close" aria-label="Close messages">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Conversation Switcher -->
      <div v-if="!chatStore.activeConversation" class="conversation-list">
        <!-- Conversations about finished gigs are deleted after 30 days -->
        <DeletionWarningBanner
          v-if="chatStore.deletionWarnings.length > 0"
          :warnings="chatStore.deletionWarnings"
          @dismiss="chatStore.dismissWarning"
        />
        <div
          v-for="conversation in chatStore.sortedConversations"
          :key="conversationKey(conversation) ?? undefined"
          :class="['conversation-item', { ended: isEnded(conversation) }]"
          @click="chatStore.joinConversation(conversation)"
        >
          <div class="conversation-info">
            <h4>{{ conversation.task_title || conversation.item_title || (conversation.task_id ? `Gig #${conversation.task_id}` : `Item #${conversation.item_id}`) }}</h4>
            <p>{{ conversation.other_user_name }}</p>
            <!-- Where the deal stands, and the rating between you once given -->
            <p class="conversation-status">
              <span v-if="isEnded(conversation)" class="status-chip expired">Expired</span>
              <span v-if="statusLabel(conversation)" :class="['status-chip', statusIcon(conversation)?.tone]">
                <i v-if="statusIcon(conversation)" :class="['fas', statusIcon(conversation)!.icon]" aria-hidden="true"></i>
                {{ statusLabel(conversation) }}
              </span>
              <span v-if="isEnded(conversation) && conversation.state_at" class="conversation-ago">{{ timeAgo(conversation.state_at) }}</span>
              <span v-if="ratingOf(conversation) !== null" class="conversation-rating">★ {{ formatRating(ratingOf(conversation)!) }}</span>
            </p>
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
          :locked="chatStore.activeLocked"
          :ended="chatStore.activeEnded"
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
import { ref, watch } from 'vue';
import { useChatStore, conversationKey } from '@/stores/chat';
import { useReviewsStore } from '@/stores/reviews';
import { statusLabel, statusIcon, timeAgo } from '@/utils/conversationStatus';
import { formatRating, type Interaction } from '@/utils/network';
import type { ConversationSummary } from '@/stores/messages';
import MessageThread from './MessageThread.vue';
import DeletionWarningBanner from '@/components/shared/DeletionWarningBanner.vue';

const chatStore = useChatStore();
const reviewsStore = useReviewsStore();

function isEnded(conversation: ConversationSummary): boolean {
  return !!conversation.ended || chatStore.endedConversations.has(conversationKey(conversation) ?? '');
}

// Your reviews with others, for the average rating between you on each
// conversation's gig or item (refreshed whenever the chat opens)
const interactions = ref<Interaction[]>([]);
watch(() => chatStore.widgetOpen, async (open) => {
  if (!open) return;
  try {
    interactions.value = await reviewsStore.fetchInteractions();
  } catch {
    // Ratings are extra: the list works without them
  }
}, { immediate: true });

function ratingOf(conversation: ConversationSummary): number | null {
  const ratings = interactions.value
    .filter(i => i.otherId === conversation.other_user_id &&
      (conversation.task_id ? i.via === 'gig' && i.linkId === conversation.task_id
        : i.via === 'item' && i.linkId === conversation.item_id))
    .map(i => i.rating);
  if (!ratings.length) return null;
  return Math.round((ratings.reduce((sum, r) => sum + r, 0) / ratings.length) * 10) / 10;
}

function toggleWidget() {
  chatStore.widgetOpen = !chatStore.widgetOpen;

  if (chatStore.widgetOpen) {
    if (!chatStore.connected) chatStore.connectSocket();
    chatStore.loadDeletionWarnings();
  }
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

.close-btn {
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

.close-btn:hover,
.close-btn:focus-visible {
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

.conversation-status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.375rem;
  margin-top: 0.25rem !important;
}

.status-chip {
  padding: 0 0.4rem;
  border-radius: 999px;
  background: #eef2ff;
  color: #3730a3;
  font-weight: 600;
}

.status-chip i {
  margin-right: 0.2rem;
}

/* Tones: done (green tick), waiting on someone, under way, stopped */
.status-chip.done {
  background: #dcfce7;
  color: #15803d;
}

.status-chip.waiting {
  background: #fef3c7;
  color: #92400e;
}

.status-chip.stopped {
  background: #fee2e2;
  color: #b91c1c;
}

.status-chip.neutral {
  background: #f3f4f6;
  color: #4b5563;
}

.status-chip.expired {
  background: #f3f4f6;
  color: #6b7280;
}

.conversation-ago {
  color: #6b7280;
}

.conversation-rating {
  color: #b45309;
  font-weight: 700;
}

/* Done or closed: still opens, shown quieter */
.conversation-item.ended h4 {
  color: #6b7280;
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
