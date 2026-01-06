# INVESTIGATION: Duplicate "Transaction completed!" Messages

**Date:** January 6, 2026
**Status:** 🔴 **BUG IDENTIFIED**
**Priority:** P2
**Area:** Chat Service - Booking Workflow

---

## Problem Statement

Users are seeing the message "Transaction completed! Both parties have confirmed." repeated multiple times in the chat when completing a store item transaction.

---

## Root Cause Analysis

### The Bug

**Location:** `chat-websocket-service/src/services/bookingMessageService.js` lines 157-160

```javascript
// Emit to both users via WebSocket
if (io) {
  io.to(`user:${requestMessage.sender_id}`).emit('message:new', statusMessage);
  io.to(`user:${approverId}`).emit('message:new', statusMessage);
```

**The Problem:**

The code emits the message to TWO user rooms:
1. `user:${requestMessage.sender_id}` - The original booking requester (BUYER)
2. `user:${approverId}` - The person who triggered the current action

### When Duplicates Occur

#### Scenario 1: Buyer Confirms Item Received (item_received status)
- `requestMessage.sender_id` = BUYER (original requester)
- `approverId` = BUYER (person confirming receipt)
- **Result:** Message sent to `user:${BUYER}` TWICE ❌

Flow:
```
Buyer clicks "I Received Item"
→ Chat service calls updateBookingMessageStatus(bookingId, 'item_received', BUYER_ID, ...)
→ Creates message: "📦 Buyer confirmed they received the item."
→ Emits to user:${BUYER} (sender_id)  ← BUYER gets it
→ Emits to user:${BUYER} (approverId) ← BUYER gets it AGAIN
```

#### Scenario 2: Seller Confirms Delivery (completed status)
- `requestMessage.sender_id` = BUYER (original requester)
- `approverId` = SELLER (person confirming delivery)
- **Result:** Each user gets message once ✅ (This is correct)

Flow:
```
Seller clicks "Confirm Delivery"
→ Chat service calls updateBookingMessageStatus(bookingId, 'completed', SELLER_ID, ...)
→ Creates message: "✅ Transaction completed! Both parties have confirmed."
→ Emits to user:${BUYER} (sender_id)  ← BUYER gets it
→ Emits to user:${SELLER} (approverId) ← SELLER gets it
```

#### Scenario 3: Seller Approves Booking (approved status)
- `requestMessage.sender_id` = BUYER (original requester)
- `approverId` = SELLER (person approving)
- **Result:** Each user gets message once ✅ (This is correct)

#### Scenario 4: Seller Rejects Booking (rejected status)
- `requestMessage.sender_id` = BUYER (original requester)
- `approverId` = SELLER (person rejecting)
- **Result:** Each user gets message once ✅ (This is correct)

### Summary

**Affected Actions:**
1. ❌ **Buyer confirms item received** - Buyer gets duplicate message
2. ❌ **Buyer rates seller** - Buyer gets duplicate message (if rating after completion)
3. ✅ **Seller approves** - No duplicate
4. ✅ **Seller rejects** - No duplicate
5. ✅ **Seller confirms delivery** - No duplicate
6. ❌ **Seller rates buyer** - Seller gets duplicate message (if rating after completion)

Wait, let me reconsider... When is the "Transaction completed!" message created?

Looking at the code (lines 122-124):
```javascript
} else if (status === 'completed') {
  messageType = 'booking_completed';
  content = '✅ Transaction completed! Both parties have confirmed.';
}
```

This is triggered when status = 'completed', which happens when the SELLER confirms delivery (only the seller can confirm delivery).

So for "Transaction completed!" specifically:
- Triggered by: Seller confirming delivery
- `approverId` = SELLER
- `requestMessage.sender_id` = BUYER
- **Result:** Message sent once to each user ✅

But the user reported seeing it repeated "a couple of times"...

### Alternative Theory: Multiple Status Updates

