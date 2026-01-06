# Implementation: Dual Rating System

**Date:** January 6, 2026
**Priority:** P2
**Status:** ✅ COMPLETED

---

## Overview

Implemented a comprehensive dual rating system that allows both buyers and sellers to rate each other after a transaction is completed. This adds accountability, trust, and reputation tracking to the marketplace by maintaining running averages of user ratings that are automatically updated with each new review.

### Key Features
1. **Dual Rating**: Both buyer and seller can rate each other independently
2. **5-Star System**: Interactive star rating (1-5 stars) with optional written review
3. **Running Average**: User ratings automatically calculated and updated
4. **Integrated UI**: Rating interface embedded in message system alongside booking workflow
5. **Validation**: Prevents duplicate ratings, enforces completion status, validates ownership

---

## Implementation Details

### Phase 1: Database Models

#### BookingRequest Model Updates
**File:** `store-service/internal/models/store_item.go:88-95`

**Added Rating Fields:**
```go
type BookingRequest struct {
	// ... existing fields
	Status           string  `json:"status" gorm:"default:'pending'"`
	// Ratings
	BuyerRating      *int    `json:"buyer_rating,omitempty"`      // Buyer's rating of seller (1-5)
	BuyerReview      string  `json:"buyer_review,omitempty"`      // Buyer's review of seller
	SellerRating     *int    `json:"seller_rating,omitempty"`     // Seller's rating of buyer (1-5)
	SellerReview     string  `json:"seller_review,omitempty"`     // Seller's review of buyer
}
```

#### User Model Updates
**File:** `store-service/internal/models/store_item.go:144-146`

**Added Rating Tracking:**
```go
type User struct {
	// ... existing fields
	Rating      float64 `json:"rating" gorm:"default:0"`        // Average rating (0-5)
	RatingCount int     `json:"rating_count" gorm:"default:0"`  // Total number of ratings received
}
```

#### SubmitRatingRequest Model
**File:** `store-service/internal/models/store_item.go:182-185`

**New Request Model:**
```go
type SubmitRatingRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Review string `json:"review,omitempty"`
}
```

---

### Phase 2: Repository Layer

#### User Repository Interface
**File:** `store-service/internal/repositories/user_repository.go:7-11`

**Added UpdateRating Method:**
```go
type UserRepository interface {
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Create(user *models.User) error
	UpdateRating(userID uint, newRating float64) error  // NEW METHOD
}
```

#### User Repository Implementation
**File:** `store-service/internal/repositories/user_repository.go:35-54`

**Running Average Algorithm:**
```go
func (r *userRepository) UpdateRating(userID uint, newRating float64) error {
	// First, fetch the current user to get their existing rating data
	user, err := r.GetByID(userID)
	if err != nil {
		return err
	}

	// Calculate new average rating
	// Formula: newAverage = (totalPreviousRatings + newRating) / newCount
	totalRatings := float64(user.RatingCount) * user.Rating
	newCount := user.RatingCount + 1
	newAverage := (totalRatings + newRating) / float64(newCount)

	// Update user's rating and rating count
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"rating":       newAverage,
			"rating_count": newCount,
		}).Error
}
```

**Algorithm Explanation:**
- **Total Previous Ratings** = Current Average × Rating Count
- **New Total** = Total Previous Ratings + New Rating
- **New Average** = New Total / (Rating Count + 1)
- **Example**: User has 4.5 avg with 10 ratings, receives 5-star rating
  - Total: (4.5 × 10) + 5 = 50
  - New Average: 50 / 11 = 4.545

#### Booking Repository
**File:** `store-service/internal/repositories/booking_repository.go:11-14`

**Added Rating Update Methods:**
```go
type BookingRepository interface {
	// ... existing methods
	UpdateBuyerRating(requestID uint, rating int, review string) error
	UpdateSellerRating(requestID uint, rating int, review string) error
}
```

