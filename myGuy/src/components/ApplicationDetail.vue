<template>
  <div class="application-detail">
    <div class="application-header">
      <div class="applicant-info">
        <h3>
          Application from 
          <router-link 
            :to="{ name: 'user-profile', params: { id: application.applicant.id } }"
            class="text-primary"
          >
            {{ application.applicant.username }}
          </router-link>
        </h3>
        <div class="application-meta">
          <span class="proposed-fee">${{ application.proposedFee }}</span>
          <span class="status-badge" :class="`status-${application.status}`">
            {{ application.status }}
          </span>
          <span class="date">{{ formatDate(application.createdAt) }}</span>
        </div>
      </div>
      
      <div v-if="isTaskOwner && application.status === 'pending'" class="application-actions">
        <button @click="$emit('accept', application.id)" class="btn btn-success btn-sm">
          Accept
        </button>
        <button @click="$emit('decline', application.id)" class="btn btn-danger btn-sm">
          Decline
        </button>
      </div>
    </div>

    <div v-if="application.message" class="application-message">
      <h4>Application Message</h4>
      <p>{{ application.message }}</p>
    </div>

    <!-- Messages Section -->
    <div class="messages-section">
      <h4>Messages</h4>
      
      <div v-if="loadingMessages" class="text-center py-3">
        <div class="spinner-border spinner-border-sm" role="status">
          <span class="visually-hidden">Loading messages...</span>
        </div>
      </div>

      <div v-else class="messages-container">
        <div v-if="messages.length === 0" class="no-messages">
          <p class="text-muted">No messages yet. Start a conversation!</p>
        </div>
        
        <div v-else class="messages-list">
          <div 
            v-for="message in messages" 
            :key="message.id"
            class="message-item"
            :class="{ 'own-message': message.senderId === currentUserId }"
          >
            <div class="message-header">
              <strong>{{ message.sender?.username || 'Unknown' }}</strong>
              <span class="message-time">{{ formatTime(message.createdAt) }}</span>
            </div>
            <div class="message-content">
              {{ message.content }}
            </div>
          </div>
        </div>
      </div>

      <!-- Message Input -->
      <div v-if="canSendMessage" class="message-input-container">
        <form @submit.prevent="sendMessage" class="message-form">
          <div class="input-group">
            <input
              v-model="newMessage"
              type="text"
              class="form-control"
              placeholder="Type your message..."
              :disabled="sendingMessage"
            />
            <button 
              type="submit" 
              class="btn btn-primary"
              :disabled="!newMessage.trim() || sendingMessage"
            >
              <span v-if="sendingMessage">Sending...</span>
              <span v-else>Send</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { format, formatDistanceToNow } from 'date-fns'
import { useAuthStore } from '@/stores/auth'
import { useMessagesStore } from '@/stores/messages'

interface Application {
  id: number
  taskId: number
  applicantId: number
  proposedFee: number
  status: string
  message?: string
  createdAt: string
  applicant: {
    id: number
    username: string
    fullName?: string
  }
}

interface Message {
  id: number
  senderId: number
  recipientId: number
  content: string
  createdAt: string
  isRead: boolean
  sender?: {
    id: number
    username: string
  }
  recipient?: {
    id: number
    username: string
  }
}

interface Props {
  application: Application
  taskOwnerId: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'accept': [applicationId: number]
  'decline': [applicationId: number]
  'message-sent': []
}>()

const authStore = useAuthStore()
const messagesStore = useMessagesStore()

const messages = ref<Message[]>([])
const newMessage = ref('')
const loadingMessages = ref(false)
const sendingMessage = ref(false)

const currentUserId = computed(() => authStore.user?.id)
const isTaskOwner = computed(() => currentUserId.value === props.taskOwnerId)
const isApplicant = computed(() => currentUserId.value === props.application.applicantId)

const canSendMessage = computed(() => {
  return (isTaskOwner.value || isApplicant.value) && props.application.status === 'pending'
})

const formatDate = (date: string) => {
  return format(new Date(date), 'MMM d, yyyy')
}

const formatTime = (date: string) => {
  const messageDate = new Date(date)
  const now = new Date()
  const diffInHours = (now.getTime() - messageDate.getTime()) / (1000 * 60 * 60)
  
  if (diffInHours < 24) {
    return formatDistanceToNow(messageDate, { addSuffix: true })
  }
  return format(messageDate, 'MMM d, h:mm a')
}

