<template>
  <PostForm
    v-model:title="title"
    v-model:description="description"
    tone="want"
    headline-placeholder="Printer wanted"
    note-placeholder="Any working laser printer for a small office. Budget around UGX 300,000."
    hint="Live for 24 hours. Sellers who have it can list it for you, and you get a message. Keep phone numbers and links for the chat."
    :error="formError"
    :busy="isSubmitting"
    @submit="handleSubmit"
    @cancel="router.back()"
  >
    <template #before>
      <PostTypeSwitch current="request" />
    </template>
    <template #extras>
      <LocationField :state="location.state.value" @request="location.request" @clear="location.clear" />
    </template>
  </PostForm>
</template>

<script setup lang="ts">
// A request ("wanted" note): sellers answer it with a listing
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import config from '@/config'
import PostForm from '@/components/PostForm.vue'
import PostTypeSwitch from '@/components/PostTypeSwitch.vue'
import LocationField from '@/components/LocationField.vue'
import { useRoughLocation } from '@/composables/useRoughLocation'
import { trackEvent } from '@/utils/analytics'

const router = useRouter()
const authStore = useAuthStore()

const title = ref('')
const description = ref('')
const location = useRoughLocation()
const isSubmitting = ref(false)
const formError = ref('')

const handleSubmit = async () => {
  isSubmitting.value = true
  formError.value = ''

  try {
    const response = await fetch(`${config.STORE_API_URL}/requests`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${authStore.token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: title.value.trim(),
        description: description.value.trim(),
        ...(location.location.value ?? {})
      })
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      // store-service explains what to change (limits, contact details)
      throw new Error(data.error || 'Could not post your request. Please try again.')
    }
    trackEvent('request-posted')
    router.push({ name: 'posted', params: { kind: 'request', id: data.id } })
  } catch (error) {
    formError.value = error instanceof Error ? error.message : 'Could not post your request. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>
