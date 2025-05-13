<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppLayout from './components/AppLayout.vue'
</script>

<template>
  <app-layout />
</template>

<style>
/* Skip importing main.css as it conflicts with Tailwind */
/* @import '@/assets/main.css'; */

/* Tailwind is already imported at the project level */
</style>
