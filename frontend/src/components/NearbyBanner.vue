<template>
  <div v-if="state !== 'ready'" class="nearby-banner" aria-live="polite">
    <template v-if="state === 'denied'">
      <p class="nearby-text">Location is off for this site, so notes are newest first.</p>
    </template>
    <template v-else>
      <p class="nearby-text">
        {{ state === 'unavailable' ? "Couldn't find your area." : 'See what is closest to you first.' }}
        Only a rough area (about 500 m) is used.
      </p>
      <button type="button" class="btn btn-outline nearby-button" :disabled="state === 'asking'" @click="$emit('request')">
        <span aria-hidden="true">📍</span>
        {{ state === 'asking' ? 'Finding your area...' : state === 'unavailable' ? 'Try again' : "Show what's near me" }}
      </button>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { ViewerLocationState } from '@/composables/useViewerLocation'

defineProps<{ state: ViewerLocationState }>()
defineEmits<{ request: [] }>()
</script>

<style scoped>
.nearby-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem 1rem;
  margin-bottom: 1.25rem;
  padding: 0.75rem 1rem;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 0.5rem;
  background: #fff;
}

.nearby-text {
  margin: 0;
  flex: 1 1 14rem;
  font-size: 0.9rem;
  color: var(--color-text-light, #6b7280);
}

.nearby-button {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  min-height: 44px;
}

@media (max-width: 480px) {
  .nearby-button {
    width: 100%;
    justify-content: center;
  }
}
</style>
