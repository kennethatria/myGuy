<template>
  <div class="message-center">
    <!-- Deletion Warnings -->
    <DeletionWarningBanner 
      v-if="chatStore.deletionWarnings.length > 0"
      :warnings="chatStore.deletionWarnings"
      @dismiss="chatStore.dismissWarning"
    />
    
    <div class="message-center-layout">
      <!-- Conversations List -->
      <div class="conversations-sidebar">
        <div class="sidebar-header">
          <h2>Messages</h2>
          <div class="total-unread" v-if="chatStore.totalUnreadCount > 0">
            {{ chatStore.totalUnreadCount }}
          </div>
        </div>
        
        <div class="conversations-list">
          <ConversationItem
            v-for="conversation in chatStore.conversations"
            :key="conversation.task_id || conversation.application_id || conversation.item_id"
            :conversation="conversation"
            :active="chatStore.activeConversation ? ((chatStore.activeConversation.task_id === conversation.task_id) || (chatStore.activeConversation.application_id === conversation.application_id) || (chatStore.activeConversation.item_id === conversation.item_id)) : false"
            @click="selectConversation(conversation)"
          />
        </div>
      </div>
      
      <!-- Message Thread -->
      <div class="message-thread-container">
        <MessageThread
          v-if="chatStore.activeConversation"
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
        />
        
        <div v-else class="no-conversation">
          <i class="fas fa-comments"></i>
          <p>Select a conversation to start messaging</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { useChatStore } from '@/stores/chat';
import ConversationItem from '@/components/messages/ConversationItem.vue';
import MessageThread from '@/components/messages/MessageThread.vue';
import DeletionWarningBanner from '@/components/shared/DeletionWarningBanner.vue';
import type { ConversationSummary } from '@/stores/messages';

const chatStore = useChatStore();

onMounted(() => {
  chatStore.connectSocket();
  chatStore.loadDeletionWarnings();
});

onUnmounted(() => {
  chatStore.disconnectSocket();
});

function selectConversation(conversation: ConversationSummary) {
  const conversationId = conversation.task_id || conversation.application_id || conversation.item_id;
  if (conversationId) {
    chatStore.joinConversation(conversationId);
  }
}

function sendMessage(content: string) {
  if (chatStore.activeConversation) {
    chatStore.sendMessage(content, chatStore.activeConversation.other_user_id);
  }
}
</script>

<style scoped>
.message-center {
  height: calc(100vh - 60px); /* Adjust based on your navbar height */
  display: flex;
  flex-direction: column;
}

.message-center-layout {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* Conversations Sidebar */
.conversations-sidebar {
  width: 320px;
  background: #ffffff;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 1.5rem;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sidebar-header h2 {
  font-size: 1.25rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.total-unread {
  background: #4F46E5;
  color: white;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 500;
}

.conversations-list {
  flex: 1;
  overflow-y: auto;
}

/* Message Thread Container */
.message-thread-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #f9fafb;
}

.no-conversation {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
}

.no-conversation i {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.no-conversation p {
  font-size: 1.125rem;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .conversations-sidebar {
    width: 100%;
    position: absolute;
    z-index: 10;
  }
  
  .message-thread-container {
    display: none;
  }
  
  .conversations-sidebar.hidden {
    display: none;
  }
  
  .conversations-sidebar.hidden + .message-thread-container {
    display: flex;
  }
}
</style>