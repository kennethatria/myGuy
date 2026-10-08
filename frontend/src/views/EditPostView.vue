<template>
  <p v-if="loading" class="edit-status">Loading your post...</p>
  <p v-else-if="loadError" class="edit-status" role="alert">{{ loadError }}</p>
  <PostForm
    v-else
    v-model:title="title"
    v-model:description="description"
    :tone="TONE[kind]"
    hint="Keep phone numbers and links for the chat."
    :error="formError"
    :busy="saving"
    submit-label="Save"
    busy-label="Saving..."
    @submit="save"
    @cancel="router.back()"
  />
</template>

<script setup lang="ts">
// Change a post's headline and note. The service checks the
// same rules as when posting (word limits, no contact details).
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PostForm from '@/components/PostForm.vue'
import { TONE, fetchPost, postRoute, updatePost, type PostKind } from '@/utils/postApi'

const route = useRoute()
const router = useRouter()
const kind = computed(() => route.params.kind as PostKind)
const id = computed(() => Number(route.params.id))

const title = ref('')
const description = ref('')
const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const formError = ref('')

const save = async () => {
  saving.value = true
  formError.value = ''
  try {
    await updatePost(kind.value, id.value, { title: title.value.trim(), description: description.value.trim() })
    router.replace(postRoute(kind.value, id.value))
  } catch (error) {
    formError.value = error instanceof Error ? error.message : 'Could not save your changes. Please try again.'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const post = await fetchPost(kind.value, id.value)
    title.value = post.title
    description.value = post.description
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : 'Could not load your post.'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.edit-status {
  margin: 32px 20px;
  text-align: center;
  color: var(--text-muted);
}
</style>
