<template>
  <div class="home-container">
    <h1 class="visually-hidden">Home</h1>
    <NearbyRadar />
  </div>
</template>

<script setup lang="ts">
// Home: what's near you. Your own gigs and applications are in the side
// navigation (Created Gigs, Assignments, Applications).
import NearbyRadar from '@/components/NearbyRadar.vue'
</script>

<style scoped>
.home-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

@media (max-width: 768px) {
  .home-container {
    padding: 1rem;
  }
}
</style>
