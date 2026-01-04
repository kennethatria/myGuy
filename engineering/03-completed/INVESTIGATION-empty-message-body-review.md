# Investigation Report: Review of "Empty Message Body" Issue

**Date:** 2026-01-04
**Status:** 🔴 Issue Persists (Frontend Logic Error)
**Scope:** Review of `/messages` endpoint behavior after recent backend fixes.

## 1. Findings
The "Empty Message Body" issue reported by the user **has not been resolved**.
-   **Backend:** The backend (WebSocket service) is functioning correctly. It successfully emits `messages:list` with the requested message history.
-   **Frontend:** The frontend (`frontend/src/stores/chat.ts`) contains a **logic error** that prevents these messages from being displayed for Store Item conversations.

## 2. Technical Analysis
The root cause is a mismatch between where messages are *stored* and where they are *retrieved* in the Pinia store.

### The Storage Bug
In `frontend/src/stores/chat.ts`, the `handleMessagesList` function receives the messages from the backend but **incorrectly** stores them in the generic `messages` Map, regardless of the conversation type.

```typescript
// Current Code (Buggy)
function handleMessagesList({ itemId, messages: msgs, ... }) {
  // ...
  // ❌ FAILS to check for itemId, so it puts store messages in the generic 'messages' map
  messages.value.set(conversationId, msgs); 
}
```

### The Retrieval Mismatch
However, the `activeMessages` computed property (used by the UI) correctly attempts to read from the specialized `storeMessages` Map when the conversation type is 'store'.

```typescript
// Current Code (Correct Expectation)
const activeMessages = computed(() => {
  if (activeConversation.value.conversation_type === 'store') {
    return storeMessages.value.get(conversationId!) || []; // ❌ Returns empty because data is in the wrong map
  }
  return messages.value.get(conversationId!) || [];
});
```

### Result
1.  Backend sends data ✅
2.  Frontend receives data ✅
3.  Frontend saves data to Map A ❌
4.  UI tries to read data from Map B ❌
5.  **User sees:** Empty message body.

## 3. Required Fix
We must update `frontend/src/stores/chat.ts` to correctly route incoming store messages to the `storeMessages` Map.

**Plan:**
1.  Modify `handleMessagesList` to check for `itemId`.
2.  If present, store messages in `storeMessages`.
3.  Modify `joinConversation` and `loadMoreMessages` to ensure they read from the correct map when determining `hasMessages`.
