<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppLayout from './components/AppLayout.vue'
</script>

<template>
  <app-layout>
    <router-view></router-view>
  </app-layout>
</template>

<style>
@import '@/assets/main.css';
</style>
