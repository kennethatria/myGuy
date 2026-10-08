<template>
  <!-- An empty list: a picture, what will show up here, and what to do -->
  <div class="empty-state">
    <div class="empty-picture" aria-hidden="true">{{ emoji }}</div>
    <h2 class="empty-title">{{ title }}</h2>
    <p class="empty-text">{{ text }}</p>
    <router-link v-if="action" :to="action.to" class="btn btn-primary empty-action">{{ action.label }}</router-link>
    <router-link v-if="secondary" :to="secondary.to" class="empty-secondary">{{ secondary.label }}</router-link>
  </div>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'

interface Link {
  label: string
  to: RouteLocationRaw
}

withDefaults(defineProps<{
  title: string
  text: string
  emoji?: string
  action?: Link | null
  secondary?: Link | null
}>(), {
  emoji: '🌱',
  action: null,
  secondary: null
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 32px;
  text-align: center;
}

.empty-picture {
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 8px;
  border: 1px dashed #A5ACFF;
  border-radius: 24px;
  background: #F5F6FF;
  font-size: 52px;
}

.empty-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--text);
}

.empty-text {
  max-width: 320px;
  margin: 0;
  font-size: 15px;
  line-height: 1.5;
  color: var(--text-muted);
}

.empty-action {
  min-height: 48px;
  margin-top: 12px;
  padding: 0 24px;
  font-size: 16px;
}

.empty-secondary {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  color: var(--accent-text);
  font-size: 15px;
  font-weight: 600;
}
</style>
