<template>
  <div class="chat-widget-container" :style="actionBarHeight ? { bottom: `${actionBarHeight + 16}px` } : undefined">
    <!-- Widget Button -->
    <button
      v-if="!chatStore.widgetOpen && !route.meta.hideChatButton"
      class="chat-widget-button"
      @click="toggleWidget"
      :aria-label="chatStore.totalUnreadCount > 0 ? `Open messages, ${chatStore.totalUnreadCount} unread` : 'Open messages'"
    >
      <svg width="26" height="26" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path d="M3 5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2H9l-4 3.5V14a2 2 0 0 1-2-2z" />
        <path d="M19 8h0a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2v2.5L15.5 19H11a2 2 0 0 1-1.7-1h5.7a3 3 0 0 0 3-3z" opacity="0.7" />
      </svg>
      <span v-if="chatStore.totalUnreadCount > 0" class="unread-dot"></span>
    </button>

    <!-- Expanded: the list of conversations, or one of them. Full screen on
         phones, a panel on wider screens. -->
    <div v-if="chatStore.widgetOpen" class="chat-widget-expanded" role="dialog" aria-label="Messages">
      <template v-if="!chatStore.activeConversation">
        <div class="widget-header">
          <button @click="toggleWidget" class="icon-btn" aria-label="Close messages">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M15 5l-7 7 7 7" />
            </svg>
          </button>
          <h3>Messages</h3>
        </div>

        <div class="list-tabs" role="tablist" aria-label="Which conversations">
          <button
            role="tab"
            :aria-selected="listTab === 'active'"
            :class="['list-tab', { active: listTab === 'active' }]"
            @click="listTab = 'active'"
          >
            Active <span class="tab-count">{{ activeCount }}</span>
          </button>
          <button
            role="tab"
            :aria-selected="listTab === 'all'"
            :class="['list-tab', { active: listTab === 'all' }]"
            @click="listTab = 'all'"
          >
            All
          </button>
        </div>

        <div class="conversation-list">
          <!-- Conversations about finished gigs are deleted after 30 days -->
          <DeletionWarningBanner
            v-if="chatStore.deletionWarnings.length > 0"
            :warnings="chatStore.deletionWarnings"
            @dismiss="chatStore.dismissWarning"
          />
          <button
            v-for="conversation in listed"
            :key="conversationKey(conversation) ?? undefined"
            type="button"
            :class="['conversation-item', { ended: isEnded(conversation) }]"
            @click="chatStore.joinConversation(conversation)"
          >
            <span class="conversation-avatar" aria-hidden="true">{{ initialOf(conversation.other_user_name) }}</span>
            <span class="conversation-info">
              <span class="conversation-title">{{ titleOf(conversation) }}</span>
              <span class="visually-hidden">with {{ conversation.other_user_name }}</span>
              <!-- Where the deal stands, and the rating between you once given -->
              <span class="conversation-status">
                <span v-if="isEnded(conversation)" class="status-chip expired">Expired</span>
                <span v-if="statusLabel(conversation)" :class="['status-chip', statusIcon(conversation)?.tone]">
                  <i v-if="statusIcon(conversation)" :class="['fas', statusIcon(conversation)!.icon]" aria-hidden="true"></i>
                  {{ statusLabel(conversation) }}
                </span>
                <span v-else-if="!isEnded(conversation)" class="status-chip going">Active</span>
                <span v-if="isEnded(conversation) && conversation.state_at" class="conversation-ago">{{ timeAgo(conversation.state_at) }}</span>
                <span v-if="ratingOf(conversation) !== null" class="conversation-rating">★ {{ formatRating(ratingOf(conversation)!) }}</span>
              </span>
            </span>
            <span
              v-if="conversation.unread_count > 0"
              class="unread-mark"
              role="img"
              :aria-label="`${conversation.unread_count} unread`"
            ></span>
          </button>

          <p v-if="chatStore.conversations.length === 0" class="no-conversations">No conversations yet</p>
          <p v-else-if="listed.length === 0" class="no-conversations">Nothing active right now</p>
          <p class="list-footer">You can type once the poster or seller says yes</p>
        </div>
      </template>

      <!-- One conversation: its header has the way back to the list -->
      <div v-else class="active-conversation">
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
        >
          <template #back>
            <button @click="chatStore.activeConversation = null" class="icon-btn" aria-label="All conversations">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M15 5l-7 7 7 7" />
              </svg>
            </button>
          </template>
        </MessageThread>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { actionBarHeight } from '@/composables/useActionBar';
import { useChatStore, conversationKey } from '@/stores/chat';
import { useReviewsStore } from '@/stores/reviews';
import { statusLabel, statusIcon, timeAgo } from '@/utils/conversationStatus';
import { formatRating, type Interaction } from '@/utils/network';
import type { ConversationSummary } from '@/stores/messages';
import MessageThread from './MessageThread.vue';
import DeletionWarningBanner from '@/components/shared/DeletionWarningBanner.vue';

const chatStore = useChatStore();
const route = useRoute();
const reviewsStore = useReviewsStore();

