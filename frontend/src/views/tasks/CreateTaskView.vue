<template>
  <PostForm
    v-model:title="title"
    v-model:description="description"
    tone="gig"
    headline-placeholder="Need my fence painted"
    note-placeholder="Small garden fence in Ntinda, paint provided. Saturday morning works best."
    :error="formError"
    :busy="isSubmitting"
    @submit="handleSubmit"
    @cancel="router.back()"
  >
    <template #before>
      <PostTypeSwitch current="task" />
    </template>
    <template #extras>
      <LocationField :state="location.state.value" @request="location.request" @clear="location.clear" />
    </template>
  </PostForm>
</template>

<script setup lang="ts">
// A gig: a headline and a note, no fee or deadline (agreed in chat; the
// backend gives it 24 hours)
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import PostForm from '@/components/PostForm.vue'
import PostTypeSwitch from '@/components/PostTypeSwitch.vue'
import LocationField from '@/components/LocationField.vue'
import { useRoughLocation } from '@/composables/useRoughLocation'

const router = useRouter()
const tasksStore = useTasksStore()

const title = ref('')
const description = ref('')
const location = useRoughLocation()
const isSubmitting = ref(false)
const formError = ref('')

const handleSubmit = async () => {
  isSubmitting.value = true
  formError.value = ''

  try {
    const created = await tasksStore.createTask({
      title: title.value.trim(),
      description: description.value.trim(),
      ...(location.location.value ?? {})
    })
    router.push({ name: 'posted', params: { kind: 'task', id: created.id } })
  } catch (error) {
    // The backend explains what to change (limits, contact details)
    formError.value = error instanceof Error ? error.message : 'Could not post your note. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>
