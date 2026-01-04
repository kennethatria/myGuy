# Implementation Plan: Unified Booking & Messaging Flow

**Status:** Planned
**Estimated Effort:** 3-4 days
**Priority:** P2 (Post-MVP, High UX Impact)
**Related RFC:** [RFC-booking-via-messages.md](./RFC-booking-via-messages.md)

---

## User Flow Summary

### How Users Will Book Items (New System)

**Buyer's Perspective:**
1. Browse store items, find something they want
2. Click **"Book Now"** button on item page
3. **Automatically redirected to chat** with the seller
4. See a system message: *"📋 You sent a booking request for [Item Name]"*
5. Start chatting with seller about details (optional)
6. Receive notification when seller approves/declines
7. If approved: Continue conversation to arrange pickup/payment

**Seller's Perspective:**
1. Receive **notification badge** in message center (new booking request)
2. Open the conversation to see a **special booking request message** with:
   - Item thumbnail and name
   - Buyer's name
   - **[Approve]** and **[Decline]** buttons right in the chat
3. Click **[Approve]** or **[Decline]** directly in the chat
4. System message confirms: *"✅ Booking approved. You can now discuss pickup details."*
5. Chat with buyer to finalize details

**Key Benefits:**
- ✅ All communication in one place
- ✅ No need to check individual item pages
- ✅ Conversation context preserved (request → negotiation → completion)
- ✅ Mobile-friendly (chat UI vs. complex management UI)

---

## Phase 1: Database Schema Updates

### 1.1 Chat Service - Add Message Types

**File:** `chat-websocket-service/migrations/XXXX_add_message_types.sql`

```sql
-- Add message_type column to messages table
ALTER TABLE messages
ADD COLUMN message_type VARCHAR(50) DEFAULT 'text' NOT NULL;

-- Add metadata column for structured messages
ALTER TABLE messages
ADD COLUMN metadata JSONB;

-- Create index for filtering system messages
CREATE INDEX idx_messages_type ON messages(message_type);

-- Add comments
COMMENT ON COLUMN messages.message_type IS 'Type of message: text, booking_request, booking_approved, booking_declined, system_alert';
COMMENT ON COLUMN messages.metadata IS 'Additional structured data for special message types (e.g., booking_id, item details)';
```

**Message Types:**
- `text` - Regular user message (default)
- `booking_request` - System message when buyer books item
- `booking_approved` - System message when seller approves
- `booking_declined` - System message when seller declines
- `system_alert` - General system notifications

**Metadata Structure (JSONB):**
```json
{
  "booking_id": 123,
  "item_id": 456,
  "item_title": "Red Bicycle",
  "item_image": "/uploads/store/bicycle.jpg",
  "status": "pending" // or "approved", "declined"
}
```

### 1.2 Store Service - Add Notification Flags

**File:** `store-service/migrations/XXXX_add_notification_tracking.sql`

```sql
-- Add column to track if chat notification was sent
ALTER TABLE booking_requests
ADD COLUMN chat_notified BOOLEAN DEFAULT false;

-- Add retry tracking for failed notifications
ALTER TABLE booking_requests
ADD COLUMN notification_attempts INTEGER DEFAULT 0,
ADD COLUMN last_notification_attempt TIMESTAMP;

CREATE INDEX idx_booking_requests_chat_notified ON booking_requests(chat_notified, status);
```

---

## Phase 2: Backend Implementation

### 2.1 Chat Service - New Endpoints & Functions

**File:** `chat-websocket-service/src/services/bookingMessageService.js`