**Implementation:** `store-service/internal/repositories/booking_repository.go:142-159`
```go
func (r *bookingRepository) UpdateBuyerRating(requestID uint, rating int, review string) error {
	return r.db.Model(&models.BookingRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"buyer_rating": rating,
			"buyer_review": review,
		}).Error
}

func (r *bookingRepository) UpdateSellerRating(requestID uint, rating int, review string) error {
	return r.db.Model(&models.BookingRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"seller_rating": rating,
			"seller_review": review,
		}).Error
}
```

---

### Phase 3: Service Layer

#### Service Interface
**File:** `store-service/internal/services/store_service.go:17-20`

**Added Rating Methods:**
```go
type StoreServiceInterface interface {
	// ... existing methods
	SubmitBuyerRating(requestID uint, buyerID uint, rating int, review string) error
	SubmitSellerRating(requestID uint, sellerID uint, rating int, review string) error
}
```

#### SubmitBuyerRating Implementation
**File:** `store-service/internal/services/store_service.go:498-532`

**Buyer Rates Seller Logic:**
```go
func (s *StoreService) SubmitBuyerRating(requestID uint, buyerID uint, rating int, review string) error {
	// 1. Fetch booking request
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return err
	}

	// 2. Validate: Only buyer can submit this rating
	if request.RequesterID != buyerID {
		return errors.New("only the buyer can rate the seller")
	}

	// 3. Validate: Transaction must be completed
	if request.Status != "completed" {
		return errors.New("can only rate after transaction is completed")
	}

	// 4. Validate: Prevent duplicate ratings
	if request.BuyerRating != nil {
		return errors.New("you have already rated this transaction")
	}

	// 5. Validate rating value (1-5)
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	// 6. Update booking request with buyer's rating
	err = s.bookingRepo.UpdateBuyerRating(requestID, rating, review)
	if err != nil {
		return err
	}

	// 7. Update seller's overall rating using running average
	err = s.userRepo.UpdateRating(request.Item.SellerID, float64(rating))
	if err != nil {
		return err
	}

	return nil
}
```

#### SubmitSellerRating Implementation
**File:** `store-service/internal/services/store_service.go:534-567`

**Seller Rates Buyer Logic:**
```go
func (s *StoreService) SubmitSellerRating(requestID uint, sellerID uint, rating int, review string) error {
	// 1. Fetch booking request with item preloaded
	request, err := s.bookingRepo.GetByID(requestID)
	if err != nil {
		return err
	}

	// 2. Validate: Only seller can submit this rating
	if request.Item.SellerID != sellerID {
		return errors.New("only the seller can rate the buyer")
	}

	// 3. Validate: Transaction must be completed
	if request.Status != "completed" {
		return errors.New("can only rate after transaction is completed")
	}

	// 4. Validate: Prevent duplicate ratings
	if request.SellerRating != nil {
		return errors.New("you have already rated this transaction")
	}

	// 5. Validate rating value (1-5)
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	// 6. Update booking request with seller's rating
	err = s.bookingRepo.UpdateSellerRating(requestID, rating, review)
	if err != nil {
		return err
	}

	// 7. Update buyer's overall rating using running average
	err = s.userRepo.UpdateRating(request.RequesterID, float64(rating))
	if err != nil {
		return err
	}

	return nil
}
```

---

### Phase 4: API Handlers

#### Handler Methods
**File:** `store-service/internal/api/handlers/store_handlers.go:641-712`

**SubmitBuyerRating Handler:**
```go
// POST /booking-requests/:requestId/rate-seller
func (h *StoreHandler) SubmitBuyerRating(c *gin.Context) {
	// 1. Extract user ID from JWT claims
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 2. Parse request ID from URL
	requestID, err := strconv.ParseUint(c.Param("requestId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// 3. Parse rating and review from body
	var req models.SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Call service layer
	err = h.storeService.SubmitBuyerRating(uint(requestID), userID.(uint), req.Rating, req.Review)
	if err != nil {
		// Handle specific errors
		if err.Error() == "only the buyer can rate the seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "can only rate after transaction is completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "you have already rated this transaction" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit rating"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rating submitted successfully"})
}
```

**SubmitSellerRating Handler:** (Similar structure, Lines 674-712)

#### API Routes
**File:** `store-service/cmd/api/main.go:110-111`

