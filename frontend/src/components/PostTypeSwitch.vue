<template>
  <!-- Gig, Sell or Request: each is its own composer page -->
  <nav class="type-switch" aria-label="What are you posting?">
    <router-link
      v-for="type in TYPES"
      :key="type.kind"
      :to="{ name: type.route }"
      replace
      :class="['type-option', `type-${TONE[type.kind]}`, { active: type.kind === current }]"
      :aria-current="type.kind === current ? 'page' : undefined"
    >
      <span aria-hidden="true">{{ type.emoji }}</span> {{ type.label }}
    </router-link>
  </nav>
</template>

<script setup lang="ts">
import { TONE, type PostKind } from '@/utils/postApi'

defineProps<{ current: PostKind }>()

const TYPES: { kind: PostKind; route: string; label: string; emoji: string }[] = [
  { kind: 'task', route: 'create-task', label: 'Gig', emoji: '🛠' },
  { kind: 'item', route: 'create-listing', label: 'Sell', emoji: '📦' },
  { kind: 'request', route: 'create-request', label: 'Request', emoji: '🙋' }
]
</script>

<style scoped>
.type-switch {
  display: flex;
  gap: 8px;
}

.type-option {
  flex: 1;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 2px solid transparent;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
}

.type-gig { background: #E8F0FF; color: #1E3A8A; }
.type-sell { background: #FBF1CE; color: #713F12; }
.type-want { background: #FCE8F2; color: #831843; }

.type-option:hover {
  color: inherit;
  filter: brightness(0.97);
}

.type-gig:hover { color: #1E3A8A; }
.type-sell:hover { color: #713F12; }
.type-want:hover { color: #831843; }

.type-option.active {
  border-color: currentColor;
  font-weight: 700;
}
</style>
