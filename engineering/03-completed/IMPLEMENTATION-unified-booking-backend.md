# Implementation Log: Unified Booking & Messaging Flow - Backend

**Status:** ✅ **BACKEND COMPLETED** - January 4, 2026
**Frontend Status:** ⏸️ Pending (awaits backend deployment)

---

## Summary

Implemented the backend foundation for unified booking and messaging flow. Sellers will now receive booking requests as system messages in their chat interface, with approve/decline buttons directly in the conversation.

---

## Changes Implemented

### 1. Database Migrations

#### Chat Service (`chat-websocket-service/migrations/002_add_booking_support.sql`)
- ✅ Added `metadata` JSONB column to `messages` table
- ✅ Created indexes on `metadata->>'booking_id'` and `metadata->>'item_id'`
- ✅ Updated message_type support for: `text`, `booking_request`, `booking_approved`, `booking_declined`, `system_alert`

#### Store Service (`store-service/migrations/003_add_booking_notification_tracking.sql`)
- ✅ Added `chat_notified` boolean column to `booking_requests`
- ✅ Added `notification_attempts` integer column for retry tracking
- ✅ Added `last_notification_attempt` timestamp column
- ✅ Created index for querying failed notifications

### 2. Chat Service Implementation

#### New Files Created:
1. **`src/services/bookingMessageService.js`** (141 lines)
   - `createBookingRequestMessage()` - Creates system message for new bookings
   - `updateBookingMessageStatus()` - Updates message when seller approves/declines
   - Emits WebSocket events to both buyer and seller

2. **`src/api/bookingNotifications.js`** (104 lines)
   - `POST /api/v1/internal/booking-created` - Internal endpoint for store-service
   - `POST /api/v1/booking-action` - Public endpoint for approve/decline actions
   - Secured with INTERNAL_API_KEY for service-to-service calls

#### Modified Files:
- **`src/server.js`**
  - Registered booking notification routes
  - Made `io` instance available to routes for WebSocket emissions

### 3. Store Service Implementation

#### New Files Created:
1. **`internal/services/chat_notification.go`** (101 lines)
   - `NotifyChatServiceAboutBooking()` - Async function to notify chat service
   - `ChatNotificationPayload` struct for API communication
   - Retry logic with attempt tracking
   - Graceful failure handling (booking succeeds even if notification fails)

#### Modified Files:
1. **`internal/models/store_item.go`**
   - Added `ChatNotified` field to BookingRequest model
   - Added `NotificationAttempts` field
   - Added `LastNotificationAttempt` field

2. **`internal/repositories/booking_request_repository.go`**
   - Added `UpdateChatNotificationStatus()` method
   - Added `IncrementNotificationAttempts()` method

3. **`internal/repositories/interfaces.go`**
   - Added new methods to `BookingRequestRepository` interface

4. **`internal/services/store_service.go`**
   - Updated `CreateBookingRequest()` to call notification function
   - Notification runs asynchronously (non-blocking)

### 4. Environment Configuration

#### Chat Service (`.env.example`)
```env
STORE_API_URL=http://localhost:8081/api/v1
INTERNAL_API_KEY=your-internal-api-key-change-in-production
```

#### Store Service (`.env.example`)
```env
CHAT_API_URL=http://localhost:8082/api/v1
INTERNAL_API_KEY=your-internal-api-key-change-in-production
```

---

## API Endpoints Added

### Chat Service

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/internal/booking-created` | POST | Internal API Key | Receive booking notifications from store-service |
| `/api/v1/booking-action` | POST | JWT | Handle approve/decline actions from chat UI |

**POST /internal/booking-created Request:**
```json
{
  "bookingId": 123,
  "itemId": 456,
  "itemTitle": "Red Bicycle",
  "itemImage": "/uploads/store/bicycle.jpg",
  "buyerId": 10,
  "sellerId": 20
}
```

**POST /booking-action Request:**
```json
{
  "bookingId": 123,
  "action": "approve"  // or "decline"
}
```

---

## Message Structure

### Booking Request Message
```json
{
  "id": 789,
  "sender_id": 10,
  "recipient_id": 20,
  "store_item_id": 456,
  "message_type": "booking_request",
  "content": "Booking request for Red Bicycle",
  "metadata": {
    "booking_id": 123,
    "item_id": 456,
    "item_title": "Red Bicycle",
    "item_image": "/uploads/store/bicycle.jpg",
    "status": "pending"
  },
  "created_at": "2026-01-04T10:30:00Z"
}
```

### Booking Approved Message
```json
{
  "id": 790,
  "sender_id": 20,
  "recipient_id": 10,
  "store_item_id": 456,
  "message_type": "booking_approved",
  "content": "Booking approved ✅. You can now discuss pickup details.",
  "metadata": {
    "booking_id": 123,
    "item_id": 456,
    "status": "approved"
  },
  "created_at": "2026-01-04T10:35:00Z"
}
```

---

## Flow Diagram

```
┌──────────────┐
│   Buyer      │
│ POST /items/ │
│ :id/booking- │
│ request      │
└──────┬───────┘
       │
       ↓
┌─────────────────────────────────┐
│  Store Service                  │
│  1. Create booking record       │
│  2. Return booking to buyer     │
│  3. Async: Notify chat service  │
└──────┬──────────────────────────┘
       │ POST /internal/booking-created
       │ (Headers: X-Internal-API-Key)
       ↓
