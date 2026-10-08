<template>
  <div class="posted-page">
    <div class="celebration">
      <div class="celebration-tile" aria-hidden="true">🎉</div>
      <h2 class="celebration-title">Posted!</h2>
      <p class="celebration-text">Live on the board for 24 hours</p>
    </div>

    <p v-if="loading" class="posted-status">Loading your post...</p>
    <p v-else-if="loadError" class="posted-status" role="alert">{{ loadError }}</p>

    <template v-else-if="post">
      <!-- The note itself opens the post's page -->
      <StickyNote class="summary" size="large" :tone="TONE[kind]" :to="postRoute(kind, post.id)" flat>
        <template #header>
          <span class="summary-head">
            <span class="summary-emoji" aria-hidden="true">{{ EMOJI[kind] }}</span>
            <span class="summary-title">{{ post.title }}</span>
            <span class="summary-status">{{ statusText }}</span>
          </span>
          <span class="summary-body">{{ post.description }}</span>
          <span class="summary-foot">Your post<template v-if="left"> · {{ left }} left</template></span>
        </template>
      </StickyNote>

      <div v-if="confirming" class="confirm" role="group" aria-label="Remove this post?">
        <p class="confirm-text">Remove this post for good?</p>
        <div class="actions">
          <button type="button" class="btn btn-danger action" :disabled="removing" @click="remove">
            {{ removing ? 'Removing...' : 'Yes, remove' }}
          </button>
          <button type="button" class="action action-secondary" :disabled="removing" @click="confirming = false">Keep it</button>
        </div>
      </div>
      <div v-else class="actions">
        <router-link v-if="EDITABLE.includes(kind)" :to="{ name: 'edit-post', params: { kind, id: post.id } }" class="btn btn-primary action">
          Edit
        </router-link>
        <button type="button" class="action action-secondary action-danger" @click="confirming = true">Remove</button>
      </div>
      <p v-if="removeError" class="posted-error" role="alert">{{ removeError }}</p>
    </template>

    <div v-if="toast" class="toast" role="status">
      <span aria-hidden="true">✅</span> Stuck on the board
    </div>
  </div>
</template>

<script setup lang="ts">
// Straight after posting: the new note, with Edit and Remove
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import StickyNote from '@/components/StickyNote.vue'
import { timeLeft } from '@/utils/gigNote'
import { EDITABLE, TONE, fetchPost, postRoute, removePost, type PostKind, type PostNote } from '@/utils/postApi'

const EMOJI: Record<PostKind, string> = { task: '🛠', item: '📦', request: '🙋' }
// Where your own posts of each kind are listed
const AFTER_REMOVE = {
  task: { name: 'my-gigs', params: { tab: 'created' } },
  item: { name: 'store', query: { tab: 'mine' } },
  request: { name: 'store', query: { tab: 'mine' } }
} as const

const route = useRoute()
const router = useRouter()
const kind = computed(() => route.params.kind as PostKind)
const id = computed(() => Number(route.params.id))

const post = ref<PostNote | null>(null)
const loading = ref(true)
const loadError = ref('')
const confirming = ref(false)
const removing = ref(false)
const removeError = ref('')
const toast = ref(true)
let toastTimer: ReturnType<typeof setTimeout> | undefined

const left = computed(() => (post.value ? timeLeft(post.value.deadline) : ''))
const statusText = computed(() => {
  const status = post.value?.status ?? ''
  if (status === 'open' || status === 'active') return '🟢 Open'
  return status ? status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ') : ''
})

const remove = async () => {
  removing.value = true
  removeError.value = ''
  try {
    await removePost(kind.value, id.value)
    router.replace(AFTER_REMOVE[kind.value])
  } catch (error) {
    removeError.value = error instanceof Error ? error.message : 'Could not remove the post. Please try again.'
    confirming.value = false
  } finally {
    removing.value = false
  }
}

onMounted(async () => {
  toastTimer = setTimeout(() => (toast.value = false), 4000)
  try {
    post.value = await fetchPost(kind.value, id.value)
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : 'Could not load your post.'
  } finally {
    loading.value = false
  }
})

onUnmounted(() => clearTimeout(toastTimer))
</script>

<style scoped>
.posted-page {
  max-width: 440px;
  margin: 0 auto;
  padding: 32px 20px 24px;
}

.celebration {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  text-align: center;
}

.celebration-tile {
  width: 160px;
  height: 160px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #A5ACFF;
  border-radius: 28px;
  background: #F5F6FF;
  font-size: 64px;
}

.celebration-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}

.celebration-text {
  margin: -12px 0 0;
  font-size: 15px;
  color: var(--text-muted);
}

.posted-status {
  margin: 24px 0 0;
  text-align: center;
  color: var(--text-muted);
}

/* Doubled to win over StickyNote's own size and tilt */
.summary.summary {
  margin: 24px 8px 0;
  min-height: 0;
  padding: 16px;
  transform: rotate(-1deg);
}

.summary-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.summary-emoji {
  font-size: 20px;
}

.summary-title {
  flex: 1;
  min-width: 0;
  font-size: 17px;
  font-weight: 600;
}

.summary-status {
  flex: none;
  padding: 3px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.7);
  color: #065F46;
  font-size: 12px;
  font-weight: 600;
}

.summary-body {
  font-size: 15px;
  line-height: 1.45;
  color: #374151;
}

.summary-foot {
  padding-top: 8px;
  border-top: 1px dashed rgba(17, 24, 39, 0.25);
  font-size: 13px;
  color: var(--text-body);
}

.actions {
  display: flex;
  gap: 10px;
  margin-top: 20px;
}

.action {
  flex: 1;
  height: 48px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
}

.action-secondary {
  border: 1px solid #D1D5DB;
  border-radius: 12px;
  background: var(--surface);
  color: var(--text);
  cursor: pointer;
}

.action-danger {
  color: #B91C1C;
}

.confirm {
  margin-top: 20px;
}

.confirm .actions {
  margin-top: 8px;
}

.confirm-text {
  margin: 0;
  text-align: center;
  font-weight: 600;
}

.posted-error {
  margin: 12px 0 0;
  color: #B91C1C;
  text-align: center;
}

.toast {
  position: fixed;
  left: 20px;
  right: 20px;
  bottom: 24px;
  z-index: 950;
  max-width: 400px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--text);
  color: #fff;
  font-size: 14px;
}
</style>
