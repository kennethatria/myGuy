<template>
  <div class="container py-4 composer-page">
    <h1 class="mb-2">Post a Request</h1>
    <p class="text-muted mb-4">
      Say what you're looking for in a few words. Sellers who have it can list it for you, and you get a
      message in Messages. Your note stays up for 24 hours.
    </p>

    <form @submit.prevent="handleSubmit" novalidate>
      <StickyNote :seed="colorSeed" size="large" flat>
        <template #header>
          <label class="note-label" for="note-headline">Headline</label>
          <input
            id="note-headline"
            v-model="title"
            class="note-input note-input-headline"
            type="text"
            placeholder="Printer wanted"
            autocomplete="off"
            :aria-invalid="!titleOk"
            aria-describedby="headline-count"
          />
          <span id="headline-count" :class="['note-count', { over: !titleOk }]" aria-live="polite">
            {{ titleWords }}/{{ HEADLINE_MAX_WORDS }} words
          </span>

          <label class="note-label" for="note-body">Note</label>
          <textarea
            id="note-body"
            v-model="description"
            class="note-input note-input-body"
            rows="4"
            placeholder="Any working laser printer for a small office. Budget around UGX 300,000."
            :aria-invalid="!descriptionOk"
            aria-describedby="body-count"
          />
          <span id="body-count" :class="['note-count', { over: !descriptionOk }]" aria-live="polite">
            {{ descriptionWords }}/{{ BODY_MAX_WORDS }} words
          </span>
        </template>
      </StickyNote>

      <LocationField :state="location.state.value" @request="location.request" @clear="location.clear" />

      <p class="composer-hint">
        Leave out phone numbers, emails and links. You can share them in chat once you've booked an item.
      </p>

      <div v-if="formError" class="composer-error" role="alert">{{ formError }}</div>

      <div class="composer-actions">
        <button type="button" @click="router.back()" class="btn btn-outline">Cancel</button>
        <button type="submit" class="btn btn-primary" :disabled="!canSubmit">
          {{ isSubmitting ? 'Posting...' : 'Stick it on the board' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import LocationField from '@/components/LocationField.vue'
import { useRoughLocation } from '@/composables/useRoughLocation'
import {
  HEADLINE_MAX_WORDS,
  BODY_MAX_WORDS,
  countWords,
  headlineFits,
  bodyFits
} from '@/utils/gigNote'

const router = useRouter()
const authStore = useAuthStore()

const title = ref('')
const description = ref('')
const location = useRoughLocation()
const isSubmitting = ref(false)
const formError = ref('')
// Any colour will do for a new note; pick one per visit.
const colorSeed = Math.floor(Math.random() * 5)

const titleWords = computed(() => countWords(title.value))
const descriptionWords = computed(() => countWords(description.value))
const titleOk = computed(() => headlineFits(title.value))
const descriptionOk = computed(() => bodyFits(description.value))
const canSubmit = computed(() =>
  !isSubmitting.value &&
  titleWords.value > 0 &&
  descriptionWords.value > 0 &&
  titleOk.value &&
  descriptionOk.value
)

const handleSubmit = async () => {
  if (!canSubmit.value) return
  isSubmitting.value = true
  formError.value = ''

  try {
    const response = await fetch(`${config.STORE_API_URL}/requests`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${authStore.token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: title.value.trim(),
        description: description.value.trim(),
        ...(location.location.value ?? {})
      })
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      // store-service explains what to change (limits, contact details)
      throw new Error(data.error || 'Could not post your request. Please try again.')
    }
    router.push({ name: 'store-request', params: { id: data.id } })
  } catch (error) {
    formError.value = error instanceof Error ? error.message : 'Could not post your request. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.composer-page {
  max-width: 560px;
  margin: 0 auto;
}

.text-muted {
  color: var(--color-text-light, #6b7280);
}

.note-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--note-muted);
}

.note-input {
  width: 100%;
  border: none;
  border-bottom: 1px dashed rgba(31, 41, 55, 0.3);
  background: transparent;
  color: var(--note-ink);
  font-family: inherit;
  padding: 0.25rem 0;
  resize: none;
}

.note-input:focus {
  outline: none;
  border-bottom: 2px solid var(--note-ink);
}

.note-input::placeholder {
  color: rgba(31, 41, 55, 0.45);
}

.note-input-headline {
  min-height: 44px;
  font-size: 1.5rem;
  font-weight: 700;
}

.note-input-body {
  font-size: 1.1rem;
  line-height: 1.45;
}

.note-count {
  align-self: flex-end;
  font-size: 0.8rem;
  color: var(--note-muted);
  margin-bottom: 0.5rem;
}

.note-count.over {
  color: #b91c1c;
  font-weight: 700;
}

.composer-hint {
  margin-top: 1.25rem;
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.composer-error {
  margin-top: 0.75rem;
  padding: 0.75rem 1rem;
  color: #842029;
  background: #f8d7da;
  border: 1px solid #f5c2c7;
  border-radius: 0.375rem;
}

.composer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.25rem;
}

@media (max-width: 480px) {
  .composer-actions {
    flex-direction: column-reverse;
  }
}
</style>
