# Roadmap: Final MVP Review & Prioritization

This document summarizes a comprehensive review of the `store`, `task`, and `chat` services (both frontend and backend). It is a prioritized action plan focused on achieving a stable and functional Minimum Viable Product (MVP) for user acceptance testing.

---

## P0: MVP Blockers ✅ ALL RESOLVED

*These issues are critical and prevent the application from being deployed or tested meaningfully. They must be fixed first.*

### 1. Hardcoded URLs Across Frontend ✅ **RESOLVED**
-   **Service(s):** Store, Chat, Task
-   **Area:** Frontend
-   **Problem:** The Vue components are filled with hardcoded `http://localhost:xxxx` URLs for making API calls and loading images.
-   **Impact:** The application is **not deployable**. It will completely fail in any staging or production environment.
-   **Action:** Systematically replace all hardcoded URLs in `.vue` and `.ts` files with centralized variables imported from `frontend/src/config.ts`.
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Fixed:** 5 hardcoded URLs across 3 files (TaskListView, StoreView, StoreItemView)
-   **Added:** Missing VITE_STORE_API_URL and VITE_STORE_API_BASE_URL environment variables
-   **Created:** `.env.example` file for deployment reference
-   **Details:** See `../03-completed/FIXLOG-p0-hardcoded-urls.md`

### 2. Broken "Create Item" Functionality ✅ **RESOLVED**
-   **Service(s):** Store
-   **Area:** Frontend
-   **Problem:** The "Create Item" form sends a data payload with incorrect field names (e.g., `price` instead of `fixed_price`).
-   **Impact:** This is a **critical bug** that makes it impossible for users to create new items in the store, a core feature of the marketplace.
-   **Action:** Correct the field names in the `createItem()` function in `frontend/src/views/store/StoreView.vue` to match the backend API contract.
-   **Status:** ✅ **RESOLVED** (Already fixed - upon review, the function correctly maps fields)

### 3. WebSocket "Failed to join conversation" Error ✅ **RESOLVED**
-   **Service(s):** Chat
-   **Area:** Backend / Infrastructure
-   **Problem:** Users cannot open existing conversations due to a server-side error during the room joining process. Error: "relation 'store_items' does not exist".
-   **Impact:** **MVP Blocker**. Users cannot read or send messages in existing threads.
-   **Action:** Resolve the database configuration mismatch in `.env` and ensure `user_activity` table integrity.
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Root Cause:** Old container code attempted cross-database query to `store_items` table (in `my_guy_store`) while connected to `my_guy_chat`
-   **Fixed:** Updated .env database URL, added ID type parsing, removed cross-database query, rebuilt container
-   **Details:** See `../03-completed/FIXLOG-p0-websocket-join-conversation.md`

---

## P1: Critical for MVP

*These issues will lead to a broken, confusing, or unusable user experience. They must be resolved for a successful user test.*

### 1. Inefficient Client-Side Filtering
-   **Service(s):** Store
-   **Area:** Frontend
-   **Problem:** The store page fetches all items from the backend and performs filtering in the browser.
-   **Impact:** The marketplace will become **unusably slow and memory-intensive** with even a moderate number of items (e.g., 50-100+), leading to a very poor user experience.
-   **Action:** Refactor the `loadItems()` function to pass filter and search parameters to the backend API. Implement pagination at the same time to handle large datasets efficiently.

### 2. Inconsistent Chat State Management ✅ **RESOLVED**
-   **Service(s):** Chat, Store, Task
-   **Area:** Frontend
-   **Problem:** The chat feature within the Store view (`StoreItemView.vue`) does not use the central Pinia chat store (`chat.ts`). It has its own disconnected implementation. TaskDetailView uses older HTTP-based messagesStore.
-   **Impact:** This leads to a **broken user experience**. Messages initiated from the store will not appear in a user's global conversation list, causing confusion and the appearance of lost messages. No real-time updates.
-   **Action:** Refactor the chat implementation in `StoreItemView.vue` and `TaskDetailView.vue` to use the central `useChatStore` for all state and actions.
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Fixed:** Created ChatWindow.vue component and refactored both StoreItemView and TaskDetailView
-   **Details:** See `../03-completed/FIXLOG-p1-p2-chat-refactoring.md`

### 3. Lack of Backend Test Coverage
-   **Service(s):** Task (Main Backend)
-   **Area:** Backend
-   **Problem:** The main Go backend, which handles core business logic like user auth and task management, has **zero test coverage**.
-   **Impact:** This poses a **major stability risk**. Any future code change could silently break critical features like login, registration, or task creation, which would immediately halt user testing.
-   **Action:** Implement a foundational testing suite for the main backend, focusing on unit and integration tests for the most critical API endpoints (e.g., `/login`, `/register`, `/tasks`).

### 4. Non-Transactional Bidding Logic
-   **Service(s):** Store
-   **Area:** Backend
-   **Problem:** The auction bidding logic (reading the current bid and writing a new one) may not be atomic.
-   **Impact:** Risk of **data corruption** due to race conditions, where two users could seemingly place a winning bid simultaneously, breaking the auction's integrity.
-   **Action:** Ensure the entire bid placement process is wrapped in a single, atomic database transaction.

---

## P2: Recommended Before Wider Launch

*These items are important for long-term health, stability, and maintainability. They should be addressed after the P0/P1 issues and before a full public launch.*

### 1. Create a Reusable Frontend Chat Component ✅ **RESOLVED**
-   **Service(s):** Chat, Task, Store
-   **Area:** Frontend
-   **Problem:** The chat UI and logic are duplicated in `TaskDetailView.vue` and `StoreItemView.vue`.
-   **Impact:** This is a maintenance bottleneck.
-   **Action:** Create a single, reusable `ChatWindow.vue` component that can be used anywhere in the application. This is the correct way to fix the P1 chat state issue.
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Created:** ChatWindow.vue component (430 lines) with full WebSocket integration
-   **Eliminated:** 193 lines of duplicated chat code from StoreItemView and TaskDetailView
-   **Details:** See `../03-completed/FIXLOG-p1-p2-chat-refactoring.md`

### 2. Address Backend Scalability and Performance
-   **Service(s):** Chat, Store
-   **Area:** Backend
-   **Problem:** The chat service cannot scale beyond a single instance, and databases may be missing key performance-related indexes.
-   **Action:**
    -   **(Chat Service):** Plan for the integration of a Redis adapter for Socket.IO to enable horizontal scaling.
    -   **(All Services):** Review database schemas and ensure indexes are present on all foreign key columns and fields frequently used in `WHERE` clauses.

---

## P3: Post-MVP / Future Enhancements

*This category includes valuable features and architectural improvements that are not required for the initial user testing phase.*

-   Implement advanced auction features (proxy bidding, anti-sniping).
-   Add an event-driven architecture to synchronize data between services.
-   Implement soft deletes instead of hard deletes.
-   Add outbid and other user notifications.
-   Offload image serving to a CDN.
-   Implement inventory management for store items.
