<template>
  <div class="deletion-warning-banner">
    <div class="warning-content">
      <i class="fas fa-exclamation-triangle"></i>
      <div class="warning-text">
        <h3>Message Deletion Notice</h3>
        <p>The following conversations will have their messages deleted soon:</p>
        <ul>
          <li v-for="warning in warnings" :key="warning.id">
            <strong>{{ warning.task_title }}</strong> - Messages will be deleted on 
            <strong>{{ formatDate(warning.deletion_scheduled_at) }}</strong>
          </li>
        </ul>
        <p class="warning-note">
          Messages are automatically deleted 6 months after task completion or 1 month after inactivity.
        </p>
      </div>
      <button @click="dismissWarnings" class="dismiss-btn">
        <i class="fas fa-times"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
interface DeletionWarning {
  id: number;
  task_id: number;
  task_title: string;
  deletion_scheduled_at: string;
}

const props = defineProps<{
  warnings: DeletionWarning[];
}>();

const emit = defineEmits<{
  dismiss: [warningId: number];
}>();

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString([], { 
    year: 'numeric', 
    month: 'long', 
    day: 'numeric' 
  });
}

function dismissWarnings() {
  // Dismiss all warnings
  props.warnings.forEach(warning => {
    emit('dismiss', warning.id);
  });
}
</script>

<style scoped>
.deletion-warning-banner {
  background: #fef3c7;
  border: 1px solid #fbbf24;
  border-radius: 0.5rem;
  padding: 1rem;
  margin: 1rem;
}

.warning-content {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.warning-content > i {
  color: #f59e0b;
  font-size: 1.5rem;
  flex-shrink: 0;
}

.warning-text {
  flex: 1;
}

.warning-text h3 {
  font-size: 1rem;
  font-weight: 600;
  color: #92400e;
  margin: 0 0 0.5rem 0;
}

.warning-text p {
  font-size: 0.875rem;
  color: #92400e;
  margin: 0 0 0.5rem 0;
}

.warning-text ul {
  margin: 0 0 0.5rem 0;
  padding-left: 1.5rem;
}

.warning-text li {
  font-size: 0.875rem;
  color: #92400e;
  margin-bottom: 0.25rem;
}

.warning-note {
  font-size: 0.75rem !important;
  font-style: italic;
  opacity: 0.8;
}

.dismiss-btn {
  background: transparent;
  border: none;
  color: #92400e;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 0.25rem;
  transition: background-color 0.15s;
}

.dismiss-btn:hover {
  background: rgba(0, 0, 0, 0.05);
}
</style>