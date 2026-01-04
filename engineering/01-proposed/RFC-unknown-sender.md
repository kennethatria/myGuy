# RFC: Resolving "Unknown User" in Messages

## 1. Context & Problem
In the chat interface, the sender of every message is displayed as "Unknown User" instead of their actual username. This happens in both the main message thread and the conversation list sidebar.

### The Issue
- **Usability & Trust:** It's impossible to know who sent a message, which makes the chat feature unusable and untrustworthy. Users cannot have a coherent conversation if they don't know who they are talking to.
- **Root Cause:** The `chat-websocket-service` operates on `sender_id` and `recipient_id` only. It does not have access to the `users` table (which is in a separate database) and therefore does not resolve user IDs to usernames before sending message data to the frontend.
- **Frontend Expectation:** The `MessageBubble.vue` component expects a `message.sender.username` property, which is always `undefined`, causing it to fall back to the "Unknown User" string.

## 2. Proposed Solution
To fix this, user data (ID and username) must be fetched and associated with the message objects on the frontend.

### Recommended Approach: Frontend Enrichment via a User Store

This approach keeps services decoupled and is the most consistent with the existing architecture.

1.  **Create a `useUserStore`:**
    *   A new Pinia store will be created to act as a global, app-wide cache for user data.
    *   It will hold a `Map<number, User>` where `User` is an interface `{ id: number; username: string; ... }`.

2.  **Enrichment Logic in `useChatStore`:**
    *   When `useChatStore` receives messages (either from `messages:list` or `message:new`), it will inspect the `sender_id` and `recipient_id`.
    *   For each ID, it will call an action in `useUserStore` like `usersStore.fetchUser(userId)`.

3.  **`useUserStore` Actions:**
    *   `fetchUser(userId)`: Checks if the user is already in the cache. If not, it fetches the user's data from the main backend API (`GET /api/users/:id`) and adds it to the cache.
    *   `fetchUsers(userIds)`: A batch-fetching version (`GET /api/users?ids=1,2,3`) would be more efficient to prevent N+1 API calls when loading a conversation history.

4.  **UI Component Update:**
    *   The `MessageBubble.vue` and `ConversationItem.vue` components will be updated to get user data from `useUserStore` based on the ID.
    *   Example: `const sender = computed(() => usersStore.getUserById(props.message.sender_id))`

### Alternative: Backend Data Duplication
*   **Description:** The `chat-websocket-service` could store a `sender_username` directly on the `messages` table.
*   **Reason for Not Recommending:** This denormalizes data and makes it stale if a user changes their username. It's better to keep the User service as the single source of truth for user information.

## 3. Implementation Plan (High-Level)

### Phase 1: Backend API Endpoint
*   Ensure a batch-capable user endpoint exists on the main backend.
    *   `GET /api/users?ids=id1,id2,id3` should return an array of user objects.

### Phase 2: Frontend Store
*   Create `frontend/src/stores/user.ts` with the caching logic described above.

### Phase 3: Integration
*   Modify `frontend/src/stores/chat.ts` to call the user store when processing incoming messages.
*   Modify UI components (`MessageBubble.vue`, `ConversationItem.vue`) to use the user store for displaying names.

## 4. Roadmap Placement
This is a **blocker** for any meaningful user testing. A chat where you cannot identify the other party is fundamentally broken.

**Priority:** P0 (MVP Blocker)
