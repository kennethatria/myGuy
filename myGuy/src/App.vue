<script setup lang="ts">
import { RouterView, useRoute } from 'vue-router'
import { computed } from 'vue'
import AppLayout from './components/AppLayout.vue'

const route = useRoute()

// Show AppLayout only for routes that are not the homepage or auth pages
const showLayout = computed(() => {
  return !['home', 'login', 'register'].includes(route.name as string)
})
</script>

<template>
  <app-layout v-if="showLayout" />
  <router-view v-else />
</template>

<style>
/* Custom styles are imported in main.ts */
</style>
