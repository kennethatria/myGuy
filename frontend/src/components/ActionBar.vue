<template>
  <!-- Keeps the page scrollable past the bar -->
  <div class="action-bar-spacer" :style="{ height: `${height}px` }" aria-hidden="true"></div>
  <div ref="bar" class="action-bar">
    <div class="action-bar-inner">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
// A detail page's main action, pinned to the bottom of the screen
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { setActionBarHeight } from '@/composables/useActionBar'

const bar = ref<HTMLElement | null>(null)
const height = ref(0)
let observer: ResizeObserver | undefined

const measure = () => {
  height.value = bar.value?.offsetHeight ?? 0
  setActionBarHeight(height.value)
}

onMounted(() => {
  measure()
  if (typeof ResizeObserver !== 'undefined' && bar.value) {
    observer = new ResizeObserver(measure)
    observer.observe(bar.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
  setActionBarHeight(0)
})
</script>

<style scoped>
.action-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 800; /* under the chat panel and the menu */
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.action-bar-inner {
  max-width: 560px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 20px 24px;
}

/* Buttons in the bar fill its width */
.action-bar-inner :deep(.btn) {
  width: 100%;
  min-height: 48px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
}

.action-bar-inner :deep(.bar-caption) {
  margin: 0;
  font-size: 12px;
  text-align: center;
  color: var(--text-muted);
}

.action-bar-inner :deep(.bar-message) {
  margin: 0;
  font-size: 14px;
  text-align: center;
  color: var(--text-body);
}

.action-bar-inner :deep(.bar-error) {
  margin: 0;
  font-size: 14px;
  text-align: center;
  color: #B91C1C;
}

.action-bar-inner :deep(.bar-row) {
  display: flex;
  gap: 8px;
}
</style>