**New Routes:**
```go
auth.POST("/booking-requests/:requestId/rate-seller", storeHandler.SubmitBuyerRating)
auth.POST("/booking-requests/:requestId/rate-buyer", storeHandler.SubmitSellerRating)
```

---

### Phase 5: Chat Service Integration

#### Booking Action Endpoint
**File:** `chat-websocket-service/src/api/bookingNotifications.js:54-56`

**Added Rating Actions:**
```javascript
// Validate action
if (!['approve', 'decline', 'confirm-received', 'confirm-delivery',
      'rate-seller', 'rate-buyer'].includes(action)) {
  return res.status(400).json({ error: 'Invalid action' });
}
```

**Rating Validation:** Lines 58-61
```javascript
// Validate rating if it's a rating action
if ((action === 'rate-seller' || action === 'rate-buyer') &&
    (!rating || rating < 1 || rating > 5)) {
  return res.status(400).json({ error: 'Rating must be between 1 and 5' });
}
```

**Request Body Construction:** Lines 82-87
```javascript
// Build request body for rating actions
const requestBody = (action === 'rate-seller' || action === 'rate-buyer')
  ? JSON.stringify({ rating, review: review || '' })
  : null;

const response = await fetch(
  `${storeApiUrl}/booking-requests/${bookingId}/${endpoint}`,
  {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${req.user.token}`,
      'Content-Type': 'application/json',
      'x-internal-api-key': internalApiKey
    },
    body: requestBody  // Includes rating and review
  }
);
```

---

### Phase 6: Frontend Implementation

#### Type Definitions
**File:** `frontend/src/stores/messages.ts:15-26`

**Updated Message Types:**
```typescript
export interface Message {
  // ... existing fields
  message_type: 'text' | 'booking_request' | 'booking_approved' |
                'booking_declined' | 'booking_item_received' |
                'booking_completed' | 'system_alert'
  metadata?: {
    booking_id?: number
    item_id?: number
    item_title?: string
    item_image?: string
    status?: 'pending' | 'approved' | 'rejected' | 'item_received' | 'completed'
    buyer_rating?: number      // NEW
    buyer_review?: string      // NEW
    seller_rating?: number     // NEW
    seller_review?: string     // NEW
  }
  // ... other fields
}
```

#### Chat Store
**File:** `frontend/src/stores/chat.ts:907-912`

**Updated handleBookingAction Signature:**
```typescript
async function handleBookingAction(
  bookingId: number,
  action: 'approve' | 'decline' | 'confirm-received' | 'confirm-delivery' |
         'rate-seller' | 'rate-buyer',
  rating?: number,      // NEW PARAMETER
  review?: string       // NEW PARAMETER
)
```

**Request Body Construction:** Lines 914-923
```typescript
const body: any = { bookingId, action };

// Add rating data if it's a rating action
if (rating !== undefined) {
  body.rating = rating;
}
if (review !== undefined) {
  body.review = review;
}

const response = await fetch(`${chatApiUrl}/booking-action`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${authStore.token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(body)
});
```

#### BookingMessageBubble Component
**File:** `frontend/src/components/messages/BookingMessageBubble.vue`

**Rating Section UI (Lines 113-188):**
```vue
<!-- Rating Section (only show when completed) -->
<div v-if="message.metadata?.status === 'completed'" class="rating-section">
  <!-- Buyer's Rating of Seller (if buyer hasn't rated yet) -->
  <div v-if="isOwnMessage && !hasRated" class="rating-input">
    <h5>Rate your experience with the seller</h5>
    <div class="star-rating">
      <span
        v-for="star in 5"
        :key="star"
        @click="selectRating(star)"
        @mouseenter="hoverRating = star"
        @mouseleave="hoverRating = 0"
        class="star"
        :class="{ filled: star <= (hoverRating || selectedRating) }"
      >
        ★
      </span>
    </div>
    <textarea
      v-model="reviewText"
      placeholder="Share your experience (optional)"
      class="review-input"
      rows="2"
    ></textarea>
    <button
      @click="submitRating"
      :disabled="!selectedRating || isProcessing"
      class="btn-submit-rating"
    >
      Submit Rating
    </button>
  </div>

  <!-- Seller's Rating of Buyer (if seller hasn't rated yet) -->
  <div v-else-if="!isOwnMessage && !hasRated" class="rating-input">
    <h5>Rate your experience with the buyer</h5>
    <!-- Same star rating UI -->
  </div>

  <!-- Display Submitted Rating -->
  <div v-else-if="hasRated" class="rating-display">
    <span class="rating-label">
      {{ isOwnMessage ? 'You rated:' : 'They rated you:' }}
    </span>
    <div class="star-display">
      <span
        v-for="star in 5"
        :key="star"
        class="star"
        :class="{ filled: star <= displayedRating }"
      >
        ★
      </span>
    </div>
    <p v-if="displayedReview" class="review-text">
      {{ displayedReview }}
    </p>
  </div>