┌─────────────────────────────────┐
│  Chat Service                   │
│  1. Create system message       │
│  2. Emit via WebSocket          │
└──────┬──────────────────────────┘
       │ WebSocket: message:new
       ↓
┌──────────────┐
│   Seller     │
│ Sees booking │
│ request in   │
│ chat with    │
│ [Approve]    │
│ [Decline]    │
│ buttons      │
└──────┬───────┘
       │ POST /booking-action
       │ {bookingId, action}
       ↓
┌─────────────────────────────────┐
│  Chat Service                   │
│  1. Forward to store-service    │
│  2. Update message status       │
│  3. Create status message       │
│  4. Emit to both users          │
└─────────────────────────────────┘
       │ POST /items/booking-requests/:id/approve
       ↓
┌─────────────────────────────────┐
│  Store Service                  │
│  Update booking status in DB    │
└─────────────────────────────────┘
```

---

## Error Handling & Resilience

### Graceful Degradation
- ✅ **Booking creation succeeds** even if chat notification fails
- ✅ **Retry mechanism** tracks failed notifications
- ✅ **Async execution** prevents blocking user requests
- ✅ **5-second timeout** on HTTP calls to prevent hanging

### Notification Retry Logic
1. Store service attempts to notify chat service
2. If failure: `notification_attempts` incremented, `chat_notified` stays false
3. Background job can later query for `chat_notified = false` and retry
4. Max 3 attempts (configurable)

---

## Testing Checklist

### Unit Tests Needed:
- [ ] `bookingMessageService.createBookingRequestMessage()`
- [ ] `bookingMessageService.updateBookingMessageStatus()`
- [ ] `NotifyChatServiceAboutBooking()` (with mocked HTTP client)
- [ ] Repository methods for notification tracking

### Integration Tests Needed:
- [ ] Full booking flow: create → notify → approve
- [ ] Notification failure handling
- [ ] WebSocket emission verification
- [ ] Internal API key authentication

### Manual Testing Steps:
1. Start all services with `docker-compose up`
2. Create a booking request via Postman/curl
3. Verify system message appears in chat DB
4. Verify WebSocket emission logged
5. Send approve/decline action
6. Verify both services update correctly

---

## Deployment Steps

1. **Run Database Migrations:**
   ```bash
   # Chat service
   cd chat-websocket-service
   npm run migrate

   # Store service migrations run automatically on startup (GORM AutoMigrate)
   ```

2. **Set Environment Variables:**
   - Generate secure `INTERNAL_API_KEY` (e.g., `openssl rand -hex 32`)
   - Set same key in both chat and store services
   - Set `CHAT_API_URL` in store service
   - Set `STORE_API_URL` in chat service

3. **Deploy Services:**
   ```bash
   docker-compose down
   docker-compose up --build
   ```

4. **Verify:**
   - Check chat service logs for "Booking notification routes registered"
   - Create test booking and check logs for notification success

---

## Files Modified

### Chat Service:
- ✅ `migrations/002_add_booking_support.sql` (new)
- ✅ `src/services/bookingMessageService.js` (new)
- ✅ `src/api/bookingNotifications.js` (new)
- ✅ `src/server.js` (modified - added routes)
- ✅ `.env.example` (modified - added INTERNAL_API_KEY, STORE_API_URL)

### Store Service:
- ✅ `migrations/003_add_booking_notification_tracking.sql` (new)
- ✅ `internal/services/chat_notification.go` (new)
- ✅ `internal/models/store_item.go` (modified - BookingRequest fields)
- ✅ `internal/repositories/booking_request_repository.go` (modified - new methods)
- ✅ `internal/repositories/interfaces.go` (modified - interface update)
- ✅ `internal/services/store_service.go` (modified - CreateBookingRequest)
- ✅ `.env.example` (modified - added INTERNAL_API_KEY, CHAT_API_URL)

**Total:** 12 files (5 new, 7 modified)
**Lines Added:** ~400 lines of code

---

## Next Steps (Frontend)

To complete this feature, frontend implementation is needed:
1. Update `Message` interface to include `message_type` and `metadata`
2. Create `BookingMessageBubble.vue` component
3. Update `ChatWindow.vue` to render booking messages
4. Implement approve/decline button handlers
5. Handle WebSocket events for booking status updates

See `engineering/01-proposed/PLAN-unified-booking-messaging.md` Phase 3 for details.

---

## Security Considerations

✅ **Internal API secured** with shared secret key
✅ **JWT authentication** on public booking-action endpoint
✅ **Authorization checks** in store service (verify user is seller)
⚠️ **Production:** Change INTERNAL_API_KEY from default
⚠️ **Production:** Use HTTPS for inter-service communication

---

## Performance Impact

- **Minimal:** Async notification doesn't block booking creation
- **HTTP call overhead:** ~50-100ms for notification (non-blocking)
- **Database writes:** +1 system message per booking (acceptable)
- **WebSocket:** Real-time delivery (no polling needed)

---

## Related Documentation

- Implementation Plan: `engineering/01-proposed/PLAN-unified-booking-messaging.md`
- RFC: `engineering/01-proposed/RFC-booking-via-messages.md`
- User Flow: `engineering/01-proposed/USER-FLOW-booking-summary.md`
- MVP Roadmap: `engineering/01-proposed/ROADMAP-mvp-prioritization.md`
