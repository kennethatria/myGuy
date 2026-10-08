<template>
  <!-- A post's own page: the note, big, with who posted it and its time -->
  <StickyNote :tone="tone" :seed="seed" size="large" class="detail-note">
    <template #header>
      <span v-if="status" class="detail-status">{{ status }}</span>
      <span class="detail-top">
        <h1 class="detail-title">{{ title }}</h1>
        <span v-if="price" class="detail-price">{{ price }}</span>
      </span>
      <p class="detail-body">{{ body }}</p>
    </template>
    <template #footer>
      <span class="detail-person">
        <span class="detail-avatar" aria-hidden="true">{{ initial }}</span>
        <router-link v-if="person" :to="{ name: 'user-profile', params: { id: String(person.id) } }" class="detail-name">
          @{{ person.username }}
        </router-link>
        <span v-else class="detail-name">@someone</span>
        <slot name="person" />
      </span>
      <span v-if="meta" class="detail-meta">{{ meta }}</span>
    </template>
  </StickyNote>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StickyNote from '@/components/StickyNote.vue'

const props = withDefaults(defineProps<{
  tone: 'gig' | 'sell' | 'want'
  seed: number
  title: string
  body: string
  price?: string
  status?: string
  person?: { id: number; username: string } | null
  // "Posted 2h ago · Expires in 23h"
  meta?: string
}>(), {
  price: '',
  status: '',
  person: null,
  meta: ''
})

const initial = computed(() => (props.person?.username ?? '?').charAt(0).toUpperCase())
</script>

<style scoped>
/* Doubled to win over StickyNote's own large size */
.detail-note.detail-note {
  min-height: 0;
  padding: 14px 16px;
  margin: 4px 4px 0;
  gap: 8px;
  border-radius: 6px;
  transform: rotate(-0.6deg);
}

.detail-status {
  align-self: flex-start;
  padding: 3px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  font-weight: 600;
  color: var(--text-body);
}

.detail-top {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.detail-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.25;
}

.detail-price {
  flex: none;
  font-size: 20px;
  font-weight: 700;
  color: var(--price);
}

.detail-body {
  margin: 0;
  font-size: 15px;
  line-height: 1.45;
  color: #374151;
}

.detail-person {
  flex: 1 1 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-body);
}

.detail-avatar {
  flex: none;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.75);
  color: var(--accent-text);
  font-size: 12px;
  font-weight: 700;
}

.detail-name {
  color: var(--text-body);
  font-weight: 600;
}

a.detail-name:hover {
  color: var(--text);
  text-decoration: underline;
}

.detail-meta {
  flex: 1 1 100%;
  font-size: 12px;
  color: var(--text-muted);
}
</style>