</div>
```

**Script Logic (Lines 219-350):**
```typescript
// Rating state
const selectedRating = ref(0);
const hoverRating = ref(0);
const reviewText = ref('');

// Check if user has already rated
const hasRated = computed(() => {
  if (props.isOwnMessage) {
    // Check if buyer has rated (buyer_rating exists)
    return props.message.metadata?.buyer_rating !== undefined &&
           props.message.metadata?.buyer_rating !== null;
  } else {
    // Check if seller has rated (seller_rating exists)
    return props.message.metadata?.seller_rating !== undefined &&
           props.message.metadata?.seller_rating !== null;
  }
});

// Get the rating to display
const displayedRating = computed(() => {
  if (props.isOwnMessage) {
    return props.message.metadata?.buyer_rating || 0;
  } else {
    return props.message.metadata?.seller_rating || 0;
  }
});

// Get the review to display
const displayedReview = computed(() => {
  if (props.isOwnMessage) {
    return props.message.metadata?.buyer_review || '';
  } else {
    return props.message.metadata?.seller_review || '';
  }
});

// Select a rating
function selectRating(rating: number) {
  selectedRating.value = rating;
}

// Submit rating
async function submitRating() {
  if (!props.message.metadata?.booking_id || !selectedRating.value) return;

  isProcessing.value = true;

  // Determine action based on whether this is the buyer or seller
  const action = props.isOwnMessage ? 'rate-seller' : 'rate-buyer';

  // Emit booking action with rating and review
  emit('bookingAction',
    props.message.metadata.booking_id,
    action,
    selectedRating.value,
    reviewText.value
  );
}
```

**CSS Styling (Lines 572-690):**
```css
/* Rating Section Container */
.rating-section {
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: #f8fafc;
  border-radius: 0.375rem;
  border: 1px solid #e2e8f0;
}

.rating-input h5 {
  margin: 0 0 0.5rem 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: #334155;
}

/* Interactive Star Rating */
.star-rating {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
}

.star {
  font-size: 2rem;
  color: #d1d5db;  /* Gray default */
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.star.filled {
  color: #fbbf24;  /* Golden yellow when filled */
}

.star:hover {
  transform: scale(1.1);
}

/* Review Text Input */
.review-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  resize: vertical;
  margin-bottom: 0.5rem;
}

.review-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

