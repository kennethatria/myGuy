# Fix Log: P0 - Core Messaging UX is Unusable

**Status:** ✅ **RESOLVED** - January 4, 2026

## Problem Statement

The chat system was unusable because:
1. **Unknown Sender Issue**: All messages displayed "Unknown User" instead of actual usernames
2. **Missing Context**: All conversations showed generic "Conversation" titles instead of task/item titles

This made it impossible for users to know who they were talking to or what the conversation was about - a complete blocker for any meaningful communication.

## Root Cause

The `chat-websocket-service` operates on IDs only (sender_id, recipient_id, task_id, item_id) and doesn't have access to other services' databases due to the microservices architecture. The frontend components expected enriched data (sender.username, task_title, item_title) that was never provided.

## Solution Implemented

Implemented **frontend enrichment strategy** as recommended in RFC-unknown-sender.md and RFC-conversation-titles.md:

### 1. Created User Store (`frontend/src/stores/user.ts`)
- Caches user data in a Map<userId, User>
- Prevents duplicate API calls with pending fetch tracking
- Provides `fetchUser()` and `fetchUsers()` for batch operations
- Auto-caches current user on login/auth check

### 2. Created Context Store (`frontend/src/stores/context.ts`)
- Caches task and store item data separately
- Provides `fetchTask()` and `fetchItem()` with deduplication
- Batch operations for efficient parallel loading

### 3. Enhanced Chat Store (`frontend/src/stores/chat.ts`)
- Added `enrichConversations()` function:
  - Collects all unique user IDs, task IDs, and item IDs from conversations
  - Fetches all data in parallel
  - Updates conversation objects with usernames and titles
- Added `enrichMessages()` function:
  - Enriches messages with sender/recipient data when loaded or received
- Integrated enrichment into:
  - `handleConversationsList()` - when conversations load
  - `handleMessagesList()` - when messages load
  - `handleNewMessage()` - when new messages arrive in real-time

### 4. Updated UI Components
- **MessageBubble.vue**:
  - Added computed `senderName` property
  - Falls back to user store lookup if message.sender is undefined
  - Reactive to user store updates

- **ConversationItem.vue**:
  - Added computed `conversationTitle` property
  - Added computed `otherUserName` property
  - Falls back to context/user stores with sensible defaults (e.g., "Item #123")

### 5. Auth Store Integration
- Updated `login()`, `checkAuth()` to cache current user
- Updated `logout()` to clear user cache
- Ensures current user is always available in user store

## Files Created
- `frontend/src/stores/user.ts` (137 lines)
- `frontend/src/stores/context.ts` (226 lines)

## Files Modified
- `frontend/src/stores/chat.ts` - Added enrichment logic
- `frontend/src/stores/auth.ts` - Added user caching integration
- `frontend/src/components/messages/MessageBubble.vue` - Reactive sender names
- `frontend/src/components/messages/ConversationItem.vue` - Reactive titles and usernames

## Architecture Benefits

1. **Service Independence**: Chat service remains decoupled, only dealing with IDs
2. **Frontend Caching**: Efficient batch loading prevents N+1 API call issues
3. **Reactive Updates**: Vue computed properties ensure UI updates when data loads
4. **Deduplication**: Pending fetch tracking prevents multiple requests for same data
5. **Graceful Degradation**: Components show placeholder text (e.g., "Item #123") while loading

## Testing Notes

Build succeeded with no new TypeScript errors. Pre-existing TypeScript errors (unrelated to this change) remain.

To test:
1. Start all services with `docker-compose up`
2. Create/view conversations
3. Verify usernames display instead of "Unknown User"
4. Verify conversation titles display task/item names instead of "Conversation"

## Related RFCs
- `engineering/01-proposed/RFC-unknown-sender.md`
- `engineering/01-proposed/RFC-conversation-titles.md`

## Next Steps
This completes the P0 blocker. The team can now proceed with P1 items:
1. Backend filtering for store items
2. Backend testing foundation
3. Transactional bidding
