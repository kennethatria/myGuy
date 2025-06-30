<template>
  <div class="review-form">
    <h3 class="mb-4">Leave a Review</h3>
    
    <div v-if="error" class="alert alert-danger">
      {{ error }}
    </div>

    <form @submit.prevent="handleSubmit">
      <div class="form-group">
        <label class="form-label">Rating</label>
        <div class="rating-selector">
          <button
            v-for="star in 5"
            :key="star"
            type="button"
            @click="rating = star"
            :class="['star-btn', { active: star <= rating }]"
            :aria-label="`Rate ${star} stars`"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
            </svg>
          </button>
        </div>
        <div v-if="formErrors.rating" class="invalid-feedback d-block">{{ formErrors.rating }}</div>
      </div>

      <div class="form-group">
        <label for="comment" class="form-label">Comment</label>
        <textarea
          id="comment"
          v-model="comment"
          class="form-input"
          :class="{ 'is-invalid': formErrors.comment }"
          rows="4"
          placeholder="Share your experience working on this task..."
          required
        ></textarea>
        <div v-if="formErrors.comment" class="invalid-feedback">{{ formErrors.comment }}</div>
      </div>

      <div class="d-flex gap-2">
        <button 
          type="submit" 
          class="btn btn-primary"
          :disabled="loading"
        >
          {{ loading ? 'Submitting...' : 'Submit Review' }}
        </button>
        <button 
          type="button" 
          class="btn btn-secondary"
          @click="$emit('cancel')"
        >
          Cancel
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useReviewsStore } from '@/stores/reviews'

interface Props {
  taskId: number
  reviewedUserId?: number
}

interface FormErrors {
  rating?: string
  comment?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'review-submitted': [review: any]
  'cancel': []
}>()

const reviewsStore = useReviewsStore()

const rating = ref(0)
const comment = ref('')
const loading = ref(false)
const error = ref('')
const formErrors = reactive<FormErrors>({})

const validateForm = (): boolean => {
  formErrors.rating = undefined
  formErrors.comment = undefined
  
  let isValid = true
  
  if (rating.value < 1 || rating.value > 5) {
    formErrors.rating = 'Please select a rating between 1 and 5 stars'
    isValid = false
  }
  
  if (!comment.value.trim()) {
    formErrors.comment = 'Please provide a comment'
    isValid = false
  }
  
  return isValid
}

const handleSubmit = async () => {
  error.value = ''
  
  if (!validateForm()) {
    return
  }
  
  loading.value = true
  
  try {
    const review = await reviewsStore.createReview(props.taskId, {
      rating: rating.value,
      comment: comment.value.trim(),
      reviewedUserId: props.reviewedUserId
    })
    
    emit('review-submitted', review)
    
    // Reset form
    rating.value = 0
    comment.value = ''
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to submit review'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.review-form {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.rating-selector {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.star-btn {
  background: none;
  border: none;
  padding: 0.25rem;
  cursor: pointer;
  color: #ddd;
  transition: color 0.2s;
}

.star-btn:hover {
  color: #ffd700;
}

.star-btn.active {
  color: #ffd700;
}

.star-btn svg {
  width: 32px;
  height: 32px;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
}

.form-input:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.1);
}

.form-input.is-invalid {
  border-color: #dc3545;
}

.invalid-feedback {
  color: #dc3545;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-primary {
  background-color: #007bff;
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

.d-flex {
  display: flex;
}

.gap-2 {
  gap: 0.5rem;
}
</style>