function isEnded(conversation: ConversationSummary): boolean {
  return !!conversation.ended || chatStore.endedConversations.has(conversationKey(conversation) ?? '');
}

// Active: deals still under way. All (the default): everything, ended last.
const listTab = ref<'active' | 'all'>('all');
const activeCount = computed(() => chatStore.sortedConversations.filter(c => !isEnded(c)).length);
const listed = computed(() =>
  listTab.value === 'active' ? chatStore.sortedConversations.filter(c => !isEnded(c)) : chatStore.sortedConversations
);

const titleOf = (c: ConversationSummary) =>
  c.task_title || c.item_title || (c.task_id ? `Gig #${c.task_id}` : `Item #${c.item_id}`);
const initialOf = (name?: string | null) => (name || '?').charAt(0).toUpperCase();

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
  bottom: 20px;
  right: 16px;
  z-index: 900; /* below modals and the menu drawer */
}

/* Floating chat button */
.chat-widget-button {
  width: 56px;
  height: 56px;
  border-radius: 28px;
  background: var(--accent);
  color: var(--on-accent);
  border: none;
  box-shadow: 0 4px 12px rgba(245, 138, 122, 0.4);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  transition: transform 0.15s;
}

.chat-widget-button:hover {
  transform: scale(1.05);
}

.chat-widget-button:focus-visible {
  outline: 3px solid var(--accent-text);
  outline-offset: 3px;
}

@media (prefers-reduced-motion: reduce) {
  .chat-widget-button {
    transition: none;
  }
}

.unread-dot {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 14px;
  height: 14px;
  border-radius: 7px;
  background: var(--badge-red);
  border: 2px solid var(--bg);
}

/* Expanded: a panel; full screen on phones */
.chat-widget-expanded {
  width: 400px;
  height: 600px;
  max-height: calc(100dvh - 40px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 16px;
  background: var(--surface);
  box-shadow: 0 20px 25px -5px rgba(17, 24, 39, 0.12), 0 10px 10px -5px rgba(17, 24, 39, 0.05);
}

@media (max-width: 640px) {
  .chat-widget-expanded {
    position: fixed;
    inset: 0;
    width: auto;
    height: auto;
    max-height: none;
    border-radius: 0;
  }
}

.widget-header {
  flex: none;
  display: flex;
  align-items: center;
  gap: 4px;
  height: 56px;
  padding: 0 8px;
  border-bottom: 1px solid #EEF0F3;
}

.widget-header h3 {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--text);
}

.icon-btn {
  flex: none;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  border-radius: 22px;
  background: transparent;
  color: #374151;
  cursor: pointer;
}

.icon-btn:hover,
.icon-btn:focus-visible {
  background: rgba(17, 24, 39, 0.05);
}

/* Active / All */
.list-tabs {
  flex: none;
  display: flex;
  gap: 24px;
  padding: 0 20px;
  border-bottom: 1px solid #EEF0F3;
}

.list-tab {
  height: 48px;
  padding: 0;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--text-muted);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
}

.list-tab.active {
  border-bottom-color: var(--accent);
  color: var(--text);
  font-weight: 600;
}

.tab-count {
  color: var(--accent-text);
  font-weight: 600;
}

/* Conversation List */
.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 0 20px;
}

.conversation-item {
  width: 100%;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 0;
  border: 0;
  border-bottom: 1px solid #F3F4F6;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.conversation-item:hover .conversation-title,
.conversation-item:focus-visible .conversation-title {
  text-decoration: underline;
}

.conversation-avatar {
  flex: none;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 22px;
  background: var(--accent-tint);
  color: var(--accent-text);
  font-size: 17px;
  font-weight: 600;
}

.conversation-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.conversation-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Done or closed: still opens, shown quieter */
.conversation-item.ended .conversation-title {
  color: var(--text-muted);
}

.conversation-status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 2px;
  font-size: 13px;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 10px;
  border-radius: 14px;
  background: #F3F4F6;
  color: #4B5563;
  font-weight: 600;
}

/* Tones: under way, done (green tick), waiting on someone, stopped */
.status-chip.going {
  background: var(--accent-tint);
  color: var(--accent-text);
}

.status-chip.done {
  background: #DCFCE7;
  color: #15803D;
}

.status-chip.waiting {
  background: #FEF3C7;
  color: #92400E;
}

.status-chip.stopped {
  background: #FEE2E2;
  color: #B91C1C;
}

.conversation-ago {
  color: var(--text-muted);
}

.conversation-rating {
  color: #B45309;
  font-weight: 700;
}

.unread-mark {
  flex: none;
  width: 10px;
  height: 10px;
  margin-top: 8px;
  border-radius: 5px;
  background: var(--accent);
}

.no-conversations {
  margin: 0;
  padding: 2rem 0 0;
  text-align: center;
  color: #9CA3AF;
}

.list-footer {
  margin: 0;
  padding: 20px 0;
  font-size: 13px;
  text-align: center;
  color: #9CA3AF;
}

.active-conversation {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
</style>
