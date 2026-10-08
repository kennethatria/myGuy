<template>
  <div class="message-bubble" :class="{ 'own-message': isOwnMessage }">
    <div class="message-content">
      <!-- One person on each side: the name is for screen readers -->
      <span class="visually-hidden">{{ senderName }}:</span>
      
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
        <span class="message-time">{{ formatTime(message.created_at) }}</span>
        <span v-if="message.is_read && isOwnMessage" class="read-receipt">
          <i class="fas fa-check-double"></i>
          Read<template v-if="message.read_at"> {{ formatTime(message.read_at) }}</template>
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
  if (props.isOwnMessage) return 'You';
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
  margin-bottom: 14px;
}

.message-bubble.own-message {
  justify-content: flex-end;
}

/* Theirs: white on the page; yours: blue */
.message-content {
  max-width: 78%;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 16px 16px 16px 4px;
  background: var(--surface);
  color: var(--text);
}

.own-message .message-content {
  border-color: var(--accent);
  border-radius: 16px 16px 4px 16px;
  background: var(--accent);
  color: var(--on-accent);
}

.message-time {
  font-size: 11px;
  opacity: 0.7;
}

.message-text {
  font-size: 15px;
  line-height: 1.35;
  word-wrap: break-word;
}

.edited-indicator {
  font-size: 0.75rem;
  opacity: 0.7;
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
  border-color: var(--color-primary);
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
  background: var(--color-primary);
  color: white;
}

.save-btn:hover {
  background: var(--color-primary-dark);
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
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.read-receipt {
  font-size: 11px;
  opacity: 0.7;
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
  color: inherit;
  opacity: 0.75;
  cursor: pointer;
  font-size: 0.75rem;
  border-radius: 0.25rem;
  transition: all 0.15s;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.35);
  opacity: 1;
}

/* Deleted Message */
.message-text[data-deleted="true"] {
  opacity: 0.7;
  font-style: italic;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  /* Finger-sized edit/delete buttons */
  .action-btn {
    min-width: 32px;
    min-height: 32px;
  }

  .message-content {
    max-width: 85%;
  }
  
  .message-actions {
    opacity: 1;
  }
}
</style>