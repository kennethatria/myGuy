<template>
  <!-- Asked only on a tap, and optional: posts without an area still go up -->
  <div class="location-field" aria-live="polite">
    <span class="location-pin" aria-hidden="true">📍</span>
    <div class="location-text">
      <span class="location-title">{{ title }}</span>
      <span :class="['location-hint', { warn: state === 'denied' || state === 'unavailable' }]">{{ hint }}</span>
    </div>
    <button v-if="state === 'added'" type="button" class="location-action" @click="$emit('clear')">Remove</button>
    <button
      v-else-if="state !== 'denied'"
      type="button"
      class="location-action"
      :disabled="state === 'asking'"
      @click="$emit('request')"
    >
      {{ state === 'asking' ? 'Finding...' : state === 'unavailable' ? 'Try again' : 'Add' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { LocationState } from '@/composables/useRoughLocation'

const props = defineProps<{ state: LocationState }>()
defineEmits<{ request: []; clear: [] }>()

const title = computed(() => (props.state === 'added' ? 'Rough area added' : 'Add my area'))
const hint = computed(() => {
  if (props.state === 'denied') return "Location is blocked for this site in your browser. You can still post; it just won't show as nearby."
  if (props.state === 'unavailable') return "Couldn't find your area. You can still post without it."
  if (props.state === 'added') return 'About 500 m, never your exact spot'
  return 'Optional. So people nearby see it first; about 500 m, never your exact spot'
})
</script>

<style scoped>
.location-field {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}

.location-pin {
  font-size: 18px;
}

.location-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.location-title {
  font-size: 14px;
  font-weight: 600;
}

.location-hint {
  font-size: 12px;
  color: var(--text-muted);
}

.location-hint.warn {
  color: #92400e;
}

.location-action {
  flex: none;
  min-height: 44px;
  padding: 0 8px;
  border: 0;
  background: transparent;
  color: var(--accent-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.location-action:disabled {
  color: var(--text-muted);
  cursor: default;
}
</style>
