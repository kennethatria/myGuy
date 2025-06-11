<template>
  <div class="container py-4">
    <div class="flex justify-between items-center mb-4">
      <div>
        <h1 class="text-2xl font-semibold">Browse Available Gigs</h1>
        <p class="text-muted mt-1">Find and apply for gigs posted by other users</p>
      </div>
      <div class="d-flex gap-2">
        <button
          @click="showFilters = !showFilters"
          class="btn btn-outline-secondary"
        >
          <i class="bi" :class="showFilters ? 'bi-funnel-fill' : 'bi-funnel'"></i>
          Filters
          <span v-if="hasActiveFilters" class="badge bg-primary ms-1">{{ activeFilterCount }}</span>
        </button>
        <router-link
          :to="{ name: 'create-task' }"
          class="btn btn-primary"
        >
          Create Gig
        </router-link>
      </div>
    </div>

    <!-- Search Bar (Always Visible) -->
    <div class="mb-4">
      <input
        v-model="searchQuery"
        type="text"
        class="form-control"
        placeholder="Search gigs by title or description..."
        @input="debouncedSearch"
      />
    </div>

    <!-- Collapsible Filters Section -->
    <transition name="slide-fade">
      <div v-if="showFilters" class="card mb-4">
        <div class="card-body">
          <!-- Filter Controls -->
          <div class="row g-3">
          <!-- Status Filter -->
          <div class="col-md-3">
            <label class="form-label">Status</label>
            <select v-model="filters.status" class="form-select">
              <option value="">All Statuses</option>
              <option value="open">Open</option>
              <option value="in_progress">In Progress</option>
              <option value="completed">Completed</option>
            </select>
          </div>

          <!-- Price Range -->
          <div class="col-md-3">
            <label class="form-label">Min Fee ($)</label>
            <input
              v-model.number="filters.minFee"
              type="number"
              min="0"
              class="form-control"
              placeholder="0"
            />
          </div>
          <div class="col-md-3">
            <label class="form-label">Max Fee ($)</label>
            <input
              v-model.number="filters.maxFee"
              type="number"
              min="0"
              class="form-control"
              placeholder="Any"
            />
          </div>

          <!-- Sort By -->
          <div class="col-md-3">
            <label class="form-label">Sort By</label>
            <select v-model="sortBy" class="form-select">
              <option value="created_at">Newest First</option>
              <option value="deadline">Deadline</option>
              <option value="fee">Fee Amount</option>
            </select>
          </div>
        </div>

          <div class="mt-3 flex gap-2">
            <button @click="applyFilters" class="btn btn-primary btn-sm">
              Apply Filters
            </button>
            <button @click="resetFilters" class="btn btn-secondary btn-sm">
              Reset
            </button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="alert alert-danger">
      {{ error }}
      <button @click="fetchTasks" class="btn btn-sm btn-outline-danger ms-3">
        Retry
      </button>
    </div>

    <!-- Results Info -->
    <div v-else-if="paginatedResult" class="mb-3">
      <p class="text-muted">
        Showing {{ (currentPage - 1) * perPage + 1 }} - 
        {{ Math.min(currentPage * perPage, paginatedResult.total) }} 
        of {{ paginatedResult.total }} gigs
      </p>
    </div>

    <!-- Task List -->
    <div v-if="paginatedResult && paginatedResult.tasks.length > 0" class="space-y-3">
      <div
        v-for="task in paginatedResult.tasks"
        :key="task.id"
        class="card hover-shadow"
      >
        <div class="card-body">
          <div class="d-flex justify-content-between align-items-start">
            <div class="flex-grow-1">
              <h3 class="h5 mb-2">
                <router-link
                  :to="{ name: 'task-detail', params: { id: task.id } }"
                  class="text-decoration-none"
                >
                  {{ task.title }}
                </router-link>
              </h3>
              <p class="text-muted mb-2">{{ task.description }}</p>
              
              <div class="d-flex align-items-center gap-3 text-sm text-muted">
                <span>
                  <i class="bi bi-person"></i>
                  {{ task.creator?.username || 'Unknown' }}
                </span>
                <span>
                  <i class="bi bi-calendar"></i>
                  {{ formatDate(task.deadline) }}
                </span>
                <span v-if="task.fee" class="text-success fw-bold">
                  <i class="bi bi-currency-dollar"></i>
                  ${{ task.fee }}
                </span>
              </div>
            </div>
            
            <div class="ms-3">
              <span :class="['badge', statusBadgeClass(task.status)]">
                {{ formatStatus(task.status) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="!loading && paginatedResult?.tasks.length === 0" class="card">
      <div class="card-body text-center py-5">
        <i class="bi bi-inbox display-1 text-muted"></i>
        <h3 class="mt-3">No gigs found</h3>
        <p class="text-muted">
          {{ searchQuery || hasActiveFilters ? 'Try adjusting your search or filters' : 'Be the first to create a gig!' }}
        </p>
        <div class="mt-4">
          <button v-if="hasActiveFilters" @click="resetFilters" class="btn btn-secondary me-2">
            Clear Filters
          </button>
          <router-link :to="{ name: 'create-task' }" class="btn btn-primary">
            Create a Gig
          </router-link>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <nav v-if="paginatedResult && paginatedResult.total_pages > 1" class="mt-4">
      <ul class="pagination justify-content-center">
        <li class="page-item" :class="{ disabled: currentPage === 1 }">
          <button class="page-link" @click="goToPage(currentPage - 1)" :disabled="currentPage === 1">
            Previous
          </button>
        </li>
        
        <li
          v-for="page in visiblePages"
          :key="page"
          class="page-item"
          :class="{ active: page === currentPage }"
        >
          <button class="page-link" @click="goToPage(page)">
            {{ page }}
          </button>
        </li>
        
        <li class="page-item" :class="{ disabled: currentPage === paginatedResult.total_pages }">
          <button
            class="page-link"
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === paginatedResult.total_pages"
          >
            Next
          </button>
        </li>
      </ul>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { format } from 'date-fns'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'
import { debounce } from 'lodash-es'

interface Task {
  id: number
  title: string
  description: string
  status: string
  deadline: string
  fee?: number
  creator?: {
    id: number
    username: string
  }
}

interface PaginatedResult {
  tasks: Task[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

const tasksStore = useTasksStore()
const authStore = useAuthStore()

// State
const loading = ref(false)
const error = ref('')
const paginatedResult = ref<PaginatedResult | null>(null)
const searchQuery = ref('')
const currentPage = ref(1)
const perPage = ref(10)
const sortBy = ref('created_at')
const sortOrder = ref('desc')
const showFilters = ref(false)

// Filters
const filters = ref({
  status: '',
  minFee: null as number | null,
  maxFee: null as number | null,
})

// Computed
const hasActiveFilters = computed(() => {
  return filters.value.status || 
         filters.value.minFee !== null || filters.value.maxFee !== null
})

const activeFilterCount = computed(() => {
  let count = 0
  if (filters.value.status) count++
  if (filters.value.minFee !== null) count++
  if (filters.value.maxFee !== null) count++
  return count
})

const visiblePages = computed(() => {
  if (!paginatedResult.value) return []
  
  const total = paginatedResult.value.total_pages
  const current = currentPage.value
  const delta = 2
  const range = []
  const rangeWithDots = []
  let l

  for (let i = 1; i <= total; i++) {
    if (i === 1 || i === total || (i >= current - delta && i <= current + delta)) {
      range.push(i)
    }
  }

  range.forEach((i) => {
    if (l) {
      if (i - l === 2) {
        rangeWithDots.push(l + 1)
      } else if (i - l !== 1) {
        rangeWithDots.push('...')
      }
    }
    rangeWithDots.push(i)
    l = i
  })

  return rangeWithDots
})

// Methods
const formatDate = (date: string) => {
  return format(new Date(date), 'MMM d, yyyy')
}

const formatStatus = (status: string) => {
  return status.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())
}

const statusBadgeClass = (status: string) => {
  const classes: Record<string, string> = {
    open: 'bg-success',
    in_progress: 'bg-warning',
    completed: 'bg-secondary'
  }
  return classes[status] || 'bg-secondary'
}

const buildQueryParams = () => {
  const params = new URLSearchParams()
  
  // Always exclude current user's tasks
  if (authStore.user?.id) {
    params.append('exclude_created_by', String(authStore.user.id))
  }
  
  // Search
  if (searchQuery.value) {
    params.append('search', searchQuery.value)
  }
  
  // Filters
  if (filters.value.status) {
    params.append('status', filters.value.status)
  }
  if (filters.value.minFee !== null) {
    params.append('min_fee', String(filters.value.minFee))
  }
  if (filters.value.maxFee !== null) {
    params.append('max_fee', String(filters.value.maxFee))
  }
  
  // Sorting
  params.append('sort_by', sortBy.value)
  params.append('sort_order', sortOrder.value)
  
  // Pagination
  params.append('page', String(currentPage.value))
  params.append('per_page', String(perPage.value))
  
  return params
}

const fetchTasks = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const response = await fetch(`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/api/v1/tasks?${buildQueryParams()}`, {
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw new Error('Failed to fetch tasks')
    }
    
    paginatedResult.value = await response.json()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load tasks'
    console.error('Error fetching tasks:', err)
  } finally {
    loading.value = false
  }
}

const debouncedSearch = debounce(() => {
  currentPage.value = 1
  fetchTasks()
}, 300)

const applyFilters = () => {
  currentPage.value = 1
  fetchTasks()
}

const resetFilters = () => {
  searchQuery.value = ''
  filters.value = {
    status: '',
    minFee: null,
    maxFee: null,
  }
  sortBy.value = 'created_at'
  sortOrder.value = 'desc'
  currentPage.value = 1
  fetchTasks()
}

const goToPage = (page: number) => {
  if (page < 1 || (paginatedResult.value && page > paginatedResult.value.total_pages)) return
  currentPage.value = page
  fetchTasks()
}

// Watch for sort changes
watch([sortBy, sortOrder], () => {
  currentPage.value = 1
  fetchTasks()
})

onMounted(() => {
  fetchTasks()
})
</script>

<style scoped>
.container {
  max-width: 1200px;
  margin: 0 auto;
}

.hover-shadow {
  transition: box-shadow 0.2s ease;
}

.hover-shadow:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.space-y-3 > * + * {
  margin-top: 1rem;
}

.form-control,
.form-select {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #ced4da;
  border-radius: 0.25rem;
  font-size: 1rem;
}

.form-control:focus,
.form-select:focus {
  border-color: #86b7fe;
  outline: 0;
  box-shadow: 0 0 0 0.25rem rgba(13, 110, 253, 0.25);
}

.btn {
  display: inline-block;
  font-weight: 400;
  text-align: center;
  vertical-align: middle;
  user-select: none;
  padding: 0.375rem 0.75rem;
  font-size: 1rem;
  line-height: 1.5;
  border-radius: 0.25rem;
  transition: color 0.15s ease-in-out, background-color 0.15s ease-in-out;
  text-decoration: none;
  border: 1px solid transparent;
  cursor: pointer;
}

.btn-primary {
  color: #fff;
  background-color: #0d6efd;
  border-color: #0d6efd;
}

.btn-outline-secondary {
  color: #6c757d;
  border-color: #6c757d;
  background-color: transparent;
}

.btn-outline-secondary:hover {
  color: #fff;
  background-color: #6c757d;
  border-color: #6c757d;
}

/* Transition for filter dropdown */
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.3s ease-in;
}

.slide-fade-enter-from {
  transform: translateY(-10px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateY(-10px);
  opacity: 0;
}

.btn-primary:hover {
  color: #fff;
  background-color: #0b5ed7;
  border-color: #0a58ca;
}

.btn-secondary {
  color: #fff;
  background-color: #6c757d;
  border-color: #6c757d;
}

.btn-secondary:hover {
  color: #fff;
  background-color: #5c636a;
  border-color: #565e64;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.875rem;
}

.badge {
  display: inline-block;
  padding: 0.35em 0.65em;
  font-size: 0.75em;
  font-weight: 700;
  line-height: 1;
  color: #fff;
  text-align: center;
  white-space: nowrap;
  vertical-align: baseline;
  border-radius: 0.25rem;
}

.bg-success {
  background-color: #198754;
}

.bg-warning {
  background-color: #ffc107;
  color: #000;
}

.bg-secondary {
  background-color: #6c757d;
}

.spinner-border {
  display: inline-block;
  width: 2rem;
  height: 2rem;
  vertical-align: text-bottom;
  border: 0.25em solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spinner-border 0.75s linear infinite;
}

@keyframes spinner-border {
  to { transform: rotate(360deg); }
}

.pagination {
  display: flex;
  padding-left: 0;
  list-style: none;
}

.page-item:not(:first-child) .page-link {
  margin-left: -1px;
}

.page-link {
  position: relative;
  display: block;
  padding: 0.375rem 0.75rem;
  color: #0d6efd;
  text-decoration: none;
  background-color: #fff;
  border: 1px solid #dee2e6;
}

.page-link:hover {
  z-index: 2;
  color: #0a58ca;
  background-color: #e9ecef;
  border-color: #dee2e6;
}

.page-item.active .page-link {
  z-index: 3;
  color: #fff;
  background-color: #0d6efd;
  border-color: #0d6efd;
}

.page-item.disabled .page-link {
  color: #6c757d;
  pointer-events: none;
  background-color: #fff;
  border-color: #dee2e6;
}

.card {
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.card-body {
  padding: 1.25rem;
}

.alert {
  padding: 0.75rem 1.25rem;
  border: 1px solid transparent;
  border-radius: 0.25rem;
}

.alert-danger {
  color: #842029;
  background-color: #f8d7da;
  border-color: #f5c2c7;
}

.text-muted {
  color: #6c757d;
}

.text-decoration-none {
  text-decoration: none;
}

.flex {
  display: flex;
}

.justify-between {
  justify-content: space-between;
}

.items-center {
  align-items: center;
}

.gap-2 {
  gap: 0.5rem;
}

.gap-3 {
  gap: 1rem;
}

.mb-2 {
  margin-bottom: 0.5rem;
}

.mb-3 {
  margin-bottom: 1rem;
}

.mb-4 {
  margin-bottom: 1.5rem;
}

.mt-1 {
  margin-top: 0.25rem;
}

.mt-3 {
  margin-top: 1rem;
}

.mt-4 {
  margin-top: 1.5rem;
}

.ms-3 {
  margin-left: 1rem;
}

.me-2 {
  margin-right: 0.5rem;
}

.py-4 {
  padding-top: 1.5rem;
  padding-bottom: 1.5rem;
}

.py-5 {
  padding-top: 3rem;
  padding-bottom: 3rem;
}

.text-center {
  text-align: center;
}

.fw-bold {
  font-weight: 700;
}

.row {
  display: flex;
  flex-wrap: wrap;
  margin-right: -0.5rem;
  margin-left: -0.5rem;
}

.g-3 {
  gap: 1rem;
}

.col-md-3 {
  flex: 0 0 auto;
  width: 25%;
  padding-right: 0.5rem;
  padding-left: 0.5rem;
}

@media (max-width: 768px) {
  .col-md-3 {
    width: 100%;
    margin-bottom: 1rem;
  }
}

.form-label {
  display: inline-block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.d-flex {
  display: flex;
}

.justify-content-between {
  justify-content: space-between;
}

.align-items-start {
  align-items: flex-start;
}

.align-items-center {
  align-items: center;
}

.flex-grow-1 {
  flex-grow: 1;
}

.h5 {
  font-size: 1.25rem;
}

.visually-hidden {
  position: absolute !important;
  width: 1px !important;
  height: 1px !important;
  padding: 0 !important;
  margin: -1px !important;
  overflow: hidden !important;
  clip: rect(0, 0, 0, 0) !important;
  white-space: nowrap !important;
  border: 0 !important;
}

.text-sm {
  font-size: 0.875rem;
}

.display-1 {
  font-size: 4rem;
}

.text-success {
  color: #198754;
}

.bi {
  display: inline-block;
  vertical-align: -0.125em;
  margin-right: 0.25rem;
}
</style>