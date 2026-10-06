<template>
  <!-- A seller listed something for the viewer's request: book it right here,
       as an applicant's message offers Accept on a gig. -->
  <div class="system-message">
    <p class="event-text">{{ message.content }}</p>
    <span class="system-message-time">{{ time }}</span>

    <template v-if="isRequester">
      <div v-if="state === 'open'" class="event-actions">
        <button type="button" class="btn btn-primary btn-sm" :disabled="busy" @click="book">
          {{ busy ? 'Booking...' : 'Book it' }}
        </button>
        <router-link :to="{ name: 'store-item', params: { id: itemId } }" class="btn btn-outline btn-sm">View item</router-link>
      </div>
      <p v-else-if="state === 'booked'" class="event-note">You booked it. The seller answers below.</p>
      <p v-else-if="state === 'gone'" class="event-note">No longer available.</p>
    </template>
    <p v-else class="event-note">Waiting for them to book it.</p>

    <p v-if="error" class="event-error" role="alert">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import config from '@/config'
import { useAuthStore } from '@/stores/auth'
import type { Message } from '@/stores/messages'

const props = defineProps<{
  message: Message
  currentUserId?: number
}>()

const authStore = useAuthStore()

const itemId = computed(() => props.message.store_item_id ?? props.message.item_id ?? 0)
// The seller tells the requester: the step is the requester's
const isRequester = computed(() => props.message.recipient_id === props.currentUserId)

// open: bookable · booked: the requester already asked · gone: booked by
// someone else, removed or sold
const state = ref<'checking' | 'open' | 'booked' | 'gone'>('checking')
const busy = ref(false)
const error = ref('')

const time = computed(() => {
  const d = new Date(props.message.created_at)
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
})

const headers = () => ({ Authorization: `Bearer ${authStore.token}`, 'Content-Type': 'application/json' })

async function check() {
  try {
    const [item, mine] = await Promise.all([
      fetch(`${config.STORE_API_URL}/items/${itemId.value}`, { headers: headers() }),
      fetch(`${config.STORE_API_URL}/items/${itemId.value}/booking-request`, { headers: headers() })
    ])
    const booking = mine.ok ? (await mine.json()).booking_request : null
    if (booking) {
      state.value = 'booked'
      return
    }
    const status = item.ok ? (await item.json()).status : null
    state.value = status === 'active' ? 'open' : 'gone'
  } catch {
    // Unknown: offer to book; store-service refuses if it can't be
    state.value = 'open'
  }
}

// Books it like the item page does; the booking then shows in this
// conversation for the seller to answer
async function book() {
  busy.value = true
  error.value = ''
  try {
    const response = await fetch(`${config.STORE_API_URL}/items/${itemId.value}/booking-request`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({})
    })
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || 'Could not book it. Please try again.')
    }
    state.value = 'booked'
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not book it. Please try again.'
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  if (isRequester.value && itemId.value) check()
})
</script>

<style scoped src="./eventCard.css"></style>
