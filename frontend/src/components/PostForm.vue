<template>
  <!-- The composer every post shares: a headline and a note on a sticky
       note in the post's colour, then whatever the kind adds (photos, area),
       then one button pinned to the bottom -->
  <form class="post-form" novalidate @submit.prevent="submit">
    <div class="post-body">
      <slot name="before" />

      <div :class="['note-form', `tone-${tone}`]">
        <div class="field">
          <label class="note-label" for="note-headline">Headline</label>
          <input
            id="note-headline"
            :value="title"
            class="note-input note-input-headline"
            type="text"
            :placeholder="headlinePlaceholder"
            autocomplete="off"
            :aria-invalid="!titleOk"
            aria-describedby="headline-count"
            @input="$emit('update:title', ($event.target as HTMLInputElement).value)"
          />
          <span id="headline-count" :class="['note-count', { over: !titleOk }]" aria-live="polite">
            {{ titleWords }} / {{ HEADLINE_MAX_WORDS }} words
          </span>
        </div>

        <div class="field">
          <label class="note-label" for="note-body">Note</label>
          <textarea
            id="note-body"
            :value="description"
            class="note-input note-input-body"
            rows="3"
            :placeholder="notePlaceholder"
            :aria-invalid="!descriptionOk"
            aria-describedby="body-count"
            @input="$emit('update:description', ($event.target as HTMLTextAreaElement).value)"
          />
          <span id="body-count" :class="['note-count', { over: !descriptionOk }]" aria-live="polite">
            {{ descriptionWords }} / {{ BODY_MAX_WORDS }} words
          </span>
        </div>
      </div>

      <slot name="extras" />

      <p class="post-hint">{{ hint }}</p>

      <div v-if="error" class="post-error" role="alert">{{ error }}</div>
    </div>

    <div class="post-actions">
      <button type="submit" class="btn btn-primary post-submit" :disabled="!canSubmit">
        {{ busy ? busyLabel : submitLabel }}
      </button>
      <button type="button" class="post-cancel" @click="$emit('cancel')">Cancel</button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { HEADLINE_MAX_WORDS, BODY_MAX_WORDS, countWords, headlineFits, bodyFits } from '@/utils/gigNote'

const props = withDefaults(defineProps<{
  tone: 'gig' | 'sell' | 'want'
  title: string
  description: string
  headlinePlaceholder?: string
  notePlaceholder?: string
  hint?: string
  error?: string
  busy?: boolean
  submitLabel?: string
  busyLabel?: string
}>(), {
  headlinePlaceholder: '',
  notePlaceholder: '',
  hint: 'Live for 24 hours. Keep phone numbers and links for the chat.',
  error: '',
  busy: false,
  submitLabel: 'Stick it on the board',
  busyLabel: 'Posting...'
})

const emit = defineEmits<{
  'update:title': [value: string]
  'update:description': [value: string]
  submit: []
  cancel: []
}>()

// The services check these again; here they only drive the counters
const titleWords = computed(() => countWords(props.title))
const descriptionWords = computed(() => countWords(props.description))
const titleOk = computed(() => headlineFits(props.title))
const descriptionOk = computed(() => bodyFits(props.description))
const canSubmit = computed(() =>
  !props.busy && titleWords.value > 0 && descriptionWords.value > 0 && titleOk.value && descriptionOk.value
)

const submit = () => {
  if (canSubmit.value) emit('submit')
}
</script>

<style scoped>
.post-form {
  max-width: 560px;
  min-height: 100%;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
}

.post-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px 20px;
}

.note-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-radius: 8px;
}

.tone-gig { background: #E8F0FF; --ink: #1E3A8A; }
.tone-sell { background: #FBF1CE; --ink: #713F12; }
.tone-want { background: #FCE8F2; --ink: #831843; }

.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.note-label {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--ink);
}

.note-input {
  width: 100%;
  padding: 0;
  border: 0;
  border-bottom: 1.5px dashed rgba(17, 24, 39, 0.3);
  background: transparent;
  color: var(--text);
  resize: none;
}

.note-input:focus {
  outline: none;
  border-bottom: 2px solid var(--ink);
}

.note-input::placeholder {
  color: rgba(17, 24, 39, 0.4);
}

.note-input-headline {
  min-height: 44px;
  font-size: 18px;
  font-weight: 600;
}

.note-input-body {
  font-size: 16px;
  line-height: 1.45;
}

.note-count {
  align-self: flex-end;
  font-size: 12px;
  color: var(--text-muted);
}

.note-count.over {
  color: #b91c1c;
  font-weight: 700;
}

.post-hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-muted);
}

.post-error {
  padding: 12px 14px;
  border: 1px solid #f5c2c7;
  border-radius: 12px;
  background: #f8d7da;
  color: #842029;
  font-size: 14px;
}

/* Pinned to the bottom of the page while the form scrolls */
.post-actions {
  position: sticky;
  bottom: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 20px 24px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.post-submit {
  height: 48px;
  font-size: 16px;
}

.post-submit:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.post-cancel {
  height: 44px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
}
</style>
