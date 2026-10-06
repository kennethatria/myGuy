<template>
  <div class="system-message" :class="{ 'has-actions': showActions }">
    <p class="event-text">{{ message.content }}</p>
    <span class="system-message-time">{{ time }}</span>

    <!-- The step this event asks of the person viewing it. Only the newest
         event in a conversation offers one; older ones are history. -->
    <div v-if="showActions" class="event-actions">
      <template v-if="action === 'answer'">
        <button type="button" class="btn btn-primary btn-sm" :disabled="busy" @click="run(() => tasksStore.respondToApplication(taskId, applicationId!, 'accepted'))">Accept</button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="run(() => tasksStore.respondToApplication(taskId, applicationId!, 'declined'))">Decline</button>
      </template>

      <button v-else-if="action === 'mark-done'" type="button" class="btn btn-primary btn-sm" :disabled="busy" @click="run(() => tasksStore.updateTaskStatus(taskId, 'pending_approval'))">
        Mark as done
      </button>

      <template v-else-if="action === 'approve'">
        <button type="button" class="btn btn-primary btn-sm" :disabled="busy" @click="run(() => tasksStore.updateTaskStatus(taskId, 'completed'))">Approve</button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="run(() => tasksStore.updateTaskStatus(taskId, 'in_progress'))">Not yet</button>
      </template>

      <form v-else-if="action === 'review'" class="event-review" @submit.prevent="submitReview">
        <p v-if="reviewed" class="event-note">Thanks, your review is in. You'll find it on the Reviews page.</p>
        <template v-else>
          <div class="stars" role="radiogroup" aria-label="Your rating">
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
        </template>
      </form>
    </div>

    <p v-if="error" class="event-error" role="alert">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import { useReviewsStore } from '@/stores/reviews'
import type { Message } from '@/stores/messages'

const props = defineProps<{
  message: Message
  currentUserId?: number
  // The newest gig event in this conversation: only it offers a step
  latest: boolean
}>()

const tasksStore = useTasksStore()
const reviewsStore = useReviewsStore()

const taskId = computed(() => props.message.task_id!)
const applicationId = computed(() => props.message.metadata?.application_id)
const isRecipient = computed(() => props.message.recipient_id === props.currentUserId)

// What this event asks of the viewer. Events go from the person who acted to
// the person who acts next, so the step is the recipient's; a completed gig
// asks both for a review.
const action = computed<'answer' | 'mark-done' | 'approve' | 'review' | null>(() => {
  switch (props.message.metadata?.event) {
    case 'application': return isRecipient.value && applicationId.value ? 'answer' : null
    case 'accepted':
    case 'not_done': return isRecipient.value ? 'mark-done' : null
    case 'done': return isRecipient.value ? 'approve' : null
    case 'completed': return 'review'
    default: return null
  }
})

const showActions = computed(() => props.latest && !!action.value && !!props.message.task_id)

const time = computed(() => {
  const d = new Date(props.message.created_at)
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
})

const busy = ref(false)
const error = ref('')

// Runs a step. Its result arrives as the next event in this conversation,
// which then becomes the one offering the next step.
async function run(step: () => Promise<unknown>) {
  busy.value = true
  error.value = ''
  try {
    await step()
  } catch (err) {
    error.value = err instanceof Error && err.message ? err.message : 'That didn\'t work. Please try again.'
  } finally {
    busy.value = false
  }
}

const rating = ref(0)
const comment = ref('')
const reviewed = ref(false)

async function submitReview() {
  if (!rating.value) return
  await run(async () => {
    await reviewsStore.createReview(taskId.value, { rating: rating.value, comment: comment.value.trim() })
    reviewed.value = true
  })
}

onMounted(async () => {
  if (showActions.value && action.value === 'review') {
    try {
      reviewed.value = await reviewsStore.hasReviewedTask(taskId.value)
    } catch {
      // Unknown: offer the form; the backend refuses a second review
    }
  }
})
</script>

<style scoped>
.system-message {
  align-self: center;
  max-width: min(90%, 28rem);
  margin: 0.5rem auto;
  padding: 0.5rem 0.75rem;
  border: 1px solid #c7d2fe;
  border-radius: 0.5rem;
  background: #eef2ff;
  color: #3730a3;
  font-size: 0.875rem;
  text-align: center;
}

.event-text {
  margin: 0;
  white-space: pre-line;
  overflow-wrap: anywhere;
}

.system-message-time {
  display: block;
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: #6366f1;
}

.event-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 0.625rem;
}

/* Thumb-sized on phones */
.event-actions .btn {
  min-height: 44px;
  min-width: 6.5rem;
}

.event-review {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
}

.stars {
  display: flex;
  gap: 0.25rem;
}

.star {
  min-width: 44px;
  min-height: 44px;
  border: none;
  background: none;
  font-size: 1.6rem;
  line-height: 1;
  color: #c7d2fe;
  cursor: pointer;
}

.star.on {
  color: #f59e0b;
}

.event-comment {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #c7d2fe;
  border-radius: 0.375rem;
  font: inherit;
  resize: vertical;
}

.event-note {
  margin: 0;
  color: #166534;
}

.event-error {
  margin: 0.5rem 0 0;
  color: #b91c1c;
}
</style>
