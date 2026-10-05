<template>
  <component
    :is="to ? RouterLink : 'article'"
    :to="to"
    :class="['sticky-note', `note-${color}`, `note-${size}`, { 'is-link': to }]"
    :style="{ '--tilt': `${tilt}deg` }"
  >
    <slot name="header">
      <h3 class="note-headline">{{ title }}</h3>
      <p class="note-body">{{ body }}</p>
    </slot>
    <footer v-if="$slots.footer" class="note-footer">
      <slot name="footer" />
    </footer>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { noteColor, noteTilt } from '@/utils/gigNote'

const props = withDefaults(defineProps<{
  title?: string
  body?: string
  // Picks the note's colour and tilt; the gig id keeps them stable.
  seed?: number
  to?: RouteLocationRaw
  size?: 'small' | 'large'
  flat?: boolean
}>(), {
  title: '',
  body: '',
  seed: 0,
  size: 'small',
  flat: false
})

const color = computed(() => noteColor(props.seed))
const tilt = computed(() => (props.flat ? 0 : noteTilt(props.seed)))
</script>

<style scoped>
.sticky-note {
  --note-ink: #1f2937;
  --note-muted: #4b5563;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  color: var(--note-ink);
  text-decoration: none;
  background: var(--note-bg);
  border-radius: 2px 2px 6px 6px;
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.08), 0 6px 14px -6px rgba(0, 0, 0, 0.25);
  transform: rotate(var(--tilt));
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  overflow-wrap: anywhere;
}

.note-small {
  aspect-ratio: 1;
  padding: 1.1rem 1.1rem 0.9rem;
}

.note-large {
  padding: 1.75rem;
  min-height: 16rem;
}

.note-yellow { --note-bg: #fef3a3; }
.note-pink { --note-bg: #fbcfe8; }
.note-blue { --note-bg: #bfdbfe; }
.note-green { --note-bg: #bbf7d0; }
.note-orange { --note-bg: #fed7aa; }

.is-link:hover,
.is-link:focus-visible {
  transform: rotate(0deg) translateY(-2px);
  box-shadow: 0 2px 2px rgba(0, 0, 0, 0.08), 0 12px 20px -8px rgba(0, 0, 0, 0.3);
}

.is-link:focus-visible {
  outline: 3px solid var(--color-primary, #4f46e5);
  outline-offset: 3px;
}

/* One note per row on a phone: let it size to its words, not a big square */
@media (max-width: 480px) {
  .note-small {
    aspect-ratio: auto;
    min-height: 9rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sticky-note {
    transition: none;
  }
}

.note-headline {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 700;
  line-height: 1.25;
}

.note-large .note-headline {
  font-size: 1.6rem;
}

.note-body {
  margin: 0;
  font-size: 0.95rem;
  line-height: 1.45;
  flex: 1;
}

.note-large .note-body {
  font-size: 1.15rem;
}

.note-small .note-body {
  display: -webkit-box;
  -webkit-line-clamp: 5;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.note-footer {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 0.25rem 0.75rem;
  font-size: 0.8rem;
  color: var(--note-muted);
  border-top: 1px dashed rgba(31, 41, 55, 0.2);
  padding-top: 0.5rem;
}
</style>
