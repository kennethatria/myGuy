# Complete Store Booking & Review Workflow

**Date:** January 6, 2026
**Status:** ✅ Documented
**Area:** Store Service - Complete Booking Workflow

---

## Overview

This document provides a comprehensive overview of the complete store item booking workflow, from initial booking request through delivery confirmation and mutual ratings/reviews.

---

## Architecture Components

### Backend Services

1. **Store Service (Go)** - Port 8081
   - Database: `my_guy_store`
   - Handles: Booking requests, status updates, ratings
   - Repository: `booking_request_repository.go`
   - Models: `BookingRequest` with rating fields

2. **Chat Service (Node.js)** - Port 8082
   - Database: `my_guy_chat`
   - Handles: Real-time notifications, message creation
   - Service: `bookingMessageService.js`

3. **Frontend (Vue 3)** - Port 5173
   - Components: `BookingMessageBubble.vue`, `ChatWidget.vue`, `StoreItemView.vue`
   - Stores: `chat.ts`, `auth.ts`

---

## Complete Workflow Steps

### Phase 1: Booking Request
**Actor:** Buyer

1. **Frontend:** Buyer views store item on `/store/:id` (StoreItemView.vue)
2. **Frontend:** Clicks "Book Now" button (line 100-107)
3. **Frontend:** Calls `sendBookingRequest()` (line 460-493)
   - POST to `/api/v1/items/:id/booking-request`
   - Shows confirmation modal
4. **Store Service:** Creates `BookingRequest` with status `pending` (store_handlers.go:420)
5. **Store Service:** Sends async notification to Chat Service
   - POST to `CHAT_API_URL/internal/booking-created`
   - Uses `INTERNAL_API_KEY` for auth
6. **Chat Service:** Creates booking message (bookingMessageService.js:6-58)
   - Message type: `booking_request`
   - Sender: buyer, Recipient: seller
   - Metadata includes: `booking_id`, `item_id`, `status: 'pending'`
7. **Chat Service:** Emits WebSocket event to seller
   - Event: `message:new`
   - Room: `user:${sellerId}`

**Result:** Seller sees booking request in Messages

---

### Phase 2: Seller Approval/Rejection
**Actor:** Seller

#### Option A: Approve

1. **Frontend:** Seller sees booking in messages (BookingMessageBubble.vue)
2. **Frontend:** Clicks "Approve" button (line 40-46)
3. **Frontend:** Emits `bookingAction` event with action: `approve`
4. **Frontend:** Parent calls `chatStore.handleBookingAction()`
5. **Chat Service:** Receives POST to `/booking-action`
   - Validates action and forwards to Store Service
6. **Store Service:** Updates booking status to `approved` (store_handlers.go:509)
   - POST `/booking-requests/:requestId/approve`
7. **Store Service:** Notifies Chat Service of status change
8. **Chat Service:** Updates message metadata (bookingMessageService.js:63-180)
   - Creates new message: `booking_approved`
   - Updates original message metadata: `status: 'approved'`
9. **Chat Service:** Emits WebSocket events to both users
   - Event: `message:new` (status update message)
   - Event: `message:updated` (original booking message)

**Result:** Both users see "Booking Approved" status

#### Option B: Decline

1-5. **Same as Approve** but action is `decline`
6. **Store Service:** Updates booking status to `rejected`
   - POST `/booking-requests/:requestId/reject`
7. **Chat Service:** Creates `booking_declined` message
8. **Result:** Booking is declined, no further actions possible

---

### Phase 3: Buyer Confirms Receipt
**Actor:** Buyer

1. **Frontend:** Buyer clicks "I Received Item" button (line 61-68)
2. **Frontend:** Emits `bookingAction` with action: `confirm-received`
3. **Chat Service:** Forwards to Store Service
4. **Store Service:** Updates booking status to `item_received`
   - POST `/booking-requests/:requestId/confirm-received`
   - Validates: only buyer can confirm, booking must be approved
5. **Chat Service:** Creates `booking_item_received` message
6. **Chat Service:** Updates message metadata: `status: 'item_received'`
7. **WebSocket:** Notifies both parties

**Result:** Seller sees "Item Received - Waiting for confirmation"

---

### Phase 4: Seller Confirms Delivery
**Actor:** Seller

1. **Frontend:** Seller clicks "Confirm Delivery" button (line 75-82)
2. **Frontend:** Emits `bookingAction` with action: `confirm-delivery`
3. **Chat Service:** Forwards to Store Service
4. **Store Service:** Updates booking status to `completed`
   - POST `/booking-requests/:requestId/confirm-delivery`
   - Validates: only seller can confirm, buyer must have confirmed receipt first
5. **Chat Service:** Creates `booking_completed` message
6. **Chat Service:** Updates message metadata: `status: 'completed'`
7. **WebSocket:** Notifies both parties

**Result:** Transaction marked as completed, rating UI appears

---

### Phase 5: Mutual Ratings/Reviews
**Actors:** Both Buyer and Seller

#### Buyer Rates Seller

1. **Frontend:** Rating UI appears when `status === 'completed'` (line 113)
2. **Frontend:** Buyer selects star rating (1-5) and optional review text
3. **Frontend:** Clicks "Submit Rating" button (line 136-142)
4. **Frontend:** Emits `bookingAction` with action: `rate-seller`
   - Includes: `rating`, `review` (optional)
5. **Chat Service:** Forwards to Store Service
6. **Store Service:** Updates booking record
   - POST `/booking-requests/:requestId/rate-seller`
   - Sets: `buyer_rating`, `buyer_review`
   - Validates: only buyer can rate seller, booking must be completed
7. **Store Service:** Returns updated booking
8. **Chat Service:** Updates message metadata with ratings
9. **WebSocket:** Notifies both parties
10. **Frontend:** UI switches to "rating display" mode (line 177-187)

#### Seller Rates Buyer

1-10. **Same process** but action is `rate-buyer`
- Store Service endpoint: `/booking-requests/:requestId/rate-buyer`
- Updates: `seller_rating`, `seller_review`

**Result:** Both users see each other's ratings in the message thread

---

## Data Flow Diagram

```
┌─────────────┐
│   Buyer     │
│  (Frontend) │
└─────┬───────┘
      │ 1. POST /items/:id/booking-request
      ▼
┌─────────────────┐
│  Store Service  │
│   (Port 8081)   │
└─────┬───────────┘
      │ 2. POST /internal/booking-created
      ▼                              ┌──────────────┐
┌─────────────────┐                 │   Seller     │
│  Chat Service   │◄────────────────┤  (Frontend)  │
│   (Port 8082)   │  3. WebSocket   └──────────────┘
└─────────────────┘     message:new

      │ 4. Seller clicks "Approve"
      ▼
┌─────────────────┐
│  Chat Service   │
│  POST /booking- │
│      action     │
└─────┬───────────┘
      │ 5. POST /booking-requests/:id/approve
      ▼
┌─────────────────┐
│  Store Service  │
│  Updates status │
└─────┬───────────┘
      │ 6. Returns success
      ▼
┌─────────────────┐
│  Chat Service   │
│  Emits WebSocket│
└─────────────────┘
      │
      ├──── message:new ────────► Buyer
      └──── message:updated ────► Seller
```

---

## API Endpoints Summary

### Store Service (Go)