```javascript
/**
 * Create a booking request system message
 */
async function createBookingRequestMessage({
  bookingId,
  itemId,
  itemTitle,
  itemImage,
  buyerId,
  sellerId
}) {
  // Create system message in chat
  const message = await db.messages.create({
    sender_id: buyerId,
    recipient_id: sellerId,
    store_item_id: itemId,
    message_type: 'booking_request',
    content: `Booking request for ${itemTitle}`,
    metadata: {
      booking_id: bookingId,
      item_id: itemId,
      item_title: itemTitle,
      item_image: itemImage,
      status: 'pending'
    }
  });

  // Emit to WebSocket
  io.to(`user_${sellerId}`).emit('message:new', message);

  return message;
}

/**
 * Update booking message status
 */
async function updateBookingMessageStatus(bookingId, status, approverId) {
  // Find the original booking request message
  const requestMessage = await db.messages.findOne({
    where: {
      message_type: 'booking_request',
      'metadata->booking_id': bookingId
    }
  });

  if (!requestMessage) {
    throw new Error('Booking request message not found');
  }

  // Update the original message metadata
  await db.messages.update({
    metadata: {
      ...requestMessage.metadata,
      status: status
    }
  }, {
    where: { id: requestMessage.id }
  });

  // Create a new system message for the status change
  const statusMessage = await db.messages.create({
    sender_id: approverId,
    recipient_id: requestMessage.sender_id,
    store_item_id: requestMessage.store_item_id,
    message_type: status === 'approved' ? 'booking_approved' : 'booking_declined',
    content: status === 'approved'
      ? `Booking approved ✅. You can now discuss pickup details.`
      : `Booking request was declined.`,
    metadata: {
      booking_id: bookingId,
      item_id: requestMessage.metadata.item_id,
      status: status
    }
  });

  // Emit to both users
  io.to(`user_${requestMessage.sender_id}`).emit('message:new', statusMessage);
  io.to(`user_${approverId}`).emit('message:new', statusMessage);

  return statusMessage;
}
```

**File:** `chat-websocket-service/src/api/bookingNotifications.js` (New internal endpoint)

```javascript
const express = require('express');
const router = express.Router();
const bookingMessageService = require('../services/bookingMessageService');

/**
 * Internal endpoint for store-service to notify chat about booking requests
 * Should be secured with internal API key or JWT
 */
router.post('/internal/booking-created', async (req, res) => {
  try {
    const { bookingId, itemId, itemTitle, itemImage, buyerId, sellerId } = req.body;

    // Validate internal API key
    if (req.headers['x-internal-api-key'] !== process.env.INTERNAL_API_KEY) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const message = await bookingMessageService.createBookingRequestMessage({
      bookingId,
      itemId,
      itemTitle,
      itemImage,
      buyerId,
      sellerId
    });

    res.json({ success: true, messageId: message.id });
  } catch (error) {
    console.error('Error creating booking notification:', error);
    res.status(500).json({ error: 'Failed to create booking notification' });
  }
});

/**
 * Endpoint for handling booking actions from chat
 */
router.post('/booking-action', authenticateJWT, async (req, res) => {
  try {
    const { bookingId, action } = req.body; // action: 'approve' or 'decline'
    const userId = req.user.id;

    // Call store-service to update booking status
    const storeApiUrl = process.env.STORE_API_URL;
    const response = await fetch(
      `${storeApiUrl}/items/booking-requests/${bookingId}/${action}`,
      {
        method: 'POST',
        headers: {
          'Authorization': req.headers.authorization,
          'Content-Type': 'application/json'
        }
      }
    );

    if (!response.ok) {
      throw new Error('Failed to update booking status');
    }

    const booking = await response.json();

    // Update chat message status
    await bookingMessageService.updateBookingMessageStatus(
      bookingId,
      booking.status,
      userId
    );

    res.json({ success: true, booking });
  } catch (error) {
    console.error('Error handling booking action:', error);
    res.status(500).json({ error: error.message });
  }
});

module.exports = router;
```

### 2.2 Store Service - Notify Chat on Booking Creation

**File:** `store-service/internal/services/booking_service.go`

```go
// Add after existing booking creation logic
func (s *BookingService) CreateBookingRequest(itemID, buyerID uint) (*models.BookingRequest, error) {
    // ... existing booking creation code ...

    booking, err := s.repo.Create(bookingRequest)
    if err != nil {
        return nil, err
    }

    // NEW: Notify chat service asynchronously
    go s.notifyChatService(booking, item)

    return booking, nil
}

func (s *BookingService) notifyChatService(booking *models.BookingRequest, item *models.StoreItem) {
    chatAPIURL := os.Getenv("CHAT_API_URL")
    internalAPIKey := os.Getenv("INTERNAL_API_KEY")

    payload := map[string]interface{}{
        "bookingId":  booking.ID,
        "itemId":     item.ID,
        "itemTitle":  item.Title,
        "itemImage":  item.Images[0], // First image
        "buyerId":    booking.BuyerID,
        "sellerId":   item.SellerID,
    }

    payloadBytes, _ := json.Marshal(payload)

    req, err := http.NewRequest(
        "POST",
        fmt.Sprintf("%s/internal/booking-created", chatAPIURL),
        bytes.NewBuffer(payloadBytes),
    )
    if err != nil {
        log.Printf("Error creating chat notification request: %v", err)
        s.markNotificationFailed(booking.ID)
        return
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Internal-API-Key", internalAPIKey)

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Error notifying chat service: %v", err)
        s.markNotificationFailed(booking.ID)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("Chat service returned non-OK status: %d", resp.StatusCode)
        s.markNotificationFailed(booking.ID)
        return
    }

    // Mark as successfully notified
    s.markNotificationSuccess(booking.ID)
}

func (s *BookingService) markNotificationSuccess(bookingID uint) {
    s.repo.UpdateChatNotificationStatus(bookingID, true, 0)
}

func (s *BookingService) markNotificationFailed(bookingID uint) {
    // Increment retry counter
    s.repo.IncrementNotificationAttempts(bookingID)
}
```

