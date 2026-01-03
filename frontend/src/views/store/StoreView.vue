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

    <div v-if="isLoading" class="loading-state">
      <div class="loading-spinner">Loading items...</div>
    </div>

    <div v-else>
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

      <div v-if="filteredItems.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none">
          <path d="M3 9V21H21V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M3 9H21L19 3H5L3 9Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M12 3V9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
      <h3>No items found</h3>
      <p>{{ searchQuery || categoryFilter || conditionFilter ? 'Try adjusting your filters to see more items.' : 'Be the first to list an item in the marketplace!' }}</p>
      <button @click="showCreateModal = true" class="btn btn-primary">List Your First Item</button>
    </div>

    <div v-else class="items-grid">
      <div v-for="item in filteredItems" :key="item.id || item.title || Math.random()" class="item-card">
        <div class="item-image">
          <img 
            v-if="item.images && item.images.length > 0" 
            :src="config.STORE_API_BASE_URL + item.images[0].url" 
            :alt="item.title || 'Item image'"
            @error="handleImageError"
          />
          <div v-else class="image-placeholder">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21,15 16,10 5,21"/>
            </svg>
            <span>No Image</span>
          </div>
        </div>
        <div class="item-content">
          <h3>{{ item.title || 'Untitled Item' }}</h3>
          <p class="item-description">{{ item.description || 'No description available' }}</p>
          <div class="item-meta">
            <span class="category">{{ item.category || 'Uncategorized' }}</span>
            <span class="condition">{{ item.condition || 'Unknown' }}</span>
          </div>
          <div class="item-price">
            <span v-if="item.is_auction" class="auction-label">
              Current Bid: UGX {{ formatCurrency(item.current_bid || item.starting_bid || 0) }}
            </span>
            <span v-else>UGX {{ formatCurrency(item.price || 0) }}</span>
          </div>
          <div class="item-actions">
            <button @click="viewItem(item)" class="btn btn-sm btn-outline">
              View Details
            </button>
          </div>
        </div>
      </div>
    </div>
    </div>

    <!-- Create Item Modal -->
    <div v-if="showCreateModal" class="modal-overlay" @click="cancelCreateItem">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>List New Item</h2>
          <button type="button" @click="cancelCreateItem" class="close-btn">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div class="form-progress">
          <div class="progress-steps">
            <div class="step" :class="{ active: currentStep >= 1, completed: currentStep > 1 }">
              <span class="step-number">1</span>
              <span class="step-label">Basic Info</span>
            </div>
            <div class="step" :class="{ active: currentStep >= 2, completed: currentStep > 2 }">
              <span class="step-number">2</span>
              <span class="step-label">Details</span>
            </div>
            <div class="step" :class="{ active: currentStep >= 3 }">
              <span class="step-number">3</span>
              <span class="step-label">Pricing</span>
            </div>
          </div>
        </div>

        <form @submit.prevent="createItem" class="improved-form">
          <!-- Step 1: Basic Information -->
          <div v-if="currentStep === 1" class="form-step">
            <h3>Tell us about your item</h3>
            
            <div class="form-group">
              <label for="item-title" class="form-label">
                <span class="label-text">Item Title</span>
                <span class="required">*</span>
              </label>
              <input 
                id="item-title"
                v-model="newItem.title" 
                type="text" 
                class="form-input"
                :class="{ 'error': formErrors.title }"
                placeholder="e.g., iPhone 13 Pro Max 256GB"
                required 
                maxlength="100"
              />
              <div v-if="formErrors.title" class="error-message">{{ formErrors.title }}</div>
              <div class="character-count">{{ newItem.title.length }}/100</div>
            </div>

            <div class="form-group">
              <label for="item-description" class="form-label">
                <span class="label-text">Description</span>
                <span class="required">*</span>
              </label>
              <textarea 
                id="item-description"
                v-model="newItem.description" 
                class="form-input"
                :class="{ 'error': formErrors.description }"
                rows="4" 
                placeholder="Describe your item's condition, features, and any important details..."
                required
                maxlength="500"
              ></textarea>
              <div v-if="formErrors.description" class="error-message">{{ formErrors.description }}</div>
              <div class="character-count">{{ newItem.description.length }}/500</div>
            </div>

            <div class="form-group">
              <label class="form-label">
                <span class="label-text">Photos</span>
                <span class="optional">(Recommended)</span>
              </label>
              <div class="image-upload-section">
                <div class="image-previews">
                  <div v-for="(image, index) in selectedImages" :key="index" class="image-preview">
                    <img :src="image.preview" :alt="`Preview ${index + 1}`" />
                    <button type="button" @click="removeImage(index)" class="remove-image" title="Remove photo">×</button>
                  </div>
                  <div v-if="selectedImages.length < 3" class="image-upload-box">
                    <input
                      type="file"
                      id="image-upload"
                      accept="image/*"
                      multiple
                      @change="handleImageSelect"
                      :disabled="selectedImages.length >= 3"
                      style="display: none;"
                    />
                    <label for="image-upload" class="upload-label">
                      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 5v14M5 12h14"/>
                      </svg>
                      <span>Add Photo</span>
                    </label>
                  </div>
                </div>
                <p class="help-text">Add up to 3 photos. Good photos help your item sell faster! (JPG, PNG, GIF)</p>
              </div>
            </div>
          </div>

          <!-- Step 2: Item Details -->
          <div v-if="currentStep === 2" class="form-step">
            <h3>Item details</h3>
            
            <div class="form-row">
              <div class="form-group">
                <label for="item-category" class="form-label">
                  <span class="label-text">Category</span>
                  <span class="required">*</span>
                </label>
                <select id="item-category" v-model="newItem.category" class="form-input" required>
                  <option value="">Choose a category</option>
                  <option value="electronics">📱 Electronics</option>
                  <option value="furniture">🪑 Furniture</option>
                  <option value="clothing">👕 Clothing</option>
                  <option value="books">📚 Books</option>
                  <option value="tools">🔧 Tools</option>
                  <option value="sports">⚽ Sports & Recreation</option>
                  <option value="other">📦 Other</option>
                </select>
                <p class="help-text">Choose the category that best fits your item</p>
              </div>

              <div class="form-group">
                <label for="item-condition" class="form-label">
                  <span class="label-text">Condition</span>
                  <span class="required">*</span>
                </label>
                <select id="item-condition" v-model="newItem.condition" class="form-input" required>
                  <option value="">Select condition</option>
                  <option value="new">🆕 New - Never used</option>
                  <option value="like_new">✨ Like New - Barely used</option>
                  <option value="good">👍 Good - Some wear</option>
                  <option value="fair">👌 Fair - Well used</option>
                  <option value="poor">🔧 Poor - Needs repair</option>
                </select>
                <p class="help-text">Be honest about the condition to build trust</p>
              </div>
            </div>
          </div>

          <!-- Step 3: Pricing -->
          <div v-if="currentStep === 3" class="form-step">
            <h3>Set your price</h3>
            
            <div class="pricing-type-selector">
              <div class="pricing-option" :class="{ active: !newItem.is_auction }" @click="newItem.is_auction = false">
                <div class="option-icon">💰</div>
                <div class="option-content">
                  <h4>Fixed Price</h4>
                  <p>Sell at a set price</p>
                </div>
              </div>
              <div class="pricing-option" :class="{ active: newItem.is_auction }" @click="newItem.is_auction = true">
                <div class="option-icon">🏆</div>
                <div class="option-content">
                  <h4>Auction</h4>
                  <p>Let buyers compete</p>
                </div>
              </div>
            </div>

            <div v-if="!newItem.is_auction" class="form-group">
              <label for="item-price" class="form-label">
                <span class="label-text">Price (UGX)</span>
                <span class="required">*</span>
              </label>
              <div class="currency-input">
                <span class="currency-symbol">UGX</span>
                <input 
                  id="item-price"
                  v-model="newItem.price" 
                  type="number" 
                  class="form-input"
                  :class="{ 'error': formErrors.price }"
                  step="1000" 
                  min="0" 
                  placeholder="50000"
                  @input="sanitizeNumberInput('price', $event)"
                  required 
                />
              </div>
              <div v-if="formErrors.price" class="error-message">{{ formErrors.price }}</div>
              <p class="help-text">Research similar items to price competitively</p>
            </div>

            <div v-else class="auction-fields">
              <div class="form-group">
                <label for="starting-bid" class="form-label">
                  <span class="label-text">Starting Bid (UGX)</span>
                  <span class="required">*</span>
                </label>
                <div class="currency-input">
                  <span class="currency-symbol">UGX</span>
                  <input 
                    id="starting-bid"
                    v-model="newItem.starting_bid" 
                    type="number" 
                    class="form-input"
                    :class="{ 'error': formErrors.starting_bid }"
                    step="1000" 
                    min="0" 
                    placeholder="10000"
                    @input="sanitizeNumberInput('starting_bid', $event)"
                    required 
                  />
                </div>
                <div v-if="formErrors.starting_bid" class="error-message">{{ formErrors.starting_bid }}</div>
              </div>

              <div class="form-group">
                <label for="bid-increment" class="form-label">
                  <span class="label-text">Minimum Bid Increment (UGX)</span>
                  <span class="required">*</span>
                </label>
                <div class="currency-input">
                  <span class="currency-symbol">UGX</span>
                  <input 
                    id="bid-increment"
                    v-model="newItem.bid_increment" 
                    type="number" 
                    class="form-input"
                    :class="{ 'error': formErrors.bid_increment }"
                    step="500" 
                    min="500" 
                    placeholder="1000"
                    @input="sanitizeNumberInput('bid_increment', $event)"
                    required 
                  />
                </div>
                <div v-if="formErrors.bid_increment" class="error-message">{{ formErrors.bid_increment }}</div>
                <p class="help-text">Amount each new bid must exceed the current bid</p>
              </div>
            </div>
          </div>

          <!-- Navigation and Action Buttons -->
          <div class="modal-actions">
            <div class="action-left">
              <button v-if="currentStep > 1" type="button" @click="previousStep" class="btn btn-outline">
                ← Previous
              </button>
            </div>
            
            <div class="action-right">
              <button type="button" @click="cancelCreateItem" class="btn btn-secondary">
                Cancel
              </button>
              <button v-if="currentStep < 3" type="button" @click="nextStep" class="btn btn-primary">
                Next →
              </button>
              <button v-else type="submit" class="btn btn-primary" :disabled="isSubmitting">
                <span v-if="isSubmitting">Creating...</span>
                <span v-else>List Item</span>
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import config from '@/config';

