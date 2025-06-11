<script setup lang="ts">
import { RouterView, useRoute } from 'vue-router'
import { computed } from 'vue'
import AppLayout from './components/AppLayout.vue'
import ModernAppLayout from './components/ModernAppLayout.vue'
import ChatWidget from './components/messages/ChatWidget.vue'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const authStore = useAuthStore()

// Use modern layout for authenticated routes
const useModernLayout = true

// Show layout only for routes that are not the homepage or auth pages
const showLayout = computed(() => {
  return !['home', 'login', 'register'].includes(route.name as string)
})

// Show chat widget on all authenticated pages except message center
const showChatWidget = computed(() => {
  return authStore.isAuthenticated && route.name !== 'messages'
})
</script>

<template>
  <div class="app-container">
    <modern-app-layout v-if="showLayout && useModernLayout" />
    <app-layout v-else-if="showLayout && !useModernLayout" />
    <router-view v-else />
    <chat-widget v-if="showChatWidget" />
  </div>
</template>

<style>
/* Custom styles are imported in main.ts */
.app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  height: 100%;
}

/* Error handling styles */
.text-red-500 {
  color: #ef4444;
}
</style>