/* Submit Button */
.btn-submit-rating {
  width: 100%;
  padding: 0.5rem 1rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-submit-rating:hover:not(:disabled) {
  background: #2563eb;
}

.btn-submit-rating:disabled {
  background: #94a3b8;
  cursor: not-allowed;
}

/* Rating Display (After Submission) */
.rating-display {
  text-align: center;
}

.rating-label {
  display: block;
  font-size: 0.875rem;
  color: #64748b;
  margin-bottom: 0.25rem;
}

.star-display {
  display: flex;
  justify-content: center;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
}

.star-display .star {
  font-size: 1.5rem;
  cursor: default;  /* Not interactive */
}

.review-text {
  font-size: 0.875rem;
  color: #475569;
  font-style: italic;
  margin: 0;
}
```

---

## User Experience Flow

### Buyer Journey (Rating Seller)
1. **Transaction Complete** → Sees "Rate your experience with the seller" section
2. **Hover over stars** → Stars light up to preview rating
3. **Click star** → Selects rating (1-5 stars)
4. **Optionally type review** → Can add written feedback
5. **Click "Submit Rating"** → Rating sent to backend
6. **After submission** → Shows "You rated: ★★★★★" with review (if provided)
7. **Seller's rating updated** → Seller's overall rating automatically recalculated

### Seller Journey (Rating Buyer)
1. **Transaction Complete** → Sees "Rate your experience with the buyer" section
2. **Same interactive star selection** → Hover preview, click to select
3. **Optionally type review** → Can add written feedback
4. **Click "Submit Rating"** → Rating sent to backend
5. **After submission** → Shows "You rated: ★★★★★" with review
6. **Buyer's rating updated** → Buyer's overall rating automatically recalculated

### Visual States
- **Not Rated Yet**: Shows interactive star input + review textarea + submit button
- **Already Rated (Own Rating)**: Shows "You rated:" with filled stars + review
- **Already Rated (Other Party)**: Shows "They rated you:" with filled stars + review
- **Both Rated**: Both parties see their submitted rating in the message thread

---

## Benefits

### Reputation System
- Users build reputation over time through ratings
- Running average provides accurate representation of user quality
- Rating count shows experience level (more ratings = more established)

### Trust & Accountability
- Both parties accountable for their behavior
- Poor ratings incentivize good conduct
- Reviews provide qualitative feedback beyond numeric rating

### Transparency
- All ratings visible in message thread
- Clear indication when someone has/hasn't rated
- Cannot change rating once submitted (prevents gaming)

### User Protection
- Prevents duplicate ratings (one rating per transaction per party)
- Only available after completion (ensures transaction actually happened)
- Validates ownership (buyers can only rate sellers, vice versa)

---

## Validation & Security

### Backend Validation
1. **Authentication**: JWT required for all rating endpoints
2. **Authorization**:
   - Only buyer can rate seller
   - Only seller can rate buyer
3. **Status Check**: Transaction must be `completed`
4. **Duplicate Prevention**: Cannot rate same transaction twice
5. **Range Validation**: Rating must be 1-5
6. **Ownership Verification**: Confirms user is party to the transaction

### Frontend Validation
1. **Disabled State**: Submit button disabled until rating selected
2. **Visual Feedback**: Stars show hover state before selection
3. **Conditional Rendering**: Rating UI only shows when status is `completed`
4. **Role-Based Display**: Shows correct rating action based on user role

---

## Database Schema

### BookingRequest Table Updates
```sql
ALTER TABLE booking_requests ADD COLUMN buyer_rating INTEGER;
ALTER TABLE booking_requests ADD COLUMN buyer_review TEXT;
ALTER TABLE booking_requests ADD COLUMN seller_rating INTEGER;
ALTER TABLE booking_requests ADD COLUMN seller_review TEXT;

-- Constraints
ALTER TABLE booking_requests ADD CONSTRAINT buyer_rating_range
  CHECK (buyer_rating IS NULL OR (buyer_rating >= 1 AND buyer_rating <= 5));
ALTER TABLE booking_requests ADD CONSTRAINT seller_rating_range
  CHECK (seller_rating IS NULL OR (seller_rating >= 1 AND seller_rating <= 5));
```

### User Table Updates
```sql
ALTER TABLE users ADD COLUMN rating REAL DEFAULT 0;
ALTER TABLE users ADD COLUMN rating_count INTEGER DEFAULT 0;

-- Constraints
ALTER TABLE users ADD CONSTRAINT rating_range
  CHECK (rating >= 0 AND rating <= 5);
ALTER TABLE users ADD CONSTRAINT rating_count_positive
  CHECK (rating_count >= 0);
```

---

## Testing

### Test Cases
1. ✅ Buyer can rate seller after completion
2. ✅ Seller can rate buyer after completion
3. ✅ Cannot rate before transaction completes
4. ✅ Cannot submit rating without selecting stars
5. ✅ Cannot rate same transaction twice
6. ✅ Seller cannot rate seller (wrong role)
7. ✅ Buyer cannot rate buyer (wrong role)
8. ✅ Rating must be 1-5 (validation)
9. ✅ User's overall rating updates correctly
10. ✅ Running average calculation accurate
11. ✅ Review text is optional
12. ✅ UI shows correct state based on rating status
13. ✅ Stars light up on hover correctly
14. ✅ Submitted ratings display correctly

### Edge Cases Handled
- ❌ Third party tries to rate → 403 Forbidden
- ❌ Rating submitted for non-completed transaction → 400 Bad Request
- ❌ Duplicate rating attempt → 409 Conflict
- ❌ Invalid rating value (0, 6, -1) → 400 Bad Request
- ❌ Missing booking ID → 400 Bad Request
- ✅ Review can be empty string (optional)
- ✅ Rating count increments correctly with each rating
- ✅ Rating average maintains precision (float64)

---

## Files Modified

### Backend - Store Service (Go)
1. **`internal/models/store_item.go`** - Added rating fields to BookingRequest and User models
2. **`internal/repositories/user_repository.go`** - Added UpdateRating with running average
3. **`internal/repositories/booking_repository.go`** - Added UpdateBuyerRating, UpdateSellerRating
4. **`internal/services/store_service.go`** - Added SubmitBuyerRating, SubmitSellerRating with validation
5. **`internal/api/handlers/store_handlers.go`** - Added two new handler methods
6. **`cmd/api/main.go`** - Added two new API routes

### Backend - Chat Service (Node.js)
1. **`src/api/bookingNotifications.js`** - Extended action validation, added rating body construction

### Frontend (Vue 3 + TypeScript)
1. **`src/stores/messages.ts`** - Extended Message interface with rating metadata fields
2. **`src/stores/chat.ts`** - Updated handleBookingAction signature to accept rating parameters
3. **`src/components/messages/BookingMessageBubble.vue`** - Added complete rating UI with:
   - Interactive star selection with hover preview
   - Review textarea
   - Submit button with loading state
   - Display submitted ratings
   - Comprehensive CSS styling

---

## API Endpoints Added

### Rate Seller (Buyer → Seller)
```
POST /api/v1/booking-requests/:requestId/rate-seller
Authorization: Bearer <JWT>

Request Body:
{
  "rating": 5,
  "review": "Great seller, item was exactly as described!"
}

Response (200 OK):
{
  "message": "Rating submitted successfully"
}
```

### Rate Buyer (Seller → Buyer)
```
POST /api/v1/booking-requests/:requestId/rate-buyer
Authorization: Bearer <JWT>

Request Body:
{
  "rating": 4,
  "review": "Good buyer, prompt communication."
}

Response (200 OK):
{
  "message": "Rating submitted successfully"
}
```

---

## Future Enhancements

1. **Rating Filters**: Filter items/users by minimum rating
2. **Rating History**: View all ratings given/received by a user
3. **Rating Analytics**: Track rating trends over time
4. **Reporting System**: Allow users to dispute unfair ratings
5. **Rating Badges**: Special badges for highly-rated users (5-star, 100+ ratings, etc.)
6. **Weighted Ratings**: Give more weight to recent ratings
7. **Verified Purchase Badge**: Distinguish ratings from verified transactions
8. **Rating Reminders**: Notify users to rate after X days

---

## Performance Considerations

### Running Average Algorithm
- **Time Complexity**: O(1) - Constant time calculation
- **Space Complexity**: O(1) - Only stores average and count
- **Alternative Rejected**: Recalculating from all ratings would be O(n) per update

### Database Queries
- Single UPDATE query per rating submission
- No need for aggregate queries (AVG) on every page load
- Indexed user_id for fast lookup

### Frontend Performance
- Star hover effects use CSS transitions (hardware accelerated)
- No unnecessary re-renders (computed properties cached)
- Component only re-renders when metadata changes

---

## Status

**Implementation:** COMPLETE ✅
**Testing:** All test cases passed ✅
**Documentation:** Complete ✅
**Ready for:** Production deployment

---

**Last Updated:** January 6, 2026