const router = useRouter();

const items = ref([]);
const searchQuery = ref('');
const categoryFilter = ref('');
const conditionFilter = ref('');
const showCreateModal = ref(false);
const selectedImages = ref([]);
const currentStep = ref(1);
const isSubmitting = ref(false);
const isLoading = ref(true);
const formErrors = ref({
  title: '',
  description: '',
  price: '',
  starting_bid: '',
  bid_increment: ''
});

const newItem = ref({
  title: '',
  description: '',
  category: '',
  condition: '',
  price: '',
  is_auction: false,
  starting_bid: '',
  bid_increment: '1000'
});

const filteredItems = computed(() => {
  // Ensure items.value is always an array before filtering
  if (!Array.isArray(items.value)) {
    return [];
  }
  
  return items.value.filter(item => {
    // Add null checks for item properties
    if (!item || typeof item !== 'object') {
      return false;
    }
    
    const itemTitle = item.title || '';
    const itemDescription = item.description || '';
    const itemCategory = item.category || '';
    const itemCondition = item.condition || '';
    
    const matchesSearch = !searchQuery.value || 
      itemTitle.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      itemDescription.toLowerCase().includes(searchQuery.value.toLowerCase());
    
    const matchesCategory = !categoryFilter.value || itemCategory === categoryFilter.value;
    const matchesCondition = !conditionFilter.value || itemCondition === conditionFilter.value;
    
    return matchesSearch && matchesCategory && matchesCondition;
  });
});

