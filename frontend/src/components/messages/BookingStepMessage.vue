<template>
  <!-- A booking step at the bottom of the conversation, like a gig event:
       what happened, then the next step for whoever acts. Only the newest
       step of a booking that is still at this stage offers one. -->
  <div class="system-message">
    <p class="event-text">{{ message.content }}</p>
    <span class="system-message-time">{{ time }}</span>

    <template v-if="current">
      <!-- Approved: the seller hands it over and marks it picked up, or
           releases it if the sale went nowhere (asks once more first) -->
      <template v-if="status === 'approved'">
        <template v-if="isSeller">
          <p v-if="confirmingRelease" class="event-note">Put it back on the board for others?</p>
          <div class="event-actions">
            <template v-if="!confirmingRelease">
              <button type="button" class="btn btn-primary btn-sm btn-picked-up" :disabled="busy" @click="act('confirm-delivery')">Picked up</button>
              <button type="button" class="btn btn-outline btn-sm btn-release" :disabled="busy" @click="confirmingRelease = true">Release reservation</button>
            </template>
            <template v-else>
              <button type="button" class="btn btn-danger btn-sm btn-decline" :disabled="busy" @click="release">Yes, release</button>
              <button type="button" class="btn btn-outline btn-sm btn-keep" :disabled="busy" @click="confirmingRelease = false">Keep it</button>
            </template>
          </div>
        </template>
        <p v-else class="event-note">Waiting for the seller to mark it picked up.</p>
      </template>

      <!-- Picked up: the buyer confirms they have it -->
      <template v-else-if="status === 'picked_up'">
        <div v-if="!isSeller" class="event-actions">
          <button type="button" class="btn btn-primary btn-sm btn-confirm-received" :disabled="busy" @click="act('confirm-received')">Confirm received</button>
        </div>
        <p v-else class="event-note">Waiting for the buyer to confirm they have it.</p>
      </template>

      <!-- Older bookings: the buyer confirmed first, the seller finishes -->
      <template v-else-if="status === 'item_received'">
        <div v-if="isSeller" class="event-actions">
          <button type="button" class="btn btn-primary btn-sm btn-confirm-delivery" :disabled="busy" @click="act('confirm-delivery')">Confirm handover</button>
        </div>
        <p v-else class="event-note">Waiting for the seller to confirm the handover.</p>
      </template>

      <!-- Completed: both review each other, comment optional -->
      <template v-else-if="status === 'completed'">
        <form v-if="!hasRated" class="event-review" @submit.prevent="submitRating">
          <div class="stars" role="radiogroup" :aria-label="isSeller ? 'Rate the buyer' : 'Rate the seller'">
            <button
              v-for="star in 5"
              :key="star"
              type="button"
              role="radio"
              :aria-checked="rating === star"
              :aria-label="`${star} star${star === 1 ? '' : 's'}`"
              :class="['star', { on: star <= rating }]"
              @click="rating = star"
            >★</button>
          </div>
          <textarea v-model="comment" rows="2" maxlength="500" placeholder="Add a comment (optional)" class="event-comment"></textarea>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="busy || !rating">Leave review</button>
        </form>
        <p v-else class="event-note done">
          You gave ★ {{ givenRating }}<template v-if="givenReview">: “{{ givenReview }}”</template>
        </p>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import type { Message, BookingAction } from '@/stores/messages'

const props = defineProps<{
  message: Message
  // The booking's request: who is buyer and seller, its current status and ratings
  request?: Message
  currentUserId?: number
  // The newest step of this booking in the conversation
  latest: boolean
}>()

const emit = defineEmits<{
  bookingAction: [bookingId: number, action: BookingAction, rating?: number, review?: string]
}>()

const status = computed(() => props.message.metadata?.status)
const bookingId = computed(() => props.message.metadata?.booking_id)
// The buyer sent the request; the seller received it
const isSeller = computed(() => props.request?.recipient_id === props.currentUserId)
// Still at this stage: the request's status hasn't moved on
const current = computed(() => props.latest && !!props.request && props.request.metadata?.status === status.value)

const givenRating = computed(() =>
  (isSeller.value ? props.request?.metadata?.seller_rating : props.request?.metadata?.buyer_rating) ?? null)
const givenReview = computed(() =>
  (isSeller.value ? props.request?.metadata?.seller_review : props.request?.metadata?.buyer_review) || '')
const hasRated = computed(() => givenRating.value !== null && givenRating.value !== undefined)

const time = computed(() => {
  const d = new Date(props.message.created_at)
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
})

const busy = ref(false)
const confirmingRelease = ref(false)
const rating = ref(0)
const comment = ref('')
let timeout: number | undefined

function done() {
  busy.value = false
  if (timeout) window.clearTimeout(timeout)
}

// Runs a step; its result arrives as the next step in this conversation
function act(action: BookingAction, stars?: number, review?: string) {
  if (!bookingId.value) return
  busy.value = true
  // Fallback if no update arrives
  timeout = window.setTimeout(done, 10000)
  emit('bookingAction', bookingId.value, action, stars, review)
}

function release() {
  confirmingRelease.value = false
  act('release')
}

function submitRating() {
  if (!rating.value) return
  act(isSeller.value ? 'rate-buyer' : 'rate-seller', rating.value, comment.value.trim())
}

// The step is done once the booking's status or ratings change
watch(() => props.request?.metadata, done, { deep: true })
onUnmounted(done)
</script>

<style scoped src="./eventCard.css"></style>

<style scoped>
.event-note.done {
  color: #166534;
}
</style>