const loadMessages = async () => {
  loadingMessages.value = true
  try {
    messages.value = await messagesStore.fetchApplicationMessages(props.application.id)
  } catch (error) {
    console.error('Failed to load messages:', error)
  } finally {
    loadingMessages.value = false
  }
}

const sendMessage = async () => {
  if (!newMessage.value.trim() || sendingMessage.value) return
  
  sendingMessage.value = true
  try {
    await messagesStore.sendApplicationMessage(props.application.id, newMessage.value.trim())
    newMessage.value = ''
    await loadMessages() // Reload messages
    emit('message-sent')
  } catch (error) {
    console.error('Failed to send message:', error)
  } finally {
    sendingMessage.value = false
  }
}

onMounted(() => {
  loadMessages()
})

// Reload messages when application changes
watch(() => props.application.id, () => {
  loadMessages()
})
</script>

<style scoped>
.application-detail {
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1rem;
}

.application-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e0e0e0;
}

.applicant-info h3 {
  margin: 0 0 0.5rem 0;
  font-size: 1.25rem;
}

.application-meta {
  display: flex;
  gap: 1rem;
  align-items: center;
  font-size: 0.875rem;
}

.proposed-fee {
  font-weight: bold;
  color: #28a745;
  font-size: 1rem;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
}

.status-pending {
  background-color: #ffc107;
  color: #000;
}

.status-accepted {
  background-color: #28a745;
  color: white;
}

.status-declined {
  background-color: #dc3545;
  color: white;
}

.date {
  color: #6c757d;
}

.application-actions {
  display: flex;
  gap: 0.5rem;
}

.application-message {
  margin-bottom: 1.5rem;
}

.application-message h4 {
  font-size: 1rem;
  margin-bottom: 0.5rem;
  color: #495057;
}

.application-message p {
  margin: 0;
  color: #6c757d;
}

.messages-section {
  margin-top: 1.5rem;
}

.messages-section h4 {
  font-size: 1rem;
  margin-bottom: 1rem;
  color: #495057;
}

.messages-container {
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  min-height: 200px;
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 1rem;
}

.no-messages {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #6c757d;
}

.messages-list {
  padding: 1rem;
}

.message-item {
  margin-bottom: 1rem;
  padding: 0.75rem;
  background-color: #f8f9fa;
  border-radius: 6px;
}

.message-item.own-message {
  background-color: #e3f2fd;
  margin-left: 20%;
}

.message-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.25rem;
  font-size: 0.875rem;
}

.message-header strong {
  color: #495057;
}

.message-time {
  color: #6c757d;
  font-size: 0.75rem;
}

.message-content {
  color: #212529;
}

.message-input-container {
  margin-top: 1rem;
}

.message-form {
  width: 100%;
}

.input-group {
  display: flex;
  gap: 0.5rem;
}

.form-control {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font-size: 1rem;
}

.form-control:focus {
  outline: none;
  border-color: #80bdff;
  box-shadow: 0 0 0 0.2rem rgba(0, 123, 255, 0.25);
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  transition: background-color 0.15s ease-in-out;
}

.btn-primary {
  background-color: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: #0056b3;
}

.btn-success {
  background-color: #28a745;
  color: white;
}

.btn-success:hover {
  background-color: #218838;
}

.btn-danger {
  background-color: #dc3545;
  color: white;
}

.btn-danger:hover {
  background-color: #c82333;
}

.btn-sm {
  padding: 0.25rem 0.75rem;
  font-size: 0.875rem;
}

.btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.text-primary {
  color: #007bff;
  text-decoration: none;
}

.text-primary:hover {
  text-decoration: underline;
}

.text-muted {
  color: #6c757d;
}

.text-center {
  text-align: center;
}

.py-3 {
  padding-top: 1rem;
  padding-bottom: 1rem;
}

.spinner-border {
  display: inline-block;
  width: 2rem;
  height: 2rem;
  vertical-align: text-bottom;
  border: 0.25em solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spinner-border 0.75s linear infinite;
}

.spinner-border-sm {
  width: 1rem;
  height: 1rem;
  border-width: 0.2em;
}

@keyframes spinner-border {
  to { transform: rotate(360deg); }
}

.visually-hidden {
  position: absolute !important;
  width: 1px !important;
  height: 1px !important;
  padding: 0 !important;
  margin: -1px !important;
  overflow: hidden !important;
  clip: rect(0, 0, 0, 0) !important;
  white-space: nowrap !important;
  border: 0 !important;
}
</style>