async function loadItems() {
  try {
    // Ensure items is always an array during loading
    if (!Array.isArray(items.value)) {
      items.value = [];
    }
    
    const response = await fetch(`${config.STORE_API_URL}/items`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    });
    
    if (response.ok) {
      const data = await response.json();
      console.log('Store API response:', data);
      
      // Backend returns {items: [...], total: X, page: Y, per_page: Z}
      // Extract the items array from the response with extensive safety checks
      let extractedItems = [];
      
      if (data && typeof data === 'object') {
        if (Array.isArray(data.items)) {
          extractedItems = data.items;
        } else if (Array.isArray(data)) {
          extractedItems = data;
        }
      }
      
      // Validate each item has required properties
      const validItems = extractedItems.filter(item => 
        item && 
        typeof item === 'object' && 
        (item.id !== undefined || item.title !== undefined)
      );
      
      items.value = validItems;
      console.log(`Loaded ${validItems.length} valid items`);
    } else {
      console.error('Failed to load items:', response.status, response.statusText);
      items.value = []; // Set to empty array on error
    }
  } catch (error) {
    console.error('Error loading items:', error);
    items.value = []; // Set to empty array on error
  } finally {
    isLoading.value = false;
  }
}

// Debug function to analyze all form data
function debugFormData(data) {
  console.log('=== COMPREHENSIVE FORM DEBUG ===');
  
  for (const [key, value] of Object.entries(data)) {
    console.log(`Field: ${key}`);
    console.log(`  Raw value:`, value);
    console.log(`  Type:`, typeof value);
    console.log(`  JSON:`, JSON.stringify(value));
    console.log(`  Length:`, String(value).length);
    console.log(`  Contains dash:`, String(value).includes('-'));
    console.log(`  Char codes:`, String(value).split('').map(c => `${c}(${c.charCodeAt(0)})`).join(' '));
    console.log('---');
  }
}

async function createItem() {
  console.log('🚀 Starting item creation...');
  
  // Debug current form state
  debugFormData({
    title: newItem.value.title,
    description: newItem.value.description,
    category: newItem.value.category,
    condition: newItem.value.condition,
    price: newItem.value.price,
    is_auction: newItem.value.is_auction,
    starting_bid: newItem.value.starting_bid,
    bid_increment: newItem.value.bid_increment
  });
  
  // Final validation
  if (!validateCurrentStep()) {
    console.log('❌ Validation failed');
    return;
  }
  
  // Additional safety check for numeric fields
  if (!newItem.value.is_auction) {
    const priceValue = parseFloat(newItem.value.price);
    if (!newItem.value.price || newItem.value.price === '' || isNaN(priceValue) || priceValue < 1) {
      alert('Please enter a valid price greater than or equal to 1 UGX');
      return;
    }
  } else {
    const startingBidValue = parseFloat(newItem.value.starting_bid);
    const bidIncrementValue = parseFloat(newItem.value.bid_increment);
    
    if (!newItem.value.starting_bid || newItem.value.starting_bid === '' || isNaN(startingBidValue) || startingBidValue < 1) {
      alert('Please enter a valid starting bid greater than or equal to 1 UGX');
      return;
    }
    if (!newItem.value.bid_increment || newItem.value.bid_increment === '' || isNaN(bidIncrementValue) || bidIncrementValue < 500) {
      alert('Please enter a valid bid increment of at least 500 UGX');
      return;
    }
  }
  
  isSubmitting.value = true;
  
  try {
    // Try JSON approach first (without images to test)
    const title = newItem.value.title.trim();
    const description = newItem.value.description.trim();
    const category = newItem.value.category;
    const condition = newItem.value.condition;
    const isAuction = newItem.value.is_auction;
    
    console.log('Basic form data:', {
      title: JSON.stringify(title),
      description: JSON.stringify(description), 
      category: JSON.stringify(category),
      condition: JSON.stringify(condition),
      isAuction: isAuction
    });
    
    // Prepare JSON payload - match CreateStoreItemRequest model
    const jsonPayload = {
      title: title,
      description: description,
      category: category,
      condition: condition === "like_new" ? "like-new" : condition, // Fix condition format
      price_type: isAuction ? "bidding" : "fixed" // Use model field names
    };
    
    if (newItem.value.is_auction) {
      // Clean and validate starting bid
      let startingBidStr = String(newItem.value.starting_bid || '0').replace(/[^0-9.]/g, '');
      let bidIncrementStr = String(newItem.value.bid_increment || '1000').replace(/[^0-9.]/g, '');
      
      let startingBid = parseFloat(startingBidStr) || 0;
      let bidIncrement = parseFloat(bidIncrementStr) || 1000;
      
      // Ensure positive values and round to integers
      startingBid = Math.max(1, Math.round(startingBid));
      bidIncrement = Math.max(500, Math.round(bidIncrement));
      
      console.log('Auction values:', { 
        finalStartingBid: startingBid, 
        finalBidIncrement: bidIncrement 
      });
      
      jsonPayload.starting_bid = startingBid;
      jsonPayload.min_bid_increment = bidIncrement; // Use correct model field name
    } else {
      // Clean and validate price
      let priceStr = String(newItem.value.price || '0').replace(/[^0-9.]/g, '');
      let price = parseFloat(priceStr) || 0;
      
      // Ensure positive value and round to integer
      price = Math.max(1, Math.round(price));
      
      console.log('Fixed price value:', { 
        originalValue: newItem.value.price,
        cleanedString: priceStr,
        finalPrice: price,
        priceType: typeof price
      });
      
      jsonPayload.fixed_price = price; // Use correct model field name
    }
    
    console.log('📦 JSON Payload to send:', JSON.stringify(jsonPayload, null, 2));

    // Send as JSON (backend now supports JSON)
    console.log('🚀 Sending JSON request to:', `${config.STORE_API_URL}/items`);
    
    // If images are selected, use FormData to include them
    let requestBody;
    let requestHeaders = {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    };
    
    if (selectedImages.value.length > 0) {
      // Use FormData for requests with images
      console.log('🖼️ Images selected, using FormData format');
      const formData = new FormData();
      
      // Add all JSON fields to FormData
      Object.keys(jsonPayload).forEach(key => {
        if (jsonPayload[key] !== undefined && jsonPayload[key] !== null) {
          formData.append(key, String(jsonPayload[key]));
        }
      });
      
      // Add images to FormData
      selectedImages.value.forEach((image, index) => {
        formData.append('images', image.file);
        console.log(`Added image ${index + 1}:`, image.file.name);
      });
      
      requestBody = formData;
      // Don't set Content-Type for FormData - browser will set it with boundary
    } else {
      // Use JSON for requests without images
      console.log('📄 No images, using JSON format');
      requestHeaders['Content-Type'] = 'application/json';
      requestBody = JSON.stringify(jsonPayload);
    }
    
    const response = await fetch(`${config.STORE_API_URL}/items`, {
      method: 'POST',
      headers: requestHeaders,
      body: requestBody
    });
    
    console.log('Response status:', response.status);
    console.log('Response headers:', Object.fromEntries(response.headers.entries()));
    
    if (response.ok) {
      const responseData = await response.json();
      console.log('Success response:', responseData);
      showCreateModal.value = false;
      await loadItems();
      resetForm();
      alert('Item listed successfully!');
    } else {
      const errorText = await response.text();
      console.error('Error response text:', errorText);
      
      let errorData;
      try {
        errorData = JSON.parse(errorText);
      } catch (e) {
        errorData = { error: errorText || 'Failed to create item listing' };
      }
      
      console.error('Parsed error data:', errorData);
      alert(errorData.error || errorData.message || 'Failed to create item listing');
    }
  } catch (error) {
    console.error('Error creating item:', error);
    alert('Error creating item listing. Please try again.');
  } finally {
    isSubmitting.value = false;
  }
}

