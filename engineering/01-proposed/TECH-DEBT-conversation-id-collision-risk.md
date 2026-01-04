# Technical Debt: Conversation ID Collision Risk

**Date Created:** 2026-01-04
**Priority:** P2 (Low)
**Complexity:** Medium
**Related Fix:** FIXLOG-store-message-routing.md

---

## Problem Description

The chat store (`frontend/src/stores/chat.ts`) uses simple integer IDs as Map keys for storing message histories. This creates a potential collision risk between different conversation types.

### Current Implementation

```typescript
// All conversation types use raw integer IDs as keys
const messages = ref<Map<number, Message[]>>(new Map());        // Tasks & Applications
const storeMessages = ref<Map<number, Message[]>>(new Map());   // Store Items
```

### The Risk

**Task and Application conversations share the same `messages` Map:**
- If Task ID = 5 and Application ID = 5 exist simultaneously
- Their message histories will overwrite each other
- Users would see incorrect messages in conversations

**Example Scenario:**
```typescript
// Task #5 messages are loaded
messages.value.set(5, [taskMessage1, taskMessage2]);

// Later, Application #5 messages are loaded
messages.value.set(5, [appMessage1, appMessage2]);  // ❌ Overwrites task messages!

// User opens Task #5 conversation
// Sees Application #5 messages instead! ❌
```

---

## Current Mitigation

**Store items use separate Map:**
- `storeMessages` Map prevents Store-Task/App collisions ✅
- Only Task-Application collisions are possible

**Why Low Priority:**
- In practice, Task IDs and Application IDs are unlikely to collide
- Applications are tied to specific tasks (relationship constraint)
- No user reports of this issue to date

---

## Proposed Solution

### Option 1: Composite Key Pattern (Recommended)

Use namespaced composite keys instead of raw integers:

```typescript
// Helper function to create composite keys
function getConversationKey(conv: ConversationSummary): string {
  if (conv.task_id) return `task:${conv.task_id}`;
  if (conv.application_id) return `app:${conv.application_id}`;
  if (conv.item_id) return `item:${conv.item_id}`;
  return '';
}

// Single unified Map with namespaced keys
const messages = ref<Map<string, Message[]>>(new Map());

// Usage
messages.value.set('task:5', [taskMessages]);
messages.value.set('app:5', [appMessages]);
messages.value.set('item:5', [storeMessages]);
```

**Benefits:**
- ✅ Eliminates all collision risks
- ✅ More explicit and readable code
- ✅ Single Map simplifies logic
- ✅ Easier to debug (keys are self-documenting)

**Effort:** ~4 hours
- Update all Map operations (set/get/has/delete)
- Update computed properties
- Update event handlers
- Test all conversation types

---

### Option 2: Separate Maps by Type

Create dedicated Maps for each conversation type:

```typescript
const taskMessages = ref<Map<number, Message[]>>(new Map());
const applicationMessages = ref<Map<number, Message[]>>(new Map());
const storeMessages = ref<Map<number, Message[]>>(new Map());
```

**Benefits:**
- ✅ Zero collision risk
- ✅ Type-safe separation
- ✅ Minimal refactoring from current state

**Drawbacks:**
- ❌ More code duplication
- ❌ More complex logic (3x the Map operations)
- ❌ Harder to maintain

**Effort:** ~2 hours

---

## Recommendation

**Implement Option 1 (Composite Keys)** in a future sprint.

**Rationale:**
- More robust and maintainable long-term
- Aligns with best practices for multi-type storage
- Small additional effort for significant architectural improvement
- Prevents future bugs as platform scales

**Suggested Timeline:** Q1 2026 (after backend testing priority)

---

## Implementation Checklist

When this is implemented:

- [ ] Create composite key helper function
- [ ] Update `handleMessagesList` to use composite keys
- [ ] Update `handleNewMessage` to use composite keys
- [ ] Update `joinConversation` to use composite keys
- [ ] Update `loadMoreMessages` to use composite keys
- [ ] Update `activeMessages` computed property
- [ ] Update all other Map operations (edit, delete, read)
- [ ] Update typing users Map
- [ ] Update unread counts Map
- [ ] Update hasMoreMessages Map
- [ ] Update totalMessageCounts Map
- [ ] Write unit tests for key generation
- [ ] Test all conversation types thoroughly
- [ ] Update this document to track completion

---

## Related Files

**Primary File:**
- `frontend/src/stores/chat.ts` (entire store)

**Test Files (to be created):**
- `frontend/src/stores/__tests__/chat.spec.ts`

**Documentation:**
- `engineering/03-completed/FIXLOG-store-message-routing.md`
- `engineering/01-proposed/REPORT-message-endpoint-findings.md`

---

## Notes

- This is architectural improvement, not a critical bug
- No user impact reported to date
- Should be addressed before platform scales significantly
- Could be part of broader chat store refactor/cleanup
