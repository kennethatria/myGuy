<template>
  <div class="location-field" aria-live="polite">
    <template v-if="state === 'added'">
      <p class="location-status added">
        <span aria-hidden="true">📍</span> Your area is added
        <button type="button" class="location-link" @click="$emit('clear')">Remove</button>
      </p>
      <p class="location-hint">Only a rough area (about 500 m) is shared, never your exact spot.</p>
    </template>

    <template v-else>
      <button
        type="button"
        class="btn btn-outline location-button"
        :disabled="state === 'asking'"
        @click="$emit('request')"
      >
        <span aria-hidden="true">📍</span>
        {{ state === 'asking' ? 'Finding your area...' : state === 'unavailable' ? 'Try again' : 'Add my area' }}
      </button>
      <p v-if="state === 'denied'" class="location-hint warn">
        Location is blocked for this site in your browser settings. You can still post; it just won't show as nearby.
      </p>
      <p v-else-if="state === 'unavailable'" class="location-hint warn">
        Couldn't find your area. You can still post without it.
      </p>
      <p v-else class="location-hint">
        So people nearby see it first. Only a rough area (about 500 m) is shared, never your exact spot. Optional.
      </p>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { LocationState } from '@/composables/useRoughLocation'

defineProps<{ state: LocationState }>()
defineEmits<{ request: []; clear: [] }>()
</script>

<style scoped>
.location-field {
  margin-top: 1.25rem;
}

.location-button {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  min-height: 44px;
}

.location-status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-weight: 600;
}

.location-link {
  min-height: 44px;
  padding: 0 0.5rem;
  border: none;
  background: none;
  color: var(--color-primary);
  font-weight: 500;
  text-decoration: underline;
  cursor: pointer;
}

.location-hint {
  margin: 0.4rem 0 0;
  font-size: 0.875rem;
  color: var(--color-text-light, #6b7280);
}

.location-hint.warn {
  color: #92400e;
}
</style>