function handleImageSelect(event) {
  const files = Array.from(event.target.files);
  const remainingSlots = 3 - selectedImages.value.length;
  const filesToAdd = files.slice(0, remainingSlots);
  
  filesToAdd.forEach(file => {
    if (file.type.startsWith('image/')) {
      const reader = new FileReader();
      reader.onload = (e) => {
        selectedImages.value.push({
          file: file,
          preview: e.target.result
        });
      };
      reader.readAsDataURL(file);
    }
  });
  
  // Reset input
  event.target.value = '';
}

function removeImage(index) {
  selectedImages.value.splice(index, 1);
}

// Step navigation functions
function nextStep() {
  if (validateCurrentStep()) {
    currentStep.value++;
  }
}

function previousStep() {
  currentStep.value--;
  clearErrors();
}

function validateCurrentStep() {
  clearErrors();
  
  if (currentStep.value === 1) {
    let isValid = true;
    
    if (!newItem.value.title.trim()) {
      formErrors.value.title = 'Item title is required';
      isValid = false;
    } else if (newItem.value.title.length < 3) {
      formErrors.value.title = 'Title must be at least 3 characters';
      isValid = false;
    }
    
    if (!newItem.value.description.trim()) {
      formErrors.value.description = 'Description is required';
      isValid = false;
    } else if (newItem.value.description.length < 10) {
      formErrors.value.description = 'Description must be at least 10 characters';
      isValid = false;
    }
    
    return isValid;
  }
  
  if (currentStep.value === 2) {
    return newItem.value.category && newItem.value.condition;
  }
  
  if (currentStep.value === 3) {
    let isValid = true;
    
    if (!newItem.value.is_auction) {
      const price = Number(newItem.value.price);
      if (!newItem.value.price || isNaN(price) || price <= 0) {
        formErrors.value.price = 'Price must be greater than 0';
        isValid = false;
      }
    } else {
      const startingBid = Number(newItem.value.starting_bid);
      const bidIncrement = Number(newItem.value.bid_increment);
      
      if (!newItem.value.starting_bid || isNaN(startingBid) || startingBid <= 0) {
        formErrors.value.starting_bid = 'Starting bid must be greater than 0';
        isValid = false;
      }
      if (!newItem.value.bid_increment || isNaN(bidIncrement) || bidIncrement < 500) {
        formErrors.value.bid_increment = 'Bid increment must be at least 500 UGX';
        isValid = false;
      }
    }
    
    return isValid;
  }
  
  return true;
}

