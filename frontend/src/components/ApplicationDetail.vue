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
          <span class="proposed-fee">UGX {{ formatCurrency(application.proposed_fee) }}</span>
          <span class="status-badge" :class="`status-${application.status}`">
            {{ application.status }}
          </span>
          <span class="date">{{ formatDate(application.created_at || application.createdAt) }}</span>
        </div>
      </div>
      
      <div v-if="isTaskOwner" class="application-actions">
        <button @click="$emit('message', application.applicant_id)" class="btn btn-outline btn-sm">
          Message
        </button>
        <template v-if="application.status === 'pending'">
          <button @click="$emit('accept', application.id)" class="btn btn-success btn-sm">
            Accept
          </button>
          <button @click="$emit('decline', application.id)" class="btn btn-danger btn-sm">
            Decline
          </button>
        </template>
      </div>
    </div>

    <div v-if="application.message" class="application-message">
      <h4>Application Message</h4>
      <p>{{ application.message }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { format } from 'date-fns'
import { useAuthStore } from '@/stores/auth'

interface Application {
  id: number
  task_id: number
  applicant_id: number
  proposed_fee: number
  status: string
  message?: string
  created_at?: string
  createdAt?: string  // For backward compatibility
  applicant: {
    id: number
    username: string
    fullName?: string
  }
}

interface Props {
  application: Application
  taskOwnerId: number
}

const props = defineProps<Props>()

defineEmits<{
  'accept': [applicationId: number]
  'decline': [applicationId: number]
  // Open the poster's conversation with this applicant
  'message': [applicantId: number]
}>()

const authStore = useAuthStore()

const currentUserId = computed(() => authStore.user?.id)
const isTaskOwner = computed(() => currentUserId.value === props.taskOwnerId)

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('en-UG', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount)
}

const formatDate = (date: string | undefined) => {
  if (!date) return 'Unknown date'
  try {
    return format(new Date(date), 'MMM d, yyyy')
  } catch {
    console.error('Invalid date:', date)
    return 'Invalid date'
  }
}

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

@media (max-width: 640px) {
  .application-header {
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .application-actions {
    width: 100%;
  }

  .application-actions .btn {
    flex: 1;
  }
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

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  transition: background-color 0.15s ease-in-out;
}

.btn-primary {
  background-color: var(--color-primary);
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
  color: var(--color-primary);
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