**File:** `store-service/internal/repositories/booking_repository.go`

```go
func (r *BookingRepository) UpdateChatNotificationStatus(bookingID uint, notified bool, attempts int) error {
    return r.db.Model(&models.BookingRequest{}).
        Where("id = ?", bookingID).
        Updates(map[string]interface{}{
            "chat_notified": notified,
            "notification_attempts": attempts,
            "last_notification_attempt": time.Now(),
        }).Error
}

func (r *BookingRepository) IncrementNotificationAttempts(bookingID uint) error {
    return r.db.Model(&models.BookingRequest{}).
        Where("id = ?", bookingID).
        Updates(map[string]interface{}{
            "notification_attempts": gorm.Expr("notification_attempts + 1"),
            "last_notification_attempt": time.Now(),
        }).Error
}
```

### 2.3 Background Job - Retry Failed Notifications

**File:** `store-service/internal/jobs/retry_notifications.go`

```go
// Runs every 5 minutes to retry failed chat notifications
func RetryFailedNotifications(db *gorm.DB) {
    var failedBookings []models.BookingRequest

    // Find bookings that failed to notify chat (max 3 attempts)
    db.Where("chat_notified = ? AND notification_attempts < ? AND status = ?",
        false, 3, "pending").
        Find(&failedBookings)

    for _, booking := range failedBookings {
        // Re-fetch item details
        var item models.StoreItem
        if err := db.First(&item, booking.ItemID).Error; err != nil {
            continue
        }

        // Retry notification
        bookingService := NewBookingService(db)
        bookingService.notifyChatService(&booking, &item)
    }
}
```

---

## Phase 3: Frontend Implementation

### 3.1 Update Message Interface

**File:** `frontend/src/stores/messages.ts`

```typescript
export interface Message {
  id: number
  task_id?: number
  application_id?: number
  item_id?: number
  store_item_id?: number
  sender_id: number
  recipient_id: number
  content: string
  message_type: 'text' | 'booking_request' | 'booking_approved' | 'booking_declined' | 'system_alert'
  metadata?: {
    booking_id?: number
    item_id?: number
    item_title?: string
    item_image?: string
    status?: 'pending' | 'approved' | 'declined'
  }
  is_read: boolean
  read_at?: string
  is_edited: boolean
  edited_at?: string
  is_deleted: boolean
  deleted_at?: string
  created_at: string
  has_removed_content?: boolean
  sender?: {
    id: number
    username: string
  }
  recipient?: {
    id: number
    username: string
  }
}
```

### 3.2 Create Booking Message Component

**File:** `frontend/src/components/messages/BookingMessageBubble.vue`

