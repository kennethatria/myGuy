<template>
  <span v-if="distance" class="distance-tag">
    <span aria-hidden="true">📍</span> {{ distance }}
  </span>
  <span
    v-else-if="showUnknown"
    class="distance-tag unknown"
    role="img"
    aria-label="Distance unknown: no location was shared"
    title="Distance unknown: no location was shared"
  >🤷</span>
</template>

<script setup lang="ts">
// A note's rough distance from the viewer ("~2 km"), or a shrug when the
// poster didn't share a location. Show the shrug only where distances are
// in play (see hasDistances), or every note would carry one.
defineProps<{
  distance?: string
  showUnknown?: boolean
}>()
</script>

<style scoped>
.distance-tag {
  font-weight: 600;
  white-space: nowrap;
}

.distance-tag.unknown {
  font-size: 1.05rem;
  line-height: 1;
}
</style>
