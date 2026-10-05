<template>
  <div class="container py-4 composer-page">
    <h1 class="mb-2">Post a Listing</h1>
    <p class="text-muted mb-4">
      Say what you're selling in a few words, and add the price if you like. Your note stays on the
      marketplace for 24 hours; agree the details in chat with whoever asks to book it.
    </p>

    <div v-if="answering" class="answering" role="status">
      <span class="answering-label">Listing for a request</span>
      <strong>{{ answering.title }}</strong>
      <span class="text-muted">@{{ answering.requester?.username || 'someone' }} gets a message when you post.</span>
    </div>

    <form @submit.prevent="handleSubmit" novalidate>
      <StickyNote :seed="colorSeed" size="large" flat :photo="photos[0]?.preview" photo-alt="Your first photo">
        <template #header>
          <label class="note-label" for="note-headline">Headline</label>
          <input
            id="note-headline"
            v-model="title"
            class="note-input note-input-headline"
            type="text"
            placeholder="Kids bike, barely used"
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
            placeholder="Red, fits ages 5-8. UGX 150,000, collect in Ntinda."
            :aria-invalid="!descriptionOk"
            aria-describedby="body-count"
          />
          <span id="body-count" :class="['note-count', { over: !descriptionOk }]" aria-live="polite">
            {{ descriptionWords }}/{{ BODY_MAX_WORDS }} words
          </span>
        </template>
      </StickyNote>

      <fieldset class="photo-picker">
        <legend class="photo-legend">Photos <span class="text-muted">(up to {{ MAX_PHOTOS }}; the first is pinned on the note)</span></legend>
        <ul class="photo-list">
          <li v-for="(photo, index) in photos" :key="photo.preview" class="photo-thumb">
            <img :src="photo.preview" :alt="`Photo ${index + 1}`" />
            <button type="button" class="photo-remove" :aria-label="`Remove photo ${index + 1}`" @click="removePhoto(index)">×</button>
          </li>
          <li v-if="photos.length < MAX_PHOTOS">
            <label class="photo-add">
              <input type="file" accept="image/jpeg,image/png,image/gif" multiple class="visually-hidden" @change="addPhotos" />
              <span aria-hidden="true">+</span>
              <span>Add photo</span>
            </label>
          </li>
        </ul>
      </fieldset>

      <p class="composer-hint">
        Leave out phone numbers, emails and links. You can share them in chat once you approve a booking.
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import StickyNote from '@/components/StickyNote.vue'
import {
  HEADLINE_MAX_WORDS,
  BODY_MAX_WORDS,
  countWords,
  headlineFits,
  bodyFits
} from '@/utils/gigNote'

// store-service keeps the first three JPG/PNG/GIF photos up to 5 MB each
const MAX_PHOTOS = 3
const MAX_PHOTO_BYTES = 5 * 1024 * 1024

interface Photo {
  file: File
  preview: string
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// Opened from a request's "I have this": the listing answers that request
const answering = ref<{ id: number; title: string; requester?: { username: string } } | null>(null)

onMounted(async () => {
  const requestId = Number(route.query.request)
  if (!Number.isInteger(requestId) || requestId <= 0) return
  try {
    const response = await fetch(`${config.STORE_API_URL}/requests/${requestId}`, {
      headers: { Authorization: `Bearer ${authStore.token}` }
    })
    if (response.ok) answering.value = await response.json()
  } catch {
    // Without it the listing is posted as a plain listing
  }
})

const title = ref('')
const description = ref('')
const photos = ref<Photo[]>([])
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

const addPhotos = (event: Event) => {
  const input = event.target as HTMLInputElement
  formError.value = ''
  for (const file of Array.from(input.files ?? [])) {
    if (photos.value.length >= MAX_PHOTOS) break
    if (file.size > MAX_PHOTO_BYTES) {
      formError.value = `${file.name} is over 5 MB, so it was left out.`
      continue
    }
    photos.value.push({ file, preview: URL.createObjectURL(file) })
  }
  input.value = ''
}

const removePhoto = (index: number) => {
  const [removed] = photos.value.splice(index, 1)
  if (removed) URL.revokeObjectURL(removed.preview)
}

onUnmounted(() => photos.value.forEach((photo) => URL.revokeObjectURL(photo.preview)))

const handleSubmit = async () => {
  if (!canSubmit.value) return
  isSubmitting.value = true
  formError.value = ''

  const fields = { title: title.value.trim(), description: description.value.trim() }
  const headers: Record<string, string> = { Authorization: `Bearer ${authStore.token}` }
  let body: BodyInit
  if (photos.value.length > 0) {
    const form = new FormData()
    form.append('title', fields.title)
    form.append('description', fields.description)
    if (answering.value) form.append('request_id', String(answering.value.id))
    photos.value.forEach((photo) => form.append('images', photo.file))
    body = form
  } else {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(answering.value ? { ...fields, request_id: answering.value.id } : fields)
  }

  try {
    const response = await fetch(`${config.STORE_API_URL}/items`, { method: 'POST', headers, body })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      // store-service explains what to change (limits, contact details)
      throw new Error(data.error || 'Could not post your listing. Please try again.')
    }
    router.push({ name: 'store-item', params: { id: data.id } })
  } catch (error) {
    formError.value = error instanceof Error ? error.message : 'Could not post your listing. Please try again.'
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

.photo-picker {
  margin: 1.5rem 0 0;
  padding: 0;
  border: none;
}

.photo-legend {
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.photo-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.photo-thumb {
  position: relative;
}

.photo-thumb img,
.photo-add {
  width: 88px;
  height: 88px;
  border-radius: 6px;
}

.photo-thumb img {
  display: block;
  object-fit: cover;
}

/* Small to look at, but a 44px target */
.photo-remove::before {
  content: '';
  position: absolute;
  inset: -0.5rem;
}

.photo-remove {
  position: absolute;
  top: -0.5rem;
  right: -0.5rem;
  width: 1.75rem;
  height: 1.75rem;
  border: none;
  border-radius: 50%;
  background: #1f2937;
  color: #fff;
  font-size: 1rem;
  line-height: 1;
  cursor: pointer;
}

.photo-add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.15rem;
  border: 2px dashed var(--color-border, #d1d5db);
  color: var(--color-text-light, #6b7280);
  font-size: 0.8rem;
  cursor: pointer;
}

.photo-add span[aria-hidden] {
  font-size: 1.5rem;
  line-height: 1;
}

.photo-add:focus-within {
  outline: 3px solid var(--color-primary, #4f46e5);
  outline-offset: 2px;
}

.answering {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  margin-bottom: 1.25rem;
  padding: 0.75rem 1rem;
  border-left: 4px solid var(--color-primary, #4f46e5);
  background: #fff;
  border-radius: 0 0.375rem 0.375rem 0;
}

.answering-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-light, #6b7280);
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
