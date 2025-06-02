<template>
  <div class="store-item-container">
    <div v-if="loading" class="loading">Loading...</div>
    
    <div v-else-if="error" class="error">
      {{ error }}
    </div>
    
    <div v-else-if="item" class="item-details">
      <div class="item-header">
        <router-link to="/store" class="back-link">
          <i class="fas fa-arrow-left"></i> Back to Store
        </router-link>
      </div>
      
      <div class="item-content">
        <div class="item-image-section">
          <img :src="item.image_url || '/placeholder.png'" :alt="item.title" />
        </div>
        
        <div class="item-info-section">
          <h1>{{ item.title }}</h1>
          
          <div class="item-meta">
            <span class="category">{{ item.category }}</span>
            <span class="condition">{{ item.condition }}</span>
            <span class="posted-date">Posted {{ formatDate(item.created_at) }}</span>
          </div>
          
          <div class="item-description">
            <h3>Description</h3>
            <p>{{ item.description }}</p>
          </div>
          
          <div class="seller-info">
            <h3>Seller</h3>
            <div class="seller-details">
              <span class="seller-name">{{ item.seller.full_name }}</span>
              <router-link 
                :to="{ name: 'user-profile', params: { id: item.seller.id } }"
                class="view-profile"
              >
                View Profile
              </router-link>
            </div>
          </div>
          
          <div class="price-section">
            <div v-if="item.is_auction" class="auction-info">
              <h3>Auction Details</h3>
              <p class="current-bid">Current Bid: ${{ item.current_bid || item.starting_bid }}</p>
              <p class="bid-increment">Minimum Increment: ${{ item.bid_increment }}</p>
              <p class="bid-count">{{ item.bid_count || 0 }} bids</p>
              
              <div v-if="item.seller.id !== userId" class="bid-form">
                <input 
                  v-model="bidAmount" 
                  type="number" 
                  :min="minBidAmount" 
                  :step="item.bid_increment"
                  placeholder="Enter bid amount"
                />
                <button @click="placeBid" class="btn btn-primary">Place Bid</button>
              </div>
            </div>
            
            <div v-else class="fixed-price">
              <h3>Price</h3>
              <p class="price">${{ item.price }}</p>
              
              <button 
                v-if="item.seller.id !== userId && item.status === 'available'"
                @click="purchaseItem" 
                class="btn btn-primary btn-large"
              >
                Purchase Item
              </button>
            </div>
          </div>
          
          <div v-if="item.status !== 'available'" class="item-status">
            <p class="status-message">This item is {{ item.status }}</p>
          </div>
        </div>
      </div>
      
      <!-- Bid History -->
      <div v-if="item.is_auction && bids.length > 0" class="bid-history">
        <h3>Bid History</h3>
        <div class="bid-list">
          <div v-for="bid in bids" :key="bid.id" class="bid-item">
            <span class="bidder">{{ bid.bidder.full_name }}</span>
            <span class="bid-amount">${{ bid.amount }}</span>
            <span class="bid-time">{{ formatDate(bid.created_at) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const item = ref(null);
const bids = ref([]);
const loading = ref(true);
const error = ref('');
const bidAmount = ref('');

const userId = computed(() => authStore.user?.id);
const itemId = computed(() => route.params.id);

const minBidAmount = computed(() => {
  if (!item.value?.is_auction) return 0;
  const currentBid = item.value.current_bid || item.value.starting_bid;
  return currentBid + item.value.bid_increment;
});

async function loadItem() {
  try {
    loading.value = true;
    const response = await fetch(`http://localhost:8081/api/v1/store/items/${itemId.value}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    });
    
    if (!response.ok) {
      throw new Error('Failed to load item');
    }
    
    item.value = await response.json();
    
    if (item.value.is_auction) {
      await loadBids();
    }
  } catch (err) {
    error.value = err.message;
  } finally {
    loading.value = false;
  }
}

async function loadBids() {
  try {
    const response = await fetch(`http://localhost:8081/api/v1/store/items/${itemId.value}/bids`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    });
    
    if (response.ok) {
      bids.value = await response.json();
    }
  } catch (err) {
    console.error('Error loading bids:', err);
  }
}

async function placeBid() {
  if (!bidAmount.value || parseFloat(bidAmount.value) < minBidAmount.value) {
    alert(`Minimum bid amount is $${minBidAmount.value}`);
    return;
  }
  
  try {
    const response = await fetch(`http://localhost:8081/api/v1/store/items/${itemId.value}/bids`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ amount: parseFloat(bidAmount.value) })
    });
    
    if (response.ok) {
      await loadItem();
      await loadBids();
      bidAmount.value = '';
    } else {
      const error = await response.json();
      alert(error.error || 'Failed to place bid');
    }
  } catch (err) {
    alert('Error placing bid');
  }
}