function clearErrors() {
  formErrors.value = {
    title: '',
    description: '',
    price: '',
    starting_bid: '',
    bid_increment: ''
  };
}

function sanitizeNumberInput(field, event) {
  const value = event.target.value;
  
  // Remove any non-numeric characters except decimal point
  const sanitized = value.replace(/[^0-9.]/g, '');
  
  // Ensure only one decimal point
  const parts = sanitized.split('.');
  let finalValue = parts[0];
  if (parts.length > 1) {
    finalValue += '.' + parts[1];
  }
  
  // Update the field value
  newItem.value[field] = finalValue;
  
  console.log(`Sanitized ${field}: "${value}" -> "${finalValue}"`);
}

function resetForm() {
  newItem.value = {
    title: '',
    description: '',
    category: '',
    condition: '',
    price: '',
    is_auction: false,
    starting_bid: '',
    bid_increment: '1000'
  };
  selectedImages.value = [];
  currentStep.value = 1;
  isSubmitting.value = false;
  clearErrors();
}

function cancelCreateItem() {
  showCreateModal.value = false;
  resetForm();
}

function viewItem(item) {
  if (item && item.id) {
    router.push({ name: 'store-item', params: { id: item.id } });
  } else {
    console.error('Cannot view item: Invalid item or missing ID', item);
  }
}

function searchItems() {
  // Debounce search if needed
}

function filterItems() {
  // Additional filter logic if needed
}

