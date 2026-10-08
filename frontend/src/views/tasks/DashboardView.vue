<template>
  <div class="home-container">
    <NearbyRadar />
  </div>
</template>

<script setup lang="ts">
// Home: what's near you (the header reads "Near you"). Your own gigs and
// applications are in the menu (My stuff, Assignments, Applications).
import NearbyRadar from '@/components/NearbyRadar.vue'
</script>

<style scoped>
.home-container {
  padding: 0 0 1rem;
}
</style>