```vue
<template>
  <div class="booking-message" :class="messageTypeClass">
    <div class="booking-icon">
      <i :class="iconClass"></i>
    </div>

    <div class="booking-content">
      <!-- Booking Request -->
      <div v-if="message.message_type === 'booking_request'" class="booking-request">
        <div class="booking-header">
          <h4>Booking Request</h4>
          <span :class="statusBadgeClass">{{ statusText }}</span>
        </div>

        <div class="item-details">
          <img
            v-if="message.metadata?.item_image"
            :src="getImageUrl(message.metadata.item_image)"
            :alt="message.metadata?.item_title"
            class="item-thumbnail"
          />
          <div class="item-info">
            <p class="item-title">{{ message.metadata?.item_title }}</p>
            <p class="requester">
              {{ isOwnMessage ? 'You requested' : `${senderName} wants to book this item` }}
            </p>
          </div>
        </div>

        <!-- Action Buttons (only show for seller if pending) -->
        <div
          v-if="!isOwnMessage && message.metadata?.status === 'pending'"
          class="booking-actions"
        >
          <button
            @click="handleApprove"
            class="btn-approve"
            :disabled="isProcessing"
          >
            <i class="fas fa-check"></i> Approve
          </button>
          <button
            @click="handleDecline"
            class="btn-decline"
            :disabled="isProcessing"
          >
            <i class="fas fa-times"></i> Decline
          </button>
        </div>

        <!-- Status Message (if already decided) -->
        <div v-else-if="message.metadata?.status !== 'pending'" class="booking-status">
          <p v-if="message.metadata?.status === 'approved'" class="approved">
            ✅ Booking approved
          </p>
          <p v-else class="declined">
            ❌ Booking declined
          </p>
        </div>
      </div>

      <!-- Status Update Messages -->
      <div v-else class="booking-status-update">
        <p>{{ message.content }}</p>
      </div>

      <span class="timestamp">{{ formatTime(message.created_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useUserStore } from '@/stores/user';
import config from '@/config';
import type { Message } from '@/stores/messages';

const props = defineProps<{
  message: Message;
  isOwnMessage: boolean;
}>();

const emit = defineEmits<{
  bookingAction: [bookingId: number, action: 'approve' | 'decline'];
}>();

const userStore = useUserStore();
const isProcessing = ref(false);

const senderName = computed(() => {
  if (props.message.sender?.username) {
    return props.message.sender.username;
  }
  if (props.message.sender_id) {
    const user = userStore.getUserById(props.message.sender_id);
    if (user) return user.username;
  }
  return 'Unknown User';
});

const messageTypeClass = computed(() => {
  return `message-type-${props.message.message_type}`;
});

const iconClass = computed(() => {
  switch (props.message.message_type) {
    case 'booking_request':
      return 'fas fa-calendar-check';
    case 'booking_approved':
      return 'fas fa-check-circle';
    case 'booking_declined':
      return 'fas fa-times-circle';
    default:
      return 'fas fa-info-circle';
  }
});

const statusText = computed(() => {
  const status = props.message.metadata?.status;
  if (status === 'pending') return 'Pending';
  if (status === 'approved') return 'Approved';
  if (status === 'declined') return 'Declined';
  return '';
});

const statusBadgeClass = computed(() => {
  const status = props.message.metadata?.status;
  return `status-badge status-${status}`;
});

function getImageUrl(imagePath: string): string {
  if (imagePath.startsWith('http')) {
    return imagePath;
  }
  return `${config.STORE_API_URL}${imagePath}`;
}

async function handleApprove() {
  if (!props.message.metadata?.booking_id) return;
  isProcessing.value = true;
  emit('bookingAction', props.message.metadata.booking_id, 'approve');
}

async function handleDecline() {
  if (!props.message.metadata?.booking_id) return;
  isProcessing.value = true;
  emit('bookingAction', props.message.metadata.booking_id, 'decline');
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();

  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
}
</script>

<style scoped>
.booking-message {
  margin: 1rem 0;
  padding: 1rem;
  background: #f0f9ff;
  border-left: 4px solid #0284c7;
  border-radius: 0.5rem;
}

.message-type-booking_approved {
  background: #f0fdf4;
  border-left-color: #10b981;
}

.message-type-booking_declined {
  background: #fef2f2;
  border-left-color: #ef4444;
}

.booking-content {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.booking-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.booking-header h4 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #111827;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-approved {
  background: #d1fae5;
  color: #065f46;
}

.status-declined {
  background: #fee2e2;
  color: #991b1b;
}

.item-details {
  display: flex;
  gap: 1rem;
  padding: 0.75rem;
  background: white;
  border-radius: 0.375rem;
}

.item-thumbnail {
  width: 60px;
  height: 60px;
  object-fit: cover;
  border-radius: 0.25rem;
}

.item-info {
  flex: 1;
}

.item-title {
  margin: 0 0 0.25rem 0;
  font-weight: 600;
  color: #111827;
}

.requester {
  margin: 0;
  font-size: 0.875rem;
  color: #6b7280;
}

.booking-actions {
  display: flex;
  gap: 0.5rem;
}

.booking-actions button {
  flex: 1;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 0.375rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.booking-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-approve {
  background: #10b981;
  color: white;
}

.btn-approve:hover:not(:disabled) {
  background: #059669;
}

.btn-decline {
  background: #ef4444;
  color: white;
}

.btn-decline:hover:not(:disabled) {
  background: #dc2626;
}

.booking-status p {
  margin: 0;
  padding: 0.5rem;
  border-radius: 0.375rem;
  text-align: center;
  font-weight: 600;
}

.booking-status .approved {
  background: #d1fae5;
  color: #065f46;
}

.booking-status .declined {
  background: #fee2e2;
  color: #991b1b;
}

.timestamp {
  font-size: 0.75rem;
  color: #6b7280;
  text-align: right;
}

.booking-icon {
  font-size: 1.5rem;
  color: #0284c7;
  margin-bottom: 0.5rem;
}

.message-type-booking_approved .booking-icon {
  color: #10b981;
}

.message-type-booking_declined .booking-icon {
  color: #ef4444;
}
</style>
```