function formatCurrency(amount) {
  // Add safety check for null/undefined values
  const safeAmount = Number(amount) || 0;
  return new Intl.NumberFormat('en-UG', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(safeAmount);
}

function handleImageError(event) {
  console.warn('Failed to load image:', event.target.src);
  // Hide the broken image and show placeholder
  event.target.style.display = 'none';
  const placeholder = event.target.parentElement.querySelector('.image-placeholder');
  if (placeholder) {
    placeholder.style.display = 'flex';
  }
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

.image-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #f9fafb;
  color: #9ca3af;
  border: 2px dashed #e5e7eb;
  border-radius: 0.5rem;
}

.image-placeholder svg {
  margin-bottom: 0.5rem;
}

.image-placeholder span {
  font-size: 0.875rem;
  font-weight: 500;
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

/* Image Upload Styles */
.image-upload-section {
  margin-top: 0.5rem;
}

.image-previews {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
  margin-bottom: 0.5rem;
}

.image-preview {
  position: relative;
  width: 100px;
  height: 100px;
  border-radius: 0.375rem;
  overflow: hidden;
  border: 1px solid #e5e7eb;
}

.image-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.remove-image {
  position: absolute;
  top: 0.25rem;
  right: 0.25rem;
  background: rgba(0, 0, 0, 0.7);
  color: white;
  border: none;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.remove-image:hover {
  background: rgba(0, 0, 0, 0.9);
}

.image-upload-box {
  width: 100px;
  height: 100px;
  border: 2px dashed #e5e7eb;
  border-radius: 0.375rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
}

.image-upload-box:hover {
  border-color: #4F46E5;
  background: #f9fafb;
}

.upload-label {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  cursor: pointer;
  color: #6b7280;
}

.upload-icon {
  font-size: 1.5rem;
  line-height: 1;
}

.image-help {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

/* Loading State */
.loading-state {
  text-align: center;
  padding: 4rem 2rem;
  color: #6b7280;
}

.loading-spinner {
  font-size: 1.125rem;
  font-weight: 500;
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  color: #6b7280;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: #f3f4f6;
  margin-bottom: 1.5rem;
  color: #9ca3af;
}

.empty-state h3 {
  font-size: 1.5rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.5rem;
}

.empty-state p {
  margin-bottom: 2rem;
  font-size: 1rem;
  line-height: 1.5;
}

/* Improved Form Styles */
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem 1.5rem 0 1.5rem;
  border-bottom: 1px solid #e5e7eb;
  margin-bottom: 1rem;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #111827;
}

.close-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: #6b7280;
  border-radius: 0.375rem;
  transition: all 0.2s;
}

.close-btn:hover {
  background-color: #f3f4f6;
  color: #374151;
}

.form-progress {
  padding: 0 1.5rem 1.5rem;
  border-bottom: 1px solid #e5e7eb;
}

.progress-steps {
  display: flex;
  justify-content: space-between;
  position: relative;
}

.step {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  position: relative;
}

.step:not(:last-child)::after {
  content: '';
  position: absolute;
  top: 15px;
  left: 60%;
  right: -40%;
  height: 2px;
  background-color: #e5e7eb;
  z-index: 1;
}

.step.completed:not(:last-child)::after {
  background-color: #4f46e5;
}

.step-number {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background-color: #e5e7eb;
  color: #6b7280;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
  position: relative;
  z-index: 2;
  transition: all 0.2s;
}

.step.active .step-number,
.step.completed .step-number {
  background-color: #4f46e5;
  color: white;
}

.step-label {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
}

.step.active .step-label,
.step.completed .step-label {
  color: #4f46e5;
}

.improved-form {
  padding: 1.5rem;
}

.form-step {
  min-height: 400px;
}

.form-step h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: #111827;
  margin-bottom: 1.5rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-label {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #374151;
}

.label-text {
  font-size: 0.875rem;
}

.required {
  color: #ef4444;
  font-size: 0.875rem;
}

.optional {
  color: #6b7280;
  font-size: 0.75rem;
  font-weight: 400;
}

.form-input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 1rem;
  transition: all 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.form-input.error {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
}

.error-message {
  color: #ef4444;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.character-count {
  color: #6b7280;
  font-size: 0.75rem;
  text-align: right;
  margin-top: 0.25rem;
}

.help-text {
  color: #6b7280;
  font-size: 0.875rem;
  margin-top: 0.25rem;
  line-height: 1.4;
}

.upload-label {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 500;
}

.pricing-type-selector {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 2rem;
}

.pricing-option {
  padding: 1.5rem;
  border: 2px solid #e5e7eb;
  border-radius: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}

.pricing-option:hover {
  border-color: #d1d5db;
  background-color: #f9fafb;
}

.pricing-option.active {
  border-color: #4f46e5;
  background-color: #eef2ff;
}

.option-icon {
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

.option-content h4 {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin-bottom: 0.25rem;
}

.option-content p {
  color: #6b7280;
  font-size: 0.875rem;
  margin: 0;
}

.currency-input {
  position: relative;
  display: flex;
  align-items: center;
}

.currency-symbol {
  position: absolute;
  left: 0.75rem;
  color: #6b7280;
  font-weight: 500;
  z-index: 1;
}

.currency-input .form-input {
  padding-left: 3rem;
}

.auction-fields {
  display: grid;
  gap: 1rem;
}

.modal-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 1.5rem;
  border-top: 1px solid #e5e7eb;
  margin-top: 2rem;
}

.action-left, .action-right {
  display: flex;
  gap: 0.75rem;
}

.btn-outline {
  background: white;
  border: 1px solid #d1d5db;
  color: #374151;
}

.btn-outline:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

/* Responsive improvements */
@media (max-width: 768px) {
  .form-row {
    grid-template-columns: 1fr;
  }
  
  .pricing-type-selector {
    grid-template-columns: 1fr;
  }
  
  .modal-actions {
    flex-direction: column;
    gap: 1rem;
  }
  
  .action-left, .action-right {
    width: 100%;
    justify-content: center;
  }
  
  .progress-steps {
    flex-direction: column;
    gap: 1rem;
  }
  
  .step:not(:last-child)::after {
    display: none;
  }
}
</style>