<template>
  <div class="store-container">
    <div class="store-header">
      <h1>Marketplace</h1>
      <div class="header-actions">
        <button @click="showCreateModal = true" class="btn btn-primary">
          <i class="fas fa-plus"></i> List Item
        </button>
      </div>
    </div>

    <div class="filters-section">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search items..."
        class="search-input"
        @input="searchItems"
      />
      <select v-model="categoryFilter" @change="filterItems" class="filter-select">
        <option value="">All Categories</option>
        <option value="electronics">Electronics</option>
        <option value="furniture">Furniture</option>
        <option value="clothing">Clothing</option>
        <option value="books">Books</option>
        <option value="tools">Tools</option>
        <option value="sports">Sports</option>
        <option value="other">Other</option>
      </select>
      <select v-model="conditionFilter" @change="filterItems" class="filter-select">
        <option value="">All Conditions</option>
        <option value="new">New</option>
        <option value="like_new">Like New</option>
        <option value="good">Good</option>
        <option value="fair">Fair</option>
        <option value="poor">Poor</option>
      </select>
    </div>

    <div class="items-grid">
      <div v-for="item in filteredItems" :key="item.id" class="item-card">
        <div class="item-image">
          <img :src="item.image_url || '/placeholder.png'" :alt="item.title" />
        </div>
        <div class="item-content">
          <h3>{{ item.title }}</h3>
          <p class="item-description">{{ item.description }}</p>
          <div class="item-meta">
            <span class="category">{{ item.category }}</span>
            <span class="condition">{{ item.condition }}</span>
          </div>
          <div class="item-price">
            <span v-if="item.is_auction" class="auction-label">
              Current Bid: ${{ item.current_bid || item.starting_bid }}
            </span>
            <span v-else>${{ item.price }}</span>
          </div>
          <div class="item-actions">
            <button @click="viewItem(item)" class="btn btn-sm btn-outline">
              View Details
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Item Modal -->
    <div v-if="showCreateModal" class="modal-overlay" @click="showCreateModal = false">
      <div class="modal-content" @click.stop>
        <h2>List New Item</h2>
        <form @submit.prevent="createItem">
          <div class="form-group">
            <label>Title</label>
            <input v-model="newItem.title" type="text" required />
          </div>
          <div class="form-group">
            <label>Description</label>
            <textarea v-model="newItem.description" rows="4" required></textarea>
          </div>
          <div class="form-group">
            <label>Category</label>
            <select v-model="newItem.category" required>
              <option value="">Select Category</option>
              <option value="electronics">Electronics</option>
              <option value="furniture">Furniture</option>
              <option value="clothing">Clothing</option>
              <option value="books">Books</option>
              <option value="tools">Tools</option>
              <option value="sports">Sports</option>
              <option value="other">Other</option>
            </select>
          </div>
          <div class="form-group">
            <label>Condition</label>
            <select v-model="newItem.condition" required>
              <option value="">Select Condition</option>
              <option value="new">New</option>
              <option value="like_new">Like New</option>
              <option value="good">Good</option>
              <option value="fair">Fair</option>
              <option value="poor">Poor</option>
            </select>
          </div>
          <div class="form-group">
            <label>
              <input type="checkbox" v-model="newItem.is_auction" />
              List as Auction
            </label>
          </div>
          <div v-if="!newItem.is_auction" class="form-group">
            <label>Price</label>
            <input v-model="newItem.price" type="number" step="0.01" min="0" required />
          </div>
          <div v-else>
            <div class="form-group">
              <label>Starting Bid</label>
              <input v-model="newItem.starting_bid" type="number" step="0.01" min="0" required />
            </div>
            <div class="form-group">
              <label>Bid Increment</label>
              <input v-model="newItem.bid_increment" type="number" step="0.01" min="0.01" required />
            </div>
          </div>
          <div class="modal-actions">
            <button type="button" @click="showCreateModal = false" class="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">List Item</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();

const items = ref([]);
const searchQuery = ref('');
const categoryFilter = ref('');
const conditionFilter = ref('');
const showCreateModal = ref(false);
const newItem = ref({
  title: '',
  description: '',
  category: '',
  condition: '',
  price: 0,
  is_auction: false,
  starting_bid: 0,
  bid_increment: 0.01
});

const filteredItems = computed(() => {
  return items.value.filter(item => {
    const matchesSearch = !searchQuery.value || 
      item.title.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      item.description.toLowerCase().includes(searchQuery.value.toLowerCase());
    
    const matchesCategory = !categoryFilter.value || item.category === categoryFilter.value;
    const matchesCondition = !conditionFilter.value || item.condition === conditionFilter.value;
    
    return matchesSearch && matchesCategory && matchesCondition;
  });
});

async function loadItems() {
  try {
    const response = await fetch('http://localhost:8081/api/v1/store/items', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    });
    if (response.ok) {
      items.value = await response.json();
    }
  } catch (error) {
    console.error('Error loading items:', error);
  }
}

async function createItem() {
  try {
    const response = await fetch('http://localhost:8081/api/v1/store/items', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify(newItem.value)
    });
    
    if (response.ok) {
      showCreateModal.value = false;
      await loadItems();
      // Reset form
      newItem.value = {
        title: '',
        description: '',
        category: '',
        condition: '',
        price: 0,
        is_auction: false,
        starting_bid: 0,
        bid_increment: 0.01
      };
    }
  } catch (error) {
    console.error('Error creating item:', error);
  }
}

function viewItem(item) {
  router.push({ name: 'store-item', params: { id: item.id } });
}

function searchItems() {
  // Debounce search if needed
}

function filterItems() {
  // Additional filter logic if needed
}

onMounted(() => {
  loadItems();
});
</script>

<style scoped>
.store-container {
  padding: 2rem;
}

.store-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.store-header h1 {
  font-size: 2rem;
  font-weight: 600;
  color: #111827;
}

.filters-section {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
}

.search-input {
  flex: 1;
  padding: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  font-size: 1rem;
}

.filter-select {
  padding: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  background-color: white;
}

.items-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

.item-card {
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  transition: transform 0.2s;
}

.item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.item-image {
  height: 200px;
  background: #f3f4f6;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.item-content {
  padding: 1.5rem;
}

.item-content h3 {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.item-description {
  color: #6b7280;
  margin-bottom: 1rem;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.item-meta {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.category, .condition {
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  background: #f3f4f6;
  color: #374151;
}

.item-price {
  font-size: 1.5rem;
  font-weight: 600;
  color: #4F46E5;
  margin-bottom: 1rem;
}

.auction-label {
  font-size: 1rem;
  color: #059669;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  padding: 2rem;
  border-radius: 0.5rem;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content h2 {
  margin-bottom: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-group input,
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
}

.modal-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: #4F46E5;
  color: white;
  border: none;
}

.btn-primary:hover {
  background: #4338CA;
}

.btn-secondary {
  background: #6b7280;
  color: white;
  border: none;
}

.btn-outline {
  background: white;
  border: 1px solid #e5e7eb;
  color: #374151;
}

.btn-outline:hover {
  background: #f9fafb;
}
</style>