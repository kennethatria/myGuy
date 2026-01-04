# Fix Log: Empty Message Body in MessageCenter

**Date:** 2026-01-04
**Status:** ✅ Resolved
**Related Issue:** "Sidebar loads, but clicking a thread results in a spinning loader/empty body."

## Problem Description
Users reported that clicking on a store item conversation in the MessageCenter loaded the conversation header but left the message body empty (or showing a loading state).

## Investigation
1.  **Frontend Logic Error (Initial Finding):** The `handleMessagesList` function in `chat.ts` was originally putting all messages into the generic `messages` Map, while the `activeMessages` computed property expected store messages to be in the `storeMessages` Map.
2.  **Fix Verification (Secondary Finding):** Upon review, the logic to route store messages to `storeMessages` was *already present* in the code, yet the issue persisted.
3.  **Root Cause (Final Analysis):** The issue was identified as a **Type Mismatch**. The `socket.io` event might be delivering IDs as strings (or loose types), while the `Map` keys in the frontend store are strict numbers. `Map.set("1", ...)` followed by `Map.get(1)` results in `undefined`.

## Resolution
1.  **Updated `chat.ts`:** Modified `handleMessagesList` to explicitly cast `taskId`, `applicationId`, and `itemId` to `Number()` before using them as Map keys.
    ```typescript
    const parsedItemId = itemId ? Number(itemId) : undefined;
    // ...
    storeMessages.value.set(parsedItemId, msgs);
    ```

## Verification
- Code now enforces number types for all Map keys.
- Ensures consistency between `Map.set` (in handler) and `Map.get` (in computed property).
