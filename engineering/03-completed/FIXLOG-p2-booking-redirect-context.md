# Fix Log: "Book Now" Redirect Context Loss (P2)

**Date:** January 5, 2026
**Priority:** P2
**Status:** ✅ COMPLETED
**Related:** [INVESTIGATION-message-seller-functionality.md](../01-proposed/INVESTIGATION-message-seller-functionality.md)

---

## Problem Statement

After clicking "Book Now" on a store item, users were redirected to `/messages` without any context. The Message Center had no way of knowing which conversation to open, forcing users to manually search for the newly created booking conversation.

### Impact
- Poor booking UX - buyers completed the booking action but landed on an empty messages page
- No indication of what happened after booking
- Undermined the unified booking flow completed in previous P2 work

### Root Cause
1. `StoreItemView.vue:467` performed a "fire-and-forget" redirect: `router.push('/messages')` with no query parameters
2. `MessageCenter.vue` had no logic to handle incoming intents (query params like `?itemId=123`)

---

## Solution Implemented

### Changes Made

#### 1. StoreItemView.vue (frontend/src/views/store/StoreItemView.vue)
**Line 467:** Updated redirect to pass itemId as query parameter
```javascript
// Before:
router.push('/messages');

// After:
router.push({ path: '/messages', query: { itemId: item.value.id } });
```

#### 2. MessageCenter.vue (frontend/src/views/messages/MessageCenter.vue)
**Lines 60, 68, 70-88:** Added query parameter handling to auto-open conversations

**Imports:**
- Added `useRoute` from vue-router

**onMounted logic:**
- Made async to support awaiting conversation joins
- Added query parameter extraction for `itemId`, `taskId`, and `conversationId`
- Auto-joins appropriate conversation type based on query param:
  - `itemId` → calls `chatStore.joinStoreConversation(parseInt(itemId))`
  - `taskId` → calls `chatStore.joinConversation(parseInt(taskId))`
  - `conversationId` → calls `chatStore.joinConversation(parseInt(conversationId))`

---

## How It Works Now

### User Flow:
1. User clicks "Book Now" on a store item
2. Booking request is created via API
3. User is redirected to `/messages?itemId=123`
4. MessageCenter mounts and:
   - Connects to WebSocket
   - Loads deletion warnings
   - Detects `itemId` query parameter
   - Automatically joins the store conversation for that item
5. User immediately sees the booking conversation with the seller

### Benefits:
- Seamless UX - users see their booking request immediately
- No manual searching required
- Completes the unified booking & messaging flow
- Extensible - also supports `taskId` and `conversationId` parameters for future use

---

## Testing

- ✅ TypeScript type check: No new errors introduced
- ✅ Code review: Changes follow existing patterns in codebase
- ✅ Pre-existing errors: 62 TypeScript errors remain (tracked separately in TODO-typescript-errors.md)

---

## Files Modified

1. `frontend/src/views/store/StoreItemView.vue` - Added itemId to redirect query params
2. `frontend/src/views/messages/MessageCenter.vue` - Added auto-open logic for query params

---

## Follow-Up

This fix completes the P2 "Book Now" redirect context issue. The implementation is extensible and can be used for:
- Task detail pages redirecting to messages
- Application detail pages redirecting to messages
- Any other feature that needs to deep-link to a specific conversation

---

## Related Work

- **Complements:** P2 Unified Booking & Messaging Flow (completed Jan 4, 2026)
- **Investigation:** [INVESTIGATION-message-seller-functionality.md](../01-proposed/INVESTIGATION-message-seller-functionality.md)
- **Roadmap:** Updated P2 item #4 to completed status