async function purchaseItem() {
  if (confirm(`Are you sure you want to purchase this item for $${item.value.price}?`)) {
    try {
      const response = await fetch(`http://localhost:8081/api/v1/store/items/${itemId.value}/purchase`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (response.ok) {
        alert('Item purchased successfully!');
        await loadItem();
      } else {
        const error = await response.json();
        alert(error.error || 'Failed to purchase item');
      }
    } catch (err) {
      alert('Error purchasing item');
    }
  }
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  if (diff < 86400000) { // Less than 24 hours
    const hours = Math.floor(diff / 3600000);
    if (hours < 1) return 'just now';
    return `${hours}h ago`;
  }
  
  if (diff < 604800000) { // Less than 7 days
    const days = Math.floor(diff / 86400000);
    return `${days}d ago`;
  }
  
  return date.toLocaleDateString();
}

onMounted(() => {
  loadItem();
});
</script>

<style scoped>
.store-item-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.loading, .error {
  text-align: center;
  padding: 4rem;
  font-size: 1.125rem;
  color: #6b7280;
}

.error {
  color: #ef4444;
}

.item-header {
  margin-bottom: 2rem;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  color: #4F46E5;
  text-decoration: none;
  font-weight: 500;
}

.back-link:hover {
  color: #4338CA;
}

.item-content {
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
}

.item-image-section {
  background: #f3f4f6;
  min-height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-image-section img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.item-info-section {
  padding: 2rem;
}

.item-info-section h1 {
  font-size: 2rem;
  font-weight: 600;
  margin-bottom: 1rem;
}

.item-meta {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.category, .condition, .posted-date {
  padding: 0.25rem 0.75rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  background: #f3f4f6;
  color: #374151;
}

.item-description {
  margin-bottom: 2rem;
}

.item-description h3,
.seller-info h3,
.price-section h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #111827;
}

.item-description p {
  color: #6b7280;
  line-height: 1.6;
}

.seller-info {
  margin-bottom: 2rem;
  padding-bottom: 2rem;
  border-bottom: 1px solid #e5e7eb;
}

.seller-details {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.seller-name {
  font-weight: 500;
  color: #374151;
}

.view-profile {
  color: #4F46E5;
  text-decoration: none;
  font-size: 0.875rem;
}

.view-profile:hover {
  color: #4338CA;
}

.price-section {
  margin-bottom: 2rem;
}

.auction-info p {
  margin-bottom: 0.5rem;
  color: #374151;
}

.current-bid {
  font-size: 1.5rem;
  font-weight: 600;
  color: #059669;
}

.bid-form {
  display: flex;
  gap: 1rem;
  margin-top: 1rem;
}

.bid-form input {
  flex: 1;
  padding: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  font-size: 1rem;
}

.fixed-price .price {
  font-size: 2rem;
  font-weight: 600;
  color: #4F46E5;
  margin-bottom: 1rem;
}

.btn {
  padding: 0.75rem 1.5rem;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
  font-size: 1rem;
}

.btn-primary {
  background: #4F46E5;
  color: white;
}

.btn-primary:hover {
  background: #4338CA;
}

.btn-large {
  padding: 1rem 2rem;
  font-size: 1.125rem;
}

.item-status {
  background: #fef3c7;
  padding: 1rem;
  border-radius: 0.375rem;
  text-align: center;
}

.status-message {
  color: #92400e;
  font-weight: 500;
}

.bid-history {
  margin-top: 2rem;
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  padding: 2rem;
}

.bid-history h3 {
  margin-bottom: 1rem;
}

.bid-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.bid-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  background: #f9fafb;
  border-radius: 0.375rem;
}

.bidder {
  font-weight: 500;
  color: #374151;
}

.bid-amount {
  font-weight: 600;
  color: #059669;
}

.bid-time {
  font-size: 0.875rem;
  color: #6b7280;
}

@media (max-width: 768px) {
  .item-content {
    grid-template-columns: 1fr;
  }
  
  .bid-form {
    flex-direction: column;
  }
}
</style>