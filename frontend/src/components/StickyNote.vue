<template>
  <component
    :is="to ? RouterLink : 'article'"
    :to="to"
    :class="['sticky-note', `note-${color}`, `note-${size}`, { 'is-link': to, 'has-photo': photo, 'has-tape': tape }]"
    :style="{ '--tilt': `${tilt}deg` }"
  >
    <span v-if="tape" class="note-tape" aria-hidden="true"></span>
    <span v-if="pin" class="note-pin" aria-hidden="true"></span>
    <figure v-if="photo" class="note-photo">
      <img :src="photo" :alt="photoAlt" loading="lazy" />
    </figure>
    <slot name="header">
      <h3 class="note-headline">{{ title }}</h3>
      <p class="note-body">{{ body }}</p>
    </slot>
    <footer v-if="$slots.footer" class="note-footer">
      <slot name="footer" />
    </footer>
    <span v-if="fold" class="note-fold" aria-hidden="true"></span>
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
  // Colour by what the note is, instead of by seed
  tone?: 'gig' | 'sell' | 'want'
  to?: RouteLocationRaw
  // A row is one line of a list: the slot lays out its own content
  size?: 'small' | 'large' | 'row'
  flat?: boolean
  // A photo taped to the top of the note (marketplace listings)
  photo?: string
  photoAlt?: string
  // A strip of tape across the top edge, a pin near it (Home's rows), and
  // a folded bottom-right corner
  tape?: boolean
  pin?: boolean
  fold?: boolean
}>(), {
  title: '',
  body: '',
  seed: 0,
  size: 'small',
  flat: false,
  photo: '',
  photoAlt: '',
  tape: false,
  pin: false,
  fold: false
})

const color = computed(() => props.tone ?? noteColor(props.seed))
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
  position: relative;
  background: var(--note-bg);
  border-radius: 4px;
  box-shadow: var(--note-shadow);
  transform: rotate(var(--tilt));
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  overflow-wrap: anywhere;
}

/* Sized to its words, like a real note */
.note-small {
  gap: 10px;
  padding: 14px;
}

/* Room under the tape */
.note-small.has-tape {
  padding-top: 20px;
}

.note-large {
  padding: 1.75rem;
  min-height: 16rem;
}

/* The photo sits in a white frame on the note */
.note-photo {
  position: relative;
  margin: 0;
  padding: 5px;
  border-radius: 3px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(17, 24, 39, 0.12);
}

/* A strip of tape holding the photo on, unless the note is taped itself */
.sticky-note:not(.has-tape) .note-photo::before {
  content: '';
  position: absolute;
  top: -0.5rem;
  left: 50%;
  width: 3.5rem;
  height: 1rem;
  transform: translateX(-50%) rotate(-3deg);
  background: rgba(255, 255, 255, 0.55);
  box-shadow: 0 0 1px rgba(0, 0, 0, 0.2);
}

.note-photo img {
  display: block;
  width: 100%;
  aspect-ratio: 4 / 3;
  border-radius: 2px;
  object-fit: cover;
}

.note-small .note-photo img {
  aspect-ratio: auto;
  height: 150px;
}

.note-large .note-photo img {
  aspect-ratio: 16 / 10;
}

.note-tape {
  position: absolute;
  top: -7px;
  left: 50%;
  width: 48px;
  height: 14px;
  margin-left: -24px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.6);
}

.note-pin {
  position: absolute;
  top: 6px;
  left: 50%;
  width: 8px;
  height: 8px;
  margin-left: -4px;
  border-radius: 4px;
  background: radial-gradient(circle at 35% 30%, #E5E7EB, #9CA3AF);
  box-shadow: 0 1px 1px rgba(17, 24, 39, 0.25);
}

/* Photo notes take a wider strip */
.has-photo > .note-tape {
  top: -8px;
  width: 64px;
  height: 18px;
  margin-left: -32px;
  background: rgba(255, 255, 255, 0.55);
}

.note-fold {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 14px;
  height: 14px;
  border-radius: 0 0 4px 0;
  background: linear-gradient(135deg, var(--bg) 50%, rgba(17, 24, 39, 0.14) 50%);
}

.note-row {
  flex-direction: row;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
}

.note-gig { --note-bg: var(--note-gig); }
.note-sell { --note-bg: var(--note-sell); }
.note-want { --note-bg: var(--note-want); }

.note-yellow { --note-bg: #fef3a3; }
.note-pink { --note-bg: #fbcfe8; }
.note-blue { --note-bg: #bfdbfe; }
.note-green { --note-bg: #bbf7d0; }
.note-orange { --note-bg: #fed7aa; }

.is-link:hover,
.is-link:focus-visible {
  transform: rotate(0deg) translateY(-2px);
  box-shadow: 0 4px 10px rgba(17, 24, 39, 0.16), 0 1px 2px rgba(17, 24, 39, 0.08);
}

.is-link:focus-visible {
  outline: 3px solid var(--color-primary);
  outline-offset: 3px;
}

/* One note per row on a phone: let it size to its words, not a big square */
@media (max-width: 480px) {
  .note-large {
    padding: 1.25rem;
    min-height: 0;
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
  border-top: 1px dashed rgba(17, 24, 39, 0.22);
  padding-top: 0.5rem;
}
</style>