### 3.3 Update ChatWindow Component

**File:** `frontend/src/components/ChatWindow.vue`

Add the booking message component and handle booking actions:

```typescript
// Import the new component
import BookingMessageBubble from './messages/BookingMessageBubble.vue';

// Add method to handle booking actions
async function handleBookingAction(bookingId: number, action: 'approve' | 'decline') {
  try {
    const response = await fetch(`${config.CHAT_API_URL}/booking-action`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ bookingId, action })
    });

    if (!response.ok) {
      throw new Error('Failed to process booking action');
    }

    // The WebSocket will receive the updated message automatically
  } catch (error) {
    console.error('Error processing booking action:', error);
    alert('Failed to process booking action. Please try again.');
  }
}
```

Update template to render booking messages:

```vue
<template>
  <!-- ... existing chat UI ... -->

  <div class="messages-container">
    <div v-for="message in messages" :key="message.id">
      <!-- Booking Messages -->
      <BookingMessageBubble
        v-if="isBookingMessage(message)"
        :message="message"
        :is-own-message="message.sender_id === authStore.user?.id"
        @booking-action="handleBookingAction"
      />

      <!-- Regular Text Messages -->
      <MessageBubble
        v-else
        :message="message"
        :is-own-message="message.sender_id === authStore.user?.id"
        @edit="handleEdit"
        @delete="handleDelete"
      />
    </div>
  </div>
</template>

<script>
function isBookingMessage(message: Message): boolean {
  return ['booking_request', 'booking_approved', 'booking_declined'].includes(message.message_type);
}
</script>
```

### 3.4 Update Store Item Page

**File:** `frontend/src/views/store/StoreItemView.vue`

Update the "Book Now" button to redirect to chat after booking:

```typescript
async function handleBookNow() {
  try {
    isBooking.value = true;

    const response = await fetch(
      `${config.STORE_API_URL}/items/${item.value.id}/booking-request`,
      {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authStore.token}`,
          'Content-Type': 'application/json',
        }
      }
    );

    if (!response.ok) {
      throw new Error('Failed to create booking request');
    }

    const booking = await response.json();

    // NEW: Redirect to chat instead of showing success message
    router.push('/messages');

    // Optional: Show a toast notification
    // toast.success('Booking request sent! Check your messages.');

  } catch (error) {
    console.error('Error creating booking request:', error);
    alert('Failed to create booking request. Please try again.');
  } finally {
    isBooking.value = false;
  }
}
```

---

## Phase 4: Environment Configuration

### 4.1 Add New Environment Variables

**Chat Service `.env`:**
```env
# Existing vars...
CHAT_API_URL=http://localhost:8082/api/v1
STORE_API_URL=http://localhost:8081/api/v1

# NEW: Internal API security
INTERNAL_API_KEY=your-secure-random-key-here
```

**Store Service `.env`:**
```env
# Existing vars...
CHAT_API_URL=http://localhost:8082/api/v1

# NEW: Internal API security
INTERNAL_API_KEY=your-secure-random-key-here
```

**Docker Compose:**
```yaml
chat-websocket-service:
  environment:
    - INTERNAL_API_KEY=${INTERNAL_API_KEY}
    - STORE_API_URL=http://store-service:8081/api/v1

store-service:
  environment:
    - INTERNAL_API_KEY=${INTERNAL_API_KEY}
    - CHAT_API_URL=http://chat-websocket-service:8082/api/v1
