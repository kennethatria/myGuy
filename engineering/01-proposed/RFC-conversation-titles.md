# RFC: Descriptive Conversation Titles

## 1. Context & Problem
Currently, in the `/messages` list, all conversations related to Tasks, Applications, or Store Items are displayed with a generic title of "Conversation".

### The Issue
- **Usability:** Users with multiple active conversations (e.g., selling 5 different items) cannot distinguish between them. They must click into each one to read the history to know what it is about.
- **Data Gap:** The frontend `ConversationItem.vue` component attempts to display `task_title`, `application_title`, or `item_title`, but these fields are undefined in the data returned by the backend.
- **Architectural Constraint:** The `chat-websocket-service` connects only to the `my_guy_chat` database. It does not have access to the `tasks` (in `my_guy` DB) or `store_items` (in `my_guy_store` DB) tables to join and fetch titles.

## 2. Proposed Solution
To provide context, we need to display the **Title** of the Task or Store Item in the conversation list.

### Option A: Frontend-Side Enrichment (Recommended for MVP)
The frontend will be responsible for fetching the missing titles.

1.  **Fetch List:** `MessageCenter.vue` loads the conversation list (containing only IDs).
2.  **Extract IDs:** The component extracts all unique `store_item_id`s and `task_id`s.
3.  **Batch/Parallel Fetch:**
    *   Call `GET /api/store/items?ids=...` (or individual calls if batch endpoint doesn't exist) to get item titles.
    *   Call `GET /api/tasks?ids=...` to get task titles.
4.  **Merge:** Update the local `conversations` state with the fetched titles.

**Pros:**
*   No changes to `chat-websocket-service` or database schema.
*   Respects service isolation.
*   Lowest risk of regression.

**Cons:**
*   "Chatty" frontend (N+1 requests if batching isn't implemented).
*   Slight delay in showing titles after the list loads.

### Option B: Backend Data Duplication (Long-Term)
Store the title in the `my_guy_chat` database.

1.  **Schema Change:** Add `context_title` column to `messages` (or a new `conversations` table).
2.  **Write Path:** When a message is sent via `POST /messages`, the client (or the calling service) must provide the current Title.
3.  **Read Path:** The `conversations:list` event simply returns this stored title.

**Pros:**
*   Fast read performance (single query).
*   Titles persist even if the original item is deleted (preserves history context).

**Cons:**
*   Requires schema migration.
*   Requires updating all message creation endpoints.
*   Data sync issues (if item title changes, chat title is stale - though this might be a feature).

## 3. Implementation Plan (Option A)

### Phase 1: Frontend Update
1.  Update `useChatStore` action `loadConversations` to:
    *   Receive the list.
    *   Identify items missing titles.
    *   Trigger an `enrichConversations` action.
2.  Implement `enrichConversations`:
    *   Use `fetch` to call Store/Task APIs.
    *   Update the `conversations` array with the results.
3.  Update `ConversationItem.vue` to handle the "loading" state of a title (e.g., show "Item #123" while loading "Red Bicycle").

## 4. Roadmap Placement
This is a **UX Priority** for the MVP. While functionally the chat works, a marketplace is unusable if sellers cannot distinguish between buyers.

**Priority:** P1 (Critical for MVP)
