<template>
  <div class="home-container">
    <h1 class="page-title">Home</h1>
    <NearbyRadar />
  </div>
</template>

<script setup lang="ts">
// Home: what's near you. Your own gigs and applications are in the side
// navigation (My Created Gigs, Gigs Assigned to Me, My Applications).
import NearbyRadar from '@/components/NearbyRadar.vue'
</script>

<style scoped>
.home-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.page-title {
  margin: 0 0 1.25rem;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--color-text, #111827);
}

@media (max-width: 768px) {
  .home-container {
    padding: 1rem;
  }

  .page-title {
    font-size: 1.5rem;
  }
}
</style>