| Endpoint | Method | Purpose | Auth | Status Transitions |
|----------|--------|---------|------|--------------------|
| `/api/v1/items/:id/booking-request` | POST | Create booking | JWT | → `pending` |
| `/api/v1/items/:id/booking-request` | GET | Get user's booking | JWT | Read status |
| `/api/v1/items/:id/booking-requests` | GET | Get all bookings (owner) | JWT | Read all |
| `/api/v1/booking-requests/:id/approve` | POST | Approve booking | JWT | `pending` → `approved` |
| `/api/v1/booking-requests/:id/reject` | POST | Reject booking | JWT | `pending` → `rejected` |
| `/api/v1/booking-requests/:id/confirm-received` | POST | Buyer confirms receipt | JWT | `approved` → `item_received` |
| `/api/v1/booking-requests/:id/confirm-delivery` | POST | Seller confirms delivery | JWT | `item_received` → `completed` |
| `/api/v1/booking-requests/:id/rate-seller` | POST | Buyer rates seller | JWT | `completed` (adds rating) |
| `/api/v1/booking-requests/:id/rate-buyer` | POST | Seller rates buyer | JWT | `completed` (adds rating) |

### Chat Service (Node.js)

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/internal/booking-created` | POST | Store notifies of new booking | Internal API Key |
| `/booking-action` | POST | Frontend initiates booking action | JWT |

---

## Database Schema

### Store Service - `booking_requests` Table

```sql
CREATE TABLE booking_requests (
  id                        SERIAL PRIMARY KEY,
  item_id                   INTEGER NOT NULL REFERENCES store_items(id),
  requester_id              INTEGER NOT NULL REFERENCES users(id),
  status                    VARCHAR DEFAULT 'pending',
  message                   TEXT,
  buyer_rating              INTEGER,      -- 1-5 stars
  buyer_review              TEXT,
  seller_rating             INTEGER,      -- 1-5 stars
  seller_review             TEXT,
  chat_notified             BOOLEAN DEFAULT FALSE,
  notification_attempts     INTEGER DEFAULT 0,
  last_notification_attempt TIMESTAMP,
  created_at                TIMESTAMP DEFAULT NOW(),
  updated_at                TIMESTAMP DEFAULT NOW()
);
```

**Status Flow:** `pending` → `approved` → `item_received` → `completed`
**Alt Flow:** `pending` → `rejected` (terminal)

### Chat Service - `messages` Table

```sql
CREATE TABLE messages (
  id             SERIAL PRIMARY KEY,
  sender_id      INTEGER NOT NULL,
  recipient_id   INTEGER NOT NULL,
  store_item_id  INTEGER,
  message_type   VARCHAR, -- 'booking_request', 'booking_approved', etc.
  content        TEXT,
  metadata       JSONB,   -- { booking_id, item_id, status, ratings, etc. }
  created_at     TIMESTAMP DEFAULT NOW()
);
```

**Metadata Example (completed with ratings):**
```json
{
  "booking_id": 123,
  "item_id": 456,
  "item_title": "iPhone 13",
  "item_image": "/uploads/store/image.jpg",
  "status": "completed",
  "buyer_rating": 5,
  "buyer_review": "Great seller, fast shipping!",
  "seller_rating": 4,
  "seller_review": "Good buyer, prompt payment."
}
```

---

## Frontend Components

### StoreItemView.vue
- **Path:** `frontend/src/views/store/StoreItemView.vue`
- **Purpose:** Display item details, initiate booking
- **Key Functions:**
  - `sendBookingRequest()` - Creates booking
  - `loadBookingRequest()` - Fetches booking status

### BookingMessageBubble.vue
- **Path:** `frontend/src/components/messages/BookingMessageBubble.vue`
- **Purpose:** Display booking messages and actions
- **Key Functions:**
  - `handleApprove()` - Approve booking
  - `handleDecline()` - Reject booking
  - `handleConfirmReceived()` - Buyer confirms receipt
  - `handleConfirmDelivery()` - Seller confirms delivery
  - `submitRating()` - Submit rating/review
- **Computed:**
  - `hasRated` - Check if user has submitted rating
  - `displayedRating` - Show rating (buyer's or seller's)

### ChatWidget.vue / MessageThread.vue
- **Path:** `frontend/src/components/messages/`
- **Purpose:** Display message threads, handle booking actions
- **Key Functions:**
  - `handleBookingAction()` - Delegate to chat store

### Chat Store
- **Path:** `frontend/src/stores/chat.ts`
- **Purpose:** Manage WebSocket connection, API calls
- **Key Functions:**
  - `handleBookingAction()` - POST to `/booking-action`
  - WebSocket listeners for `message:new`, `message:updated`

---

## Current Issues

### Critical Issues

1. **[P1] Review Button Stays Disabled**
   - **FIXLOG:** `FIXLOG-booking-review-button-disabled.md`
   - **Impact:** Users cannot submit reviews without refresh
   - **Root Cause:** `isProcessing` state never resets in BookingMessageBubble
   - **Affects:** All booking actions (approve, decline, confirm, rate)

### Potential Improvements

1. **Add Loading States**
   - Show spinner when action is processing
   - Provide visual feedback that action is being handled

2. **Error Handling**
   - Display error messages if action fails
   - Allow retry on network errors

3. **Optimistic Updates**
   - Update UI immediately when user clicks action
   - Roll back if server rejects

4. **Rating Analytics**
   - Track average ratings per user
   - Display seller/buyer reputation

5. **Email Notifications**
   - Notify seller of new booking request
   - Notify buyer when booking approved
   - Remind users to leave reviews

6. **Dispute Resolution**
   - Add "Report Problem" option
   - Allow admins to intervene in disputes

---

## Testing Scenarios

### Happy Path
1. Buyer requests booking → ✅ Seller sees request
2. Seller approves → ✅ Both see approval
3. Buyer confirms receipt → ✅ Seller sees confirmation
4. Seller confirms delivery → ✅ Status becomes 'completed'
5. Both submit ratings → ✅ Ratings visible to both parties

### Edge Cases
- [ ] Multiple booking requests for same item
- [ ] Buyer cancels after approval (not implemented)
- [ ] Seller never responds (timeout not implemented)
- [ ] Network failure during action
- [ ] WebSocket disconnected during flow
- [ ] User closes browser mid-flow
- [ ] Item deleted while booking active

### Error Scenarios
- [ ] Booking when item already sold
- [ ] Rating before delivery confirmed
- [ ] Double-clicking action buttons
- [ ] Submitting rating twice
- [ ] Invalid rating value (< 1 or > 5)

---

## Deployment Checklist

Before deploying booking/review features:
- [ ] Fix P1 issue: Review button disabled bug
- [ ] Test all status transitions
- [ ] Verify WebSocket reliability
- [ ] Test with slow network conditions
- [ ] Ensure `INTERNAL_API_KEY` is set in both services
- [ ] Verify `JWT_SECRET` matches across all services
- [ ] Test email notifications (if implemented)
- [ ] Monitor error rates after deployment
- [ ] Set up analytics for review completion rate

---

## Related Documentation

- `CLAUDE.md` - System overview
- `FIXLOG-booking-review-button-disabled.md` - Critical bug details
- `engineering/03-completed/FIXLOG-booking-workflow-gaps.md` - Previous booking fixes
- `engineering/03-completed/IMPLEMENTATION-unified-booking-frontend.md` - Frontend implementation

---

## Monitoring & Metrics

**Key Metrics to Track:**
1. **Booking Request Rate** - Requests created per day
2. **Approval Rate** - % of bookings approved
3. **Completion Rate** - % of approved bookings completed
4. **Review Submission Rate** - % of completed bookings with reviews
5. **Average Rating** - Mean rating for sellers and buyers
6. **Time to Complete** - Average time from booking to completion

**Alerts:**
- Review completion rate drops below 50%
- Approval rate drops significantly
- Error rate on booking endpoints exceeds 5%
- WebSocket disconnection rate increases

---

**Last Updated:** January 6, 2026
**Next Review:** After P1 bug fix is deployed