What if the flow is:
1. Seller clicks "Confirm Delivery"
2. Store service updates status to 'completed'
3. Chat service creates "Transaction completed" message
4. Then a rating is submitted
5. Chat service is called AGAIN with status 'completed' (status hasn't changed)
6. Creates another "Transaction completed" message?

Let me check if there's duplicate calling...

Actually, looking more carefully at the user's report: "this is triggered while seller and buyer are accepting requests about item"

This sounds like it might be happening during:
- Multiple booking requests for the same item?
- Or multiple actions in quick succession?

### Most Likely Root Cause

The real issue is probably:

**When ratings are submitted after completion**, the chat service is called with the booking status 'completed' (because that's the current status in the booking record), which triggers creation of ANOTHER "Transaction completed" message.

Flow:
1. Seller confirms delivery → Status becomes 'completed' → Message created ✅
2. Buyer rates seller → Status is still 'completed' → Message created AGAIN ❌
3. Seller rates buyer → Status is still 'completed' → Message created AGAIN ❌

Let me verify by checking what happens when ratings are submitted...

Looking at `/booking-action` endpoint (lines 119-125):
```javascript
const booking = await response.json();

// Get io instance from app
const io = req.app.get('io');

// Update chat message status (pass full booking object for ratings)
await bookingMessageService.updateBookingMessageStatus(
  bookingId,
  booking.status,  // ← This is 'completed' when rating
  userId,
  io,
  booking
);
```

**BINGO!** When a user submits a rating:
1. Store service updates the rating fields but status stays 'completed'
2. Chat service receives booking with status='completed'
3. Calls `updateBookingMessageStatus(bookingId, 'completed', ...)`
4. This creates a NEW "Transaction completed" message EVERY TIME

So:
- Seller confirms delivery → 1st "Transaction completed" message
- Buyer rates seller → 2nd "Transaction completed" message
- Seller rates buyer → 3rd "Transaction completed" message

---

## Solution

### Option 1: Don't Create Status Message for Unchanged Status

Modify `updateBookingMessageStatus` to check if the status actually changed before creating a new message.

**Implementation:**
1. Query for existing status messages for this booking
2. Check if the most recent status message already has this status
3. Only create a new message if status changed

**Pros:**
- Prevents all duplicate status messages
- Clean and logical

**Cons:**
- Requires additional database query

### Option 2: Special Handling for Rating Actions

Don't call `updateBookingMessageStatus` for rating actions, only update the metadata on the original booking message.

**Implementation:**
1. In `/booking-action` endpoint, check if action is 'rate-seller' or 'rate-buyer'
2. For ratings: Only update the original booking message metadata, don't create status message
3. For other actions: Call `updateBookingMessageStatus` as normal

**Pros:**
- Simple and targeted fix
- No additional queries needed

**Cons:**
- Special case logic

### Option 3: Track Status in Message Metadata

Store the last emitted status in the booking message metadata, only create new message if different.

**Implementation:**
1. When creating status message, store `last_status` in booking request message metadata
2. Before creating new status message, check if `last_status === new_status`
3. Skip creation if they match

**Pros:**
- No additional database table needed
- Self-contained solution

**Cons:**
- Metadata becomes stateful

---

## Recommended Fix: Option 2

Modify the `/booking-action` endpoint to NOT call `updateBookingMessageStatus` for rating actions.

Instead, for ratings:
1. Update the original booking request message metadata directly
2. Emit `message:updated` event to both users
3. No new status message created

This is the cleanest fix because:
- Ratings don't change the booking status
- Ratings already update the original message metadata
- No need for a status change notification when only ratings change

---

## Implementation Plan

### File to Modify
`chat-websocket-service/src/api/bookingNotifications.js`

### Changes Needed

```javascript
// After fetching booking from store service (line 113)
const booking = await response.json();

// Get io instance from app
const io = req.app.get('io');

// For rating actions, only update metadata without creating status message
if (action === 'rate-seller' || action === 'rate-buyer') {
  // Find the original booking request message
  const findResult = await db.query(
    `SELECT * FROM messages
     WHERE message_type = 'booking_request'
     AND metadata->>'booking_id' = $1
     LIMIT 1`,
    [bookingId.toString()]
  );

  if (findResult.rows.length > 0) {
    const requestMessage = findResult.rows[0];

    // Update metadata with ratings
    const updatedMetadata = {
      ...requestMessage.metadata,
      status: booking.status
    };

    if (booking.buyer_rating) updatedMetadata.buyer_rating = booking.buyer_rating;
    if (booking.buyer_review) updatedMetadata.buyer_review = booking.buyer_review;
    if (booking.seller_rating) updatedMetadata.seller_rating = booking.seller_rating;
    if (booking.seller_review) updatedMetadata.seller_review = booking.seller_review;

    // Update the message
    await db.query(
      `UPDATE messages SET metadata = $1 WHERE id = $2`,
      [JSON.stringify(updatedMetadata), requestMessage.id]
    );

    // Emit update to both users
    const updatedMessage = { ...requestMessage, metadata: updatedMetadata };
    io.to(`user:${requestMessage.sender_id}`).emit('message:updated', updatedMessage);
    io.to(`user:${booking.item.seller_id}`).emit('message:updated', updatedMessage);

    console.log(`✅ Rating submitted for booking ${bookingId} - metadata updated`);
  }
} else {
  // For non-rating actions, create status message as normal
  await bookingMessageService.updateBookingMessageStatus(
    bookingId,
    booking.status,
    userId,
    io,
    booking
  );
}
```

---

## Testing Checklist

After fix:
- [ ] Seller approves → 1 status message ✅
- [ ] Buyer confirms received → 1 status message ✅
- [ ] Seller confirms delivery → 1 "Transaction completed" message ✅
- [ ] Buyer rates seller → 0 new messages, metadata updated ✅
- [ ] Seller rates buyer → 0 new messages, metadata updated ✅
- [ ] Verify ratings still display correctly in UI
- [ ] Verify both users see updated ratings in real-time

Expected total messages for complete flow:
1. Booking request (initial)
2. Booking approved
3. Item received confirmation
4. Transaction completed
5. **NO additional messages for ratings**

---

## Related Issues

- Original fix for button disabled state (already completed)
- This is a separate issue in the notification logic

---

**Status:** Ready for implementation
**Priority:** P2 (not critical, but creates poor UX)
