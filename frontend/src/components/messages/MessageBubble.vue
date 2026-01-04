<template>
  <div class="message-bubble" :class="{ 'own-message': isOwnMessage }">
    <div class="message-content">
      <div class="message-header">
        <span class="sender-name">{{ senderName }}</span>
        <span class="message-time">{{ formatTime(message.created_at) }}</span>
      </div>
      
      <div v-if="!isEditing" class="message-text">
        {{ message.content }}
        <span v-if="message.is_edited" class="edited-indicator">(edited)</span>
      </div>
      
      <div v-else class="edit-form">
        <input
          v-model="editText"
          type="text"
          class="edit-input"
          @keyup.enter="saveEdit"
          @keyup.esc="cancelEdit"
          maxlength="1000"
        />
        <div class="edit-actions">
          <button @click="saveEdit" class="save-btn">Save</button>
          <button @click="cancelEdit" class="cancel-btn">Cancel</button>
        </div>
      </div>
      
      <div v-if="message.has_removed_content" class="content-warning">
        <i class="fas fa-info-circle"></i>
        Links and contact information were removed from this message
      </div>
      
      <div class="message-footer">
        <span v-if="message.is_read && isOwnMessage" class="read-receipt">
          <i class="fas fa-check-double"></i>
          Read {{ formatTime(message.read_at!) }}
        </span>
        
        <div v-if="isOwnMessage && !message.is_deleted" class="message-actions">
          <button @click="startEdit" class="action-btn" title="Edit">
            <i class="fas fa-edit"></i>
          </button>
          <button @click="deleteMessage" class="action-btn" title="Delete">
            <i class="fas fa-trash"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useUserStore } from '@/stores/user';
import type { Message } from '@/stores/messages';

const props = defineProps<{
  message: Message;
  isOwnMessage: boolean;
}>();

const emit = defineEmits<{
  edit: [content: string];
  delete: [];
}>();

const userStore = useUserStore();
const isEditing = ref(false);
const editText = ref('');

// Compute sender name from enriched message data or user store
const senderName = computed(() => {
  // First try the enriched sender object on the message
  if (props.message.sender?.username) {
    return props.message.sender.username;
  }

  // Fallback to user store lookup
  if (props.message.sender_id) {
    const user = userStore.getUserById(props.message.sender_id);
    if (user) {
      return user.username;
    }
  }

  return 'Unknown User';
});

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  // Today
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  
  // Yesterday
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return `Yesterday ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  }
  
  // Within this week
  if (diff < 604800000) {
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    return `${days[date.getDay()]} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  }
  
  // Older
  return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
}

function startEdit() {
  if (props.message.is_deleted) return;
  isEditing.value = true;
  editText.value = props.message.content;
}

function saveEdit() {
  if (editText.value.trim() && editText.value !== props.message.content) {
    emit('edit', editText.value);
  }
  cancelEdit();
}

function cancelEdit() {
  isEditing.value = false;
  editText.value = '';
}

function deleteMessage() {
  if (confirm('Are you sure you want to delete this message?')) {
    emit('delete');
  }
}
</script>

<style scoped>
.message-bubble {
  display: flex;
  margin-bottom: 0.5rem;
}

.message-bubble.own-message {
  justify-content: flex-end;
}

.message-content {
  max-width: 70%;
  padding: 0.75rem 1rem;
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.own-message .message-content {
  background: #ede9fe;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 0.25rem;
  font-size: 0.75rem;
}

.sender-name {
  font-weight: 600;
  color: #4F46E5;
}

.own-message .sender-name {
  color: #4338ca;
}

.message-time {
  color: #6b7280;
  margin-left: 0.5rem;
}

.message-text {
  font-size: 0.875rem;
  color: #111827;
  word-wrap: break-word;
}

.edited-indicator {
  font-size: 0.75rem;
  color: #6b7280;
  font-style: italic;
  margin-left: 0.25rem;
}

/* Edit Form */
.edit-form {
  margin-top: 0.5rem;
}

.edit-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.25rem;
  font-size: 0.875rem;
}

.edit-input:focus {
  outline: none;
  border-color: #4F46E5;
}

.edit-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.save-btn, .cancel-btn {
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
  border: none;
  border-radius: 0.25rem;
  cursor: pointer;
  transition: background-color 0.15s;
}

.save-btn {
  background: #4F46E5;
  color: white;
}

.save-btn:hover {
  background: #4338ca;
}

.cancel-btn {
  background: #e5e7eb;
  color: #6b7280;
}

.cancel-btn:hover {
  background: #d1d5db;
}

/* Content Warning */
.content-warning {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: #fef3c7;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  color: #92400e;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Message Footer */
.message-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 0.5rem;
}

.read-receipt {
  font-size: 0.75rem;
  color: #10b981;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.message-actions {
  display: flex;
  gap: 0.5rem;
  opacity: 0;
  transition: opacity 0.15s;
}

.message-bubble:hover .message-actions {
  opacity: 1;
}

.action-btn {
  padding: 0.25rem 0.5rem;
  background: transparent;
  border: none;
  color: #6b7280;
  cursor: pointer;
  font-size: 0.75rem;
  border-radius: 0.25rem;
  transition: all 0.15s;
}

.action-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

/* Deleted Message */
.message-text[data-deleted="true"] {
  color: #6b7280;
  font-style: italic;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .message-content {
    max-width: 85%;
  }
  
  .message-actions {
    opacity: 1;
  }
}
</style>