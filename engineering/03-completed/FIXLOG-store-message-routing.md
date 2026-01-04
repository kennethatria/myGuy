# Fix Log: Store Message Routing in Chat Store

**Date:** 2026-01-04
**Issue:** Empty Message Body Bug in MessageCenter for Store Item Conversations
**Severity:** High - Critical user-facing bug
**Status:** ✅ Fixed

---

## Problem Summary

The `/messages` endpoint (MessageCenter) was displaying empty message bodies for store item conversations, despite successfully connecting to the chat WebSocket service and receiving message data.

### Root Cause

**Frontend Logic Error** in `frontend/src/stores/chat.ts` - Messages were being stored in the wrong Map:
- Store item messages were stored in the generic `messages` Map
- But the `activeMessages` computed property expected them in the `storeMessages` Map
- Result: Empty UI because the component couldn't find the data

---

## Issues Fixed

### Issue #1: handleMessagesList Function (Line 409)
**Problem:** All incoming messages were routed to the generic `messages` Map, regardless of conversation type.

**Impact:**
- ❌ Store item conversations showed empty message bodies
- ❌ Messages were received but not displayed to users

**Fix:** Added conditional routing logic to direct store item messages to `storeMessages` Map:
```typescript
if (itemId) {
  // Store item messages go to storeMessages Map
  storeMessages.value.set(itemId, msgs);
} else {
  // Task/Application messages go to generic messages Map
  messages.value.set(conversationId, msgs);
}
```

**File:** `frontend/src/stores/chat.ts:409-442`

---

### Issue #2: joinConversation Function (Line 520)
**Problem:** Check for existing messages always looked in the generic `messages` Map, even for store items.

**Impact:**
- ❌ Duplicate message loading requests for store items
- ❌ Unnecessary network calls
- ⚠️ Performance degradation

**Fix:** Check correct Map based on conversation type:
```typescript
const hasMessages = conv.item_id
  ? storeMessages.value.has(conversationId)
  : messages.value.has(conversationId);
```

**File:** `frontend/src/stores/chat.ts:520-533`

---

### Issue #3: loadMoreMessages Function (Line 604)
**Problem:** Always retrieved current messages from generic `messages` Map to calculate pagination offset.

**Impact:**
- ❌ "Load More" button broken for store item conversations
- ❌ Offset calculation always returned 0 (empty array length)
- ❌ Same first batch reloaded repeatedly instead of older messages

**Fix:** Get messages from correct Map based on conversation type:
```typescript
const currentMessages = activeConversation.value.item_id
  ? (storeMessages.value.get(conversationId) || [])
  : (messages.value.get(conversationId) || []);
```

**File:** `frontend/src/stores/chat.ts:604-631`

---

### Issue #4: MessageCenter Calling Wrong Method (CRITICAL)
**Problem:** MessageCenter.vue called `joinStoreConversation()` for store items instead of `joinConversation()`.

**Impact:**
- ❌ **CRITICAL:** Store conversations didn't set activeConversation
- ❌ **CRITICAL:** Messages never loaded when selecting store conversation
- ❌ **CRITICAL:** Empty message body on every store conversation selection
- ❌ Conversation not marked as read
- ❌ Previous conversation not properly left

**Root Cause:** `joinStoreConversation` is an incomplete stub method that only emits join event. It doesn't:
- Set `activeConversation`
- Load messages
- Mark conversation as read
- Leave previous conversation

But `joinConversation` already handles ALL conversation types including store items properly!

**Fix:** Remove conditional logic and always use `joinConversation`:
```typescript
// Before (incorrect)
if (conversation.conversation_type === 'store') {
  chatStore.joinStoreConversation(conversationId);
} else {
  chatStore.joinConversation(conversationId);
}

// After (correct)
chatStore.joinConversation(conversationId);
```

**File:** `frontend/src/views/messages/MessageCenter.vue:76-82`

---

## Files Modified

1. **frontend/src/stores/chat.ts**
   - `handleMessagesList()` - Lines 409-442
   - `joinConversation()` - Lines 520-533
   - `loadMoreMessages()` - Lines 604-631

2. **frontend/src/views/messages/MessageCenter.vue**
   - `selectConversation()` - Lines 76-82 (removed incorrect conditional)

---

## Testing Recommendations

### Manual Testing Checklist
- [ ] Open MessageCenter (`/messages`)
- [ ] Navigate to a store item conversation
- [ ] Verify messages display correctly (not empty)
- [ ] Send a new message in store conversation
- [ ] Verify sent message appears immediately
- [ ] Scroll up and click "Load More" (if >20 messages)
- [ ] Verify older messages load correctly
- [ ] Switch between store and task conversations
- [ ] Verify no duplicate loading requests (check Network tab)

### Regression Testing
- [ ] Task conversations still work correctly
- [ ] Application conversations still work correctly
- [ ] Message sending/receiving for all conversation types
- [ ] Real-time message updates via WebSocket
- [ ] Typing indicators
- [ ] Unread count updates

---

## Known Technical Debt (Not Addressed in This Fix)

### ID Collision Risk
**Issue:** Task IDs and Application IDs are simple integers sharing the same `messages` Map. If a Task ID equals an Application ID, their message histories could overwrite each other.

**Current Mitigation:** Store items use separate `storeMessages` Map, preventing Store-Task/App collisions.

**Recommendation:** Future refactor should use composite keys (e.g., `task:1`, `app:1`, `item:1`) instead of raw integers.

**Priority:** P2 (Low risk in practice, but architectural improvement needed)

**Tracking:** Document in `engineering/01-proposed/` for future sprint

---

## Related Documents

- **Original Report:** `engineering/01-proposed/REPORT-message-endpoint-findings.md`
- **Architecture Reference:** `CLAUDE.md` (Chat Service section)
- **Service Code:** `chat-websocket-service/src/handlers/socketHandlers.js`

---

## Deployment Notes

### Pre-Deployment
1. Review changes in `frontend/src/stores/chat.ts`
2. Run frontend type checking: `npm run type-check`
3. Run frontend build: `npm run build`
4. Test in staging environment

### Post-Deployment
1. Monitor browser console for WebSocket errors
2. Check Sentry/error tracking for new frontend errors
3. Verify MessageCenter works for store conversations
4. Monitor chat service logs for any anomalies

---

## Credits

- **Issue Discovery:** Gemini Agent (Automated Audit)
- **Additional Issues:** Claude Code (Code Review)
- **Implementation:** Claude Code
- **Date Fixed:** 2026-01-04