```

---

## Phase 5: Testing Plan

### 5.1 Unit Tests

**Store Service:**
- Test booking creation triggers chat notification
- Test notification retry logic
- Test notification failure handling

**Chat Service:**
- Test booking message creation
- Test booking status updates
- Test message metadata handling

### 5.2 Integration Tests

1. **Full Booking Flow:**
   - Buyer books item → Verify system message created in chat
   - Seller approves → Verify both messages update
   - Both users see conversation in message center

2. **Notification Failures:**
   - Simulate chat service down → Verify retry logic
   - Verify booking still succeeds even if notification fails

3. **Edge Cases:**
   - Multiple booking requests for same item
   - Approve/decline after item deleted
   - Concurrent approve/decline clicks

### 5.3 E2E Tests (Playwright)

```typescript
test('buyer can book item and chat with seller', async ({ page, context }) => {
  // Login as buyer
  await page.goto('/store/item/123');
  await page.click('button:has-text("Book Now")');

  // Should redirect to messages
  await expect(page).toHaveURL('/messages');

  // Should see booking request message
  await expect(page.locator('.booking-message')).toBeVisible();
  await expect(page.locator('.booking-message')).toContainText('Booking Request');

  // Open new page as seller
  const sellerPage = await context.newPage();
  await sellerPage.goto('/messages');

  // Should see notification badge
  await expect(sellerPage.locator('.unread-badge')).toBeVisible();

  // Click conversation
  await sellerPage.click('.conversation-item:first-child');

  // Should see approve/decline buttons
  await expect(sellerPage.locator('button:has-text("Approve")')).toBeVisible();

  // Click approve
  await sellerPage.click('button:has-text("Approve")');

  // Both users should see approval message
  await expect(page.locator('text=Booking approved')).toBeVisible();
  await expect(sellerPage.locator('text=Booking approved')).toBeVisible();
});
```

---

## Phase 6: Migration & Deployment

### 6.1 Database Migration Steps

1. **Run migrations in production:**
   ```bash
   # Chat service
   cd chat-websocket-service
   npm run migrate

   # Store service
   cd store-service
   go run migrations/migrate.go up
   ```

2. **Verify migrations:**
   - Check message_type column exists
   - Check metadata column exists
   - Check booking notification columns exist

### 6.2 Deployment Sequence

1. **Deploy database migrations** (can run before code deploy)
2. **Deploy chat service** with new endpoints
3. **Deploy store service** with notification logic
4. **Deploy frontend** with new UI components
5. **Monitor logs** for notification failures

### 6.3 Rollback Plan

If issues occur:
1. Frontend: Revert to show old booking management UI
2. Store Service: Disable chat notifications (feature flag)
3. Chat Service: Old messages still work (backward compatible)

---

## Phase 7: Monitoring & Observability

### 7.1 Metrics to Track

- **Booking notification success rate**
- **Average notification retry attempts**
- **Time from booking to first seller response**
- **Booking approval rate via chat vs. item page**

### 7.2 Logging

```javascript
// Chat Service
logger.info('Booking notification received', {
  booking_id: bookingId,
  item_id: itemId,
  buyer_id: buyerId,
  seller_id: sellerId
});

// Store Service
log.Printf("Notifying chat service about booking %d (attempt %d)",
  booking.ID, attempts)
```

### 7.3 Alerts

- Alert if notification failure rate > 5%
- Alert if retry queue grows beyond 100 items
- Alert if chat service internal endpoint returns 5xx errors

---

## Success Criteria

✅ **User Experience:**
- Buyers are redirected to chat after booking
- Sellers receive notification badge for new bookings
- Sellers can approve/decline from chat interface
- Both parties see status updates in conversation

✅ **Technical:**
- 95%+ notification success rate
- Failed notifications retry successfully
- System messages display correctly
- WebSocket updates work in real-time

✅ **Performance:**
- Booking creation < 500ms (including notification)
- Chat notification delivery < 2 seconds
- No blocking on chat service failures

---

## Timeline Estimate

| Phase | Effort | Dependencies |
|-------|--------|--------------|
| Phase 1: Schema | 2 hours | None |
| Phase 2: Backend | 1 day | Phase 1 |
| Phase 3: Frontend | 1 day | Phase 2 |
| Phase 4: Config | 1 hour | None |
| Phase 5: Testing | 1 day | Phases 2-3 |
| Phase 6: Deploy | 2 hours | All phases |
| **Total** | **3-4 days** | - |

---

## Future Enhancements (Post-Implementation)

1. **Push Notifications:** Browser/mobile push for booking requests
2. **Email Notifications:** Fallback if seller doesn't check messages
3. **Auto-Decline:** Decline bookings after 48 hours of no response
4. **Booking Calendar:** Visual calendar for sellers with multiple bookings
5. **Quick Replies:** Pre-written seller responses ("Available today", "Sold")

---

## Related Documentation

- [RFC: Unified Booking & Messaging Flow](./RFC-booking-via-messages.md)
- [RFC: Unknown Sender Issue](./RFC-unknown-sender.md)
- [RFC: Conversation Titles](./RFC-conversation-titles.md)
- [CLAUDE.md - Architecture Overview](../../CLAUDE.md)
