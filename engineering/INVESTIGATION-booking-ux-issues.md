# INVESTIGATION: Booking UX Issues

**Date:** January 6, 2026
**Status:** 🔴 **ISSUES IDENTIFIED**
**Priority:** P2
**Area:** Frontend - Store Booking UX

---

## Issues Identified

### Issue #1: Booking Request Message Not Captured
**Location:** `frontend/src/views/store/StoreItemView.vue` line 460-493

**Problem:**
- When user clicks "Book Now", they're shown a confirmation modal with a text field
- User expects to write a message BEFORE sending the booking request
- Currently, a hardcoded message is sent: `"I'm interested in booking this item: ${item.value.title}"`
- The text field in BookingConfirmationModal is for AFTER booking is sent, not part of initial request

**User Flow (Current):**
1. User clicks "Book Now" button
2. `sendBookingRequest()` is called immediately
3. Hardcoded message sent to backend
4. BookingConfirmationModal opens AFTER request is sent
5. User can then send additional messages (but initial request already sent)

**Expected Flow:**
1. User clicks "Book Now" button
2. Modal opens with textarea for custom message
3. User writes custom message
4. User clicks "Send Request" in modal
5. Custom message included in booking request

---

### Issue #2: Modal Doesn't Close After Sending Message
**Location:** `frontend/src/components/BookingConfirmationModal.vue` line 213-244

**Problem:**
- When user sends a message in the modal, the modal stays open
- User expects modal to close automatically after sending
- Forces user to manually click "Stay on Page" or "View in Messages"

**Code:**
```typescript
async function sendMessage() {
  if (!messageText.value.trim() || sending.value) return;

  sending.value = true;

  try {
    await chatStore.sendStoreMessage(
      messageText.value,
      props.sellerId,
      props.itemId
    );

    // Clear input after sending
    messageText.value = '';

    // Refresh messages
    const allMessages = chatStore.getStoreMessages(props.itemId);
    messages.value = allMessages.filter(/* ... */);
  } catch (err) {
    console.error('Error sending message:', err);
    alert('Failed to send message. Please try again.');
  } finally {
    sending.value = false;
  }
  // ❌ Missing: close() call here
}
```

---

### Issue #3: "Message Seller" Button Active After Transaction Complete
**Location:** `frontend/src/views/store/StoreItemView.vue` line 62-68

**Problem:**
- "Message Seller" button is visible even when booking is completed
- Should be disabled or hidden once transaction is finalized
- Other users can still message seller about a sold item

**Current Code:**
```vue
<button
  v-if="item.seller.id !== userId"
  @click="openStoreChat"
  class="btn btn-outline btn-sm message-btn"
>
  <i class="fas fa-comment"></i> Message Seller
</button>
```

**Missing Check:**
- Should check if user has a completed booking
- Should disable button if item is sold to this user
- Should show appropriate message (e.g., "Transaction Completed")

---

## Solutions

### Fix #1: Add Message Input BEFORE Sending Booking Request

**Option A: Add to Existing Modal (Recommended)**
- Show BookingConfirmationModal BEFORE sending request
- Add "message" input field above the chat section
- Only send booking request when user clicks "Send Request" button
- Include custom message in request payload

**Option B: Create Separate Booking Request Modal**
- New modal specifically for creating booking request
- Simple form: Item info + Message textarea + Send button
- Show confirmation modal AFTER request is sent

**Recommendation:** Option A - Reuse existing modal but reverse the flow

---

### Fix #2: Close Modal After Sending Message

**Simple Fix:**
```typescript
async function sendMessage() {
  // ... existing code ...

  try {
    await chatStore.sendStoreMessage(/* ... */);

    messageText.value = '';

    // Refresh messages
    const allMessages = chatStore.getStoreMessages(props.itemId);
    messages.value = allMessages.filter(/* ... */);

    // NEW: Close modal after successful send
    setTimeout(() => {
      close();
    }, 500); // Small delay so user sees the message was sent
  } catch (err) {
    // ... error handling ...
  }
}
```

**Alternative:** Add a setting/preference for this behavior

---

### Fix #3: Disable "Message Seller" for Completed Bookings

**Approach:**
1. Check if current user has a booking for this item
2. Check if booking status is 'completed' or 'item_received'
3. If yes, disable button and show "Transaction Completed" text

**Implementation:**
```vue
<div v-if="item.seller.id !== userId" class="seller-message-section">
  <button
    v-if="!hasCompletedBooking"
    @click="openStoreChat"
    class="btn btn-outline btn-sm message-btn"
  >
    <i class="fas fa-comment"></i> Message Seller
  </button>
  <div v-else class="transaction-complete-badge">
    <i class="fas fa-check-circle"></i>
    Transaction Completed
  </div>
</div>
```

**Computed Property:**
```typescript
const hasCompletedBooking = computed(() => {
  if (!bookingRequest.value) return false;
  const status = bookingRequest.value.status;
  return status === 'completed' || status === 'item_received';
});
```

---

## Implementation Plan

### Step 1: Fix Modal Close on Message Send (Easiest)
- File: `BookingConfirmationModal.vue`
- Add `close()` call after successful message send
- Testing: Send message → Modal should close

### Step 2: Disable Message Button for Completed Transactions
- File: `StoreItemView.vue`
- Add computed property `hasCompletedBooking`
- Update template to show badge instead of button
- Testing: Complete transaction → Button should be replaced with badge

### Step 3: Capture Custom Message in Booking Request (Complex)
- Files: `StoreItemView.vue`, `BookingConfirmationModal.vue`
- Requires refactoring the booking flow
- Two approaches:
  - **A:** Show modal before sending request (requires state management)
  - **B:** Add optional message field to booking request (simpler)

**Recommendation for Step 3:**
Start with Option B (simpler):
- Keep current flow (booking sent immediately)
- Add textarea for initial message on StoreItemView before clicking "Book Now"
- Pass message to `sendBookingRequest(message)`
- Modal shows after request sent (as currently)

---

## Priority Order

1. **P1: Fix #2** - Close modal after sending message (5 min fix)
2. **P1: Fix #3** - Disable button for completed transactions (10 min fix)
3. **P2: Fix #1** - Capture custom booking message (30 min refactor)

---

**Status:** Ready for implementation
