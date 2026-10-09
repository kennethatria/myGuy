<template>
  <PostForm
    v-model:title="title"
    v-model:description="description"
    tone="sell"
    headline-placeholder="Kids bike, barely used"
    note-placeholder="Red, fits ages 5-8. UGX 150,000, collect in Ntinda."
    hint="Live for 24 hours. Put the price in the note or agree it in chat. Keep phone numbers and links for the chat."
    :error="formError"
    :busy="isSubmitting"
    @submit="handleSubmit"
    @cancel="router.back()"
  >
    <template #before>
      <!-- Answering a request: switching kind would drop that -->
      <div v-if="answering" class="answering" role="status">
        <span class="answering-label">Listing for a request</span>
        <strong>{{ answering.title }}</strong>
        <span class="answering-hint">@{{ answering.requester?.username || 'someone' }} gets a message when you post.</span>
      </div>
      <PostTypeSwitch v-else current="item" />
    </template>

    <template #extras>
      <fieldset class="photo-picker">
        <legend class="photo-legend">
          <span>Photos (optional)</span>
          <span class="photo-count">{{ photos.length }} / {{ MAX_PHOTOS }}</span>
        </legend>
        <ul class="photo-list">
          <li v-for="(photo, index) in photos" :key="photo.preview" class="photo-thumb">
            <img :src="photo.preview" :alt="`Photo ${index + 1}`" />
            <button type="button" class="photo-remove" :aria-label="`Remove photo ${index + 1}`" @click="removePhoto(index)">✕</button>
          </li>
          <li v-if="photos.length < MAX_PHOTOS">
            <label class="photo-add">
              <input type="file" accept="image/jpeg,image/png,image/gif" multiple class="visually-hidden" @change="addPhotos" />
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
              <span class="visually-hidden">Add photo</span>
            </label>
          </li>
        </ul>
      </fieldset>

      <LocationField :state="location.state.value" @request="location.request" @clear="location.clear" />
    </template>
  </PostForm>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import PostForm from '@/components/PostForm.vue'
import PostTypeSwitch from '@/components/PostTypeSwitch.vue'
import LocationField from '@/components/LocationField.vue'
import { useRoughLocation } from '@/composables/useRoughLocation'

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
const location = useRoughLocation()
const isSubmitting = ref(false)
const formError = ref('')

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
  isSubmitting.value = true
  formError.value = ''

  const fields = {
    title: title.value.trim(),
    description: description.value.trim(),
    ...(location.location.value ?? {})
  }
  const headers: Record<string, string> = { Authorization: `Bearer ${authStore.token}` }
  let body: BodyInit
  if (photos.value.length > 0) {
    const form = new FormData()
    form.append('title', fields.title)
    form.append('description', fields.description)
    if (location.location.value) {
      form.append('lat', String(location.location.value.lat))
      form.append('lng', String(location.location.value.lng))
    }
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
    router.push({ name: 'posted', params: { kind: 'item', id: data.id } })
  } catch (error) {
    formError.value = error instanceof Error ? error.message : 'Could not post your listing. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.answering {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-left: 4px solid var(--accent);
  border-radius: 0 12px 12px 0;
  background: var(--surface);
}

.answering-label {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.answering-hint {
  font-size: 13px;
  color: var(--text-muted);
}

.photo-picker {
  margin: 0;
  padding: 0;
  border: 0;
}

.photo-legend {
  width: 100%;
  display: flex;
  align-items: baseline;
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
}

.photo-legend span:first-child {
  flex: 1;
}

.photo-count {
  font-size: 12px;
  font-weight: 400;
  color: #9CA3AF;
}

.photo-list {
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin: 0;
  padding: 0;
}

.photo-thumb {
  position: relative;
}

.photo-thumb img,
.photo-add {
  width: 72px;
  height: 72px;
  border-radius: 10px;
}

.photo-thumb img {
  display: block;
  object-fit: cover;
}

.photo-remove {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 2px solid var(--bg);
  border-radius: 11px;
  background: var(--text);
  color: #fff;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}

/* A bigger tap area than the 22px dot */
.photo-remove::before {
  content: '';
  position: absolute;
  inset: -11px;
}

.photo-add {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1.5px dashed #B8BAC4;
  color: var(--text-muted);
  cursor: pointer;
}

.photo-add:focus-within {
  outline: 3px solid var(--accent-text);
  outline-offset: 2px;
}
</style>
