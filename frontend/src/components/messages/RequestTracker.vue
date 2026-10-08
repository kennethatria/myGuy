<template>
  <!-- Where a booking or application stands: sent, with the other person,
       then the chat opens -->
  <ol class="tracker" :aria-label="`Step ${current + 1} of ${steps.length}: ${steps[current]}`">
    <template v-for="(step, index) in steps" :key="step">
      <li v-if="index > 0" class="tracker-line" aria-hidden="true"></li>
      <li :class="['tracker-step', index < current ? 'done' : index === current ? 'current' : 'todo']">
        <span class="tracker-dot" aria-hidden="true"></span>
        <span class="tracker-label">{{ step }}</span>
      </li>
    </template>
  </ol>
</template>

<script setup lang="ts">
defineProps<{
  steps: string[]
  // Index of the step under way
  current: number
}>()
</script>

<style scoped>
.tracker {
  display: flex;
  align-items: center;
  margin: 0;
  padding: 4px 4px 0;
  list-style: none;
}

.tracker-step {
  width: 56px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.tracker-dot {
  width: 14px;
  height: 14px;
  border-radius: 7px;
  box-sizing: border-box;
}

.done .tracker-dot {
  background: var(--accent);
}

.current .tracker-dot {
  border: 2px solid var(--accent);
  background: transparent;
}

.todo .tracker-dot {
  background: rgba(17, 24, 39, 0.15);
}

.tracker-label {
  font-size: 11px;
  color: var(--text-body);
}

.current .tracker-label {
  font-weight: 700;
  color: var(--text);
}

.todo .tracker-label {
  color: var(--text-muted);
}

.tracker-line {
  flex: 1;
  height: 2px;
  margin-bottom: 16px;
  background: repeating-linear-gradient(90deg, rgba(17, 24, 39, 0.3) 0 4px, transparent 4px 8px);
}
</style>
