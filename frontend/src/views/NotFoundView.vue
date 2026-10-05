<template>
  <div class="not-found">
    <img class="logo" src="../assets/myguy-icon.svg" alt="MyGuy" />
    <h1>Page not found</h1>
    <p>The link may be old or mistyped.</p>
    <router-link :to="{ name: authStore.isAuthenticated ? 'dashboard' : 'home' }" class="btn btn-primary">
      {{ authStore.isAuthenticated ? 'Go to your dashboard' : 'Go to the home page' }}
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
</script>

<style scoped>
.not-found {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 1.5rem;
  text-align: center;
}

.logo {
  height: 3rem;
}

.not-found p {
  color: #6b7280;
  margin-bottom: 0.5rem;
}
</style>
