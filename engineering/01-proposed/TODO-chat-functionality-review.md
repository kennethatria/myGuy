# TODO: Chat Functionality Review & Improvements

**Status:** Proposed
**Priority:** 🔴 Critical / 🟡 High

This document summarizes a review of the chat functionality across the backend service and frontend implementations. It highlights critical architectural issues, bugs, and areas for improvement.

---

## 1. Backend (`chat-websocket-service`)

### 🔴 Issue 1.1: Service is Not Horizontally Scalable
-   **Observation:** The service uses Socket.IO's default in-memory adapter.
-   **Impact:** This implementation only works for a **single service instance**. In a production environment with multiple instances for high availability or load balancing, users connected to different instances will not be able to communicate with each other.
-   **Suggestion:**
    1.  Integrate the **Socket.IO Redis Adapter** (`@socket.io/redis-adapter`).
    2.  This will use Redis Pub/Sub to broadcast events across all service instances, enabling seamless horizontal scaling.
    3.  Update the deployment configuration to include a Redis instance.

### 🟡 Issue 1.2: Lack of Explicit Database Indexing
-   **Observation:** The `chat-websocket-service/README.md` details the database schema but does not explicitly mention which columns are indexed.
-   **Impact:** As the `messages` table grows, queries to fetch messages for a specific conversation (e.g., `WHERE task_id = ?`) will become very slow without proper database indexes, leading to poor performance.
-   **Suggestion:**
    1.  Add indexes to all foreign key columns used for querying conversations in the `messages` table: `task_id`, `application_id`, `store_item_id`.
    2.  Add a composite index on `(sender_id, recipient_id)` if there are frequent lookups of conversations between two specific users.
    3.  Document these indexes in the `chat-websocket-service/README.md`.

---

## 2. Frontend (General Implementation)

### 🔴 Issue 2.1: Massive Code Duplication due to Lack of Reusable Chat Component
-   **Observation:** There is no generic, reusable chat component (e.g., `ChatWindow.vue`). The entire chat UI and associated logic are implemented directly inside `TaskDetailView.vue` and, even worse, separately inside `StoreItemView.vue`.
-   **Impact:**
    -   **Maintenance Nightmare:** Any change or bug fix to the chat UI must be manually applied in multiple places.
    -   **Inconsistent UX:** The two implementations will inevitably diverge, leading to an inconsistent user experience.
    -   **Violates DRY Principle:** This is a major violation of the "Don't Repeat Yourself" principle.
-   **Suggestion:**
    1.  Create a new, reusable component named `ChatWindow.vue`.
    2.  This component should encapsulate all UI elements for a conversation: the message list, the message input box, and the send button.
    3.  It should interact with the central `useChatStore` (Pinia) to get messages and send new ones based on a `conversationId` prop.
    4.  Refactor both `TaskDetailView.vue` and `StoreItemView.vue` to remove their duplicated chat UI and simply use the new component: `<ChatWindow :conversation-id="activeId" />`.

### 🔴 Issue 2.2: Inconsistent State Management (`StoreItemView.vue`)
-   **File:** `frontend/src/views/store/StoreItemView.vue`
-   **Observation:** The chat modal within this component **does not use the central Pinia store (`useChatStore`)**. It has its own disconnected, component-level state (`ref`s) and its own logic for fetching and sending messages.
-   **Impact:** This is a critical architectural flaw. It completely breaks the concept of centralized state management. Conversations initiated from the store will not appear in a global message center, and state will not be shared.
-   **Suggestion:**
    1.  **Completely remove** the internal chat state and API call logic from `StoreItemView.vue`.
    2.  Refactor the component to use the methods and state from `useChatStore` exclusively. For example, instead of `fetch(...)`, it should call `chatStore.sendStoreMessage(...)`.

### 🟡 Issue 2.3: Inconsistent Use of API URLs
-   **Observation:** Multiple files, including `chat.ts` and `StoreItemView.vue`, contain hardcoded `http://localhost:xxxx` URLs as fallbacks or primary endpoints.
-   **Impact:** This makes the application brittle and non-portable. It will fail in any environment other than the original developer's local machine.
-   **Suggestion:**
    1.  Enforce a strict policy that **all** API calls must use the URLs defined in `frontend/src/config.ts`.
    2.  Remove all hardcoded `http://localhost...` URLs from all `.vue` and `.ts` files.

---

## 3. Action Plan

1.  **(Backend) High Priority:** Add database indexes to the `messages` table for all foreign key columns used in `WHERE` clauses.
2.  **(Frontend) Critical:** Create a reusable `ChatWindow.vue` component.
3.  **(Frontend) Critical:** Refactor `StoreItemView.vue` to remove its internal chat implementation and use the central `useChatStore` and the new `ChatWindow.vue` component.
4.  **(Frontend) High Priority:** Refactor `TaskDetailView.vue` to use the new `ChatWindow.vue` component.
5.  **(Frontend) High Priority:** Audit all files and remove hardcoded URLs, ensuring all API calls use the central `config.ts` file.
6.  **(Backend) Medium Priority:** For future production scaling, plan the integration of a Redis instance and the Socket.IO Redis adapter. Document this requirement.
