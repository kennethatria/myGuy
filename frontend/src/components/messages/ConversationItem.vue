<template>
  <div 
    class="conversation-item"
    :class="{ 'active': active }"
    @click="$emit('click')"
  >
    <div class="conversation-header">
      <h3 class="task-title" :class="{ 'unread': conversation.unread_count > 0 }">{{ conversation.task_title || conversation.application_title || 'Application' }}</h3>
      <span class="timestamp">{{ formatTime(conversation.last_message_time) }}</span>
    </div>
    
    <div class="conversation-body">
      <p class="other-user">{{ conversation.other_user_name }}</p>
      <p class="last-message" :class="{ 'unread': conversation.unread_count > 0 }">{{ conversation.last_message }}</p>
    </div>
    
    <div v-if="conversation.unread_count > 0" class="unread-badge">
      {{ conversation.unread_count }}
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConversationSummary } from '@/stores/messages';

defineProps<{
  conversation: ConversationSummary;
  active: boolean;
}>();

defineEmits<{
  click: [];
}>();

function formatTime(timestamp: string): string {
  if (!timestamp) {
    return 'No messages';
  }
  
  const date = new Date(timestamp);
  
  // Check if date is valid
  if (isNaN(date.getTime())) {
    console.warn('Invalid timestamp:', timestamp);
    return 'Invalid date';
  }
  
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  // Less than 1 minute
  if (diff < 60000) {
    return 'just now';
  }
  
  // Less than 1 hour
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000);
    return `${minutes}m ago`;
  }
  
  // Less than 24 hours
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000);
    return `${hours}h ago`;
  }
  
  // Less than 7 days
  if (diff < 604800000) {
    const days = Math.floor(diff / 86400000);
    return `${days}d ago`;
  }
  
  // Format as date
  return date.toLocaleDateString();
}
</script>

<style scoped>
.conversation-item {
  position: relative;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #e5e7eb;
  cursor: pointer;
  transition: background-color 0.15s;
}

.conversation-item:hover {
  background-color: #f9fafb;
}

.conversation-item.active {
  background-color: #ede9fe;
}

.conversation-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.5rem;
}

.task-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 0.5rem;
}

.task-title.unread {
  font-weight: 700;
  color: #4F46E5;
}

.timestamp {
  font-size: 0.75rem;
  color: #6b7280;
  white-space: nowrap;
}

.conversation-body {
  font-size: 0.875rem;
}

.other-user {
  color: #4F46E5;
  margin: 0 0 0.25rem 0;
  font-weight: 500;
}

.last-message {
  color: #6b7280;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.last-message.unread {
  font-weight: 600;
  color: #111827;
}

.unread-badge {
  position: absolute;
  top: 1rem;
  right: 1rem;
  background: #4F46E5;
  color: white;
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  min-width: 1.25rem;
  text-align: center;
}
</style>