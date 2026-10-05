<template>
  <Teleport to="body">
    <div v-if="isOpen" class="modal-backdrop" @click="handleBackdropClick">
      <div class="modal-container" @click.stop>
        <div class="modal-header">
          <h2>Reply to this note</h2>
          <button @click="close" class="close-btn" aria-label="Close">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <div class="task-info">
            <h3>{{ task.title }}</h3>
            <p class="task-meta">
              Posted by {{ task.creator?.username || 'Unknown' }}
            </p>
          </div>

          <form @submit.prevent="handleSubmit">
            <div class="form-group">
              <label for="message" class="form-label">
                Message
                <span class="required">*</span>
              </label>
              <textarea
                id="message"
                v-model="formData.message"
                rows="5"
                class="form-input"
                :class="{ 'is-invalid': errors.message }"
                :maxlength="MESSAGE_MAX"
                placeholder="Say why you're a good fit and when you could do it..."
                required
              ></textarea>
              <div v-if="errors.message" class="invalid-feedback">
                {{ errors.message }}
              </div>
              <p class="form-helper">
                Agree the price in chat. Keep phone numbers for after the poster accepts you.
                ({{ formData.message.length }}/{{ MESSAGE_MAX }})
              </p>
            </div>

            <div v-if="error" class="alert alert-danger">
              {{ error }}
            </div>

            <div class="modal-footer">
              <button type="button" @click="close" class="btn btn-secondary">
                Cancel
              </button>
              <button type="submit" class="btn btn-primary" :disabled="isSubmitting">
                {{ isSubmitting ? 'Sending...' : 'Send' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'

interface Task {
  id: number
  title: string
  creator?: {
    id: number
    username: string
  }
}

interface Props {
  isOpen: boolean
  task: Task
}

interface FormData {
  message: string
}

// Matches the backend's limit on application messages
const MESSAGE_MAX = 500

const props = defineProps<Props>()
const emit = defineEmits<{
  'close': []
  'submit': [data: { message: string }]
}>()

const formData = reactive<FormData>({
  message: ''
})

const errors = reactive({
  message: ''
})

const error = ref('')
const isSubmitting = ref(false)

// Reset form when modal opens
watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    formData.message = ''
    errors.message = ''
    error.value = ''
    isSubmitting.value = false
  }
})

const validateForm = (): boolean => {
  const message = formData.message.trim()
  errors.message = ''

  if (message.length < 10) {
    errors.message = 'Please write at least a short sentence (10 characters).'
  } else if (message.length > MESSAGE_MAX) {
    errors.message = `Please keep it under ${MESSAGE_MAX} characters.`
  }

  return !errors.message
}

const handleSubmit = () => {
  if (!validateForm()) {
    return
  }

  emit('submit', { message: formData.message.trim() })
}

const close = () => {
  emit('close')
}

const handleBackdropClick = () => {
  // Close modal when clicking outside
  close()
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-container {
  background: white;
  border-radius: 8px;
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e0e0e0;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: #666;
  transition: color 0.2s;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
}

.task-info {
  background-color: #f8f9fa;
  padding: 1rem;
  border-radius: 6px;
  margin-bottom: 1.5rem;
}

.task-info h3 {
  margin: 0 0 0.5rem 0;
  color: #333;
}

.task-meta {
  margin: 0;
  color: #666;
  font-size: 0.875rem;
}

.separator {
  margin: 0 0.5rem;
}

.budget {
  color: #28a745;
  font-weight: 500;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #333;
}

.required {
  color: #dc3545;
}

.optional {
  color: #6c757d;
  font-weight: normal;
  font-style: italic;
}

.form-input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
  transition: border-color 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.1);
}

.form-input.is-invalid {
  border-color: #dc3545;
}

textarea.form-input {
  resize: vertical;
  min-height: 100px;
}

.invalid-feedback {
  color: #dc3545;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.form-helper {
  color: #6c757d;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.alert {
  padding: 0.75rem 1rem;
  border-radius: 4px;
  margin-bottom: 1rem;
}

.alert-danger {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 2rem;
}

.btn {
  padding: 0.625rem 1.25rem;
  border: none;
  border-radius: 4px;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-primary {
  background-color: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: #0056b3;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background-color: #545b62;
}

.btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

@media (max-width: 640px) {
  .modal-container {
    max-width: 100%;
    margin: 1rem;
  }
  
  .modal-body {
    padding: 1rem;
  }
}
</style>