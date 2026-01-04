# Roadmap: MVP Launch Prioritization

**[NOTICE] This document is the single source of truth for MVP priorities. When updating this file, please ensure the summary in `engineering/❗-current-focus.md` is also updated to reflect the current P0 and top P1 items.**

This document outlines the prioritized list of issues to be addressed before deploying a Minimum Viable Product (MVP) for user testing. The goal is to ensure a stable, functional, and usable product for early adopters.

The items are sourced from the `TODO` and `ROADMAP` documents in this directory.

---

## P0: MVP Blockers

*These issues make the application undeployable or non-functional in a core area. They must be fixed first.*

### 1. Fix Hardcoded URLs Across the Frontend ✅ **RESOLVED**
-   **Problem:** The frontend is filled with hardcoded `http://localhost:xxxx` URLs for API calls and image links.
-   **Impact:** The application is not deployable and will not work in any environment other than the original developer's local machine.
-   **Action:** Replace all hardcoded URLs with the centralized variables from `frontend/src/config.ts`.
-   **Source:** `TODO-frontend-store-service-integration.md`
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Details:** All 5 hardcoded URLs fixed across 3 files. Added missing env vars. Created `.env.example`. See `../03-completed/FIXLOG-p0-hardcoded-urls.md`

### 2. Fix "Create Item" Functionality ✅ **RESOLVED**
-   **Problem:** The form for creating a new store item sends a data payload with incorrect field names, causing the request to fail on the backend.
-   **Impact:** Users cannot create new items in the store, which is a fundamental feature.
-   **Action:** Correct the field names in the `createItem()` function in `frontend/src/views/store/StoreView.vue` to match the backend API.
-   **Source:** `TODO-frontend-store-service-integration.md`
-   **Status:** ✅ **RESOLVED** (Upon re-examination, the `createItem` function already correctly maps fields as per backend contract.)

### 3. Fix WebSocket "Failed to join conversation" Error ✅ **RESOLVED**
-   **Problem:** Users receive a "Failed to join conversation" error when clicking on a message summary.
-   **Impact:** Core messaging functionality is broken; users cannot read or respond to existing conversations.
-   **Root Cause:** Old container code trying to query `store_items` table (exists in `my_guy_store` database, not `my_guy_chat`). Container wasn't rebuilt after code changes.
-   **Action:** Sync `.env` with `docker-compose.yml`, verify `my_guy_chat` database schema, and ensure robust ID parsing in WebSocket handlers.
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Fixed:** Updated .env to point to correct database, added ID parsing (parseInt), removed cross-database query, rebuilt container
-   **Details:** See `../03-completed/FIXLOG-p0-websocket-join-conversation.md`

### 4. MessageCenter Loading & Display Failures ✅ **RESOLVED**
-   **Problem:** Store item messages failed to load in MessageCenter due to cross-database queries and frontend logic errors.
-   **Impact:** MVP Blocker. Users could not view or reply to store-related messages.
-   **Action:** Removed cross-database queries and fixed Map key type mismatches.
-   **Status:** ✅ **RESOLVED** - January 4, 2026
-   **Details:** See `../03-completed/FIXLOG-messagecenter-loading-failure.md` and `../03-completed/FIXLOG-empty-message-body.md`

### 5. Core Messaging UX is Unusable
- **Problem:** Users cannot identify who they are talking to ("Unknown User") or what they are talking about ("Conversation" title). This makes the chat feature non-functional for its primary purpose.
- **Impact:** Blocker for any user testing. A chat system without sender and context is unusable.
- **Action:** Implement the frontend enrichment strategies to resolve user and context (task/item) titles.
- **Source:** [RFC-unknown-sender.md](./RFC-unknown-sender.md), [RFC-conversation-titles.md](./RFC-conversation-titles.md)
- **Status:** 🔴 Open

---

## P1: Critical for MVP

*These issues will lead to a broken or confusing user experience. They are critical for a successful user test.*

### 1. Implement Backend Filtering for Store Items
-   **Problem:** The store page fetches all items and filters them on the client-side.
-   **Impact:** The page will be extremely slow and memory-intensive with even a moderate number of items, making the marketplace unusable.
-   **Action:** Modify the frontend to send filter parameters to the backend API and handle the filtered results. Implement pagination at the same time.
-   **Source:** `TODO-frontend-store-service-integration.md`

### 2. Add Backend Testing Foundation
-   **Problem:** The main `backend` service has zero test coverage.
-   **Impact:** There is a very high risk of regressions and uncaught bugs in core business logic (tasks, applications, user management). A single change could break the platform without warning.
-   **Action:** Implement a basic testing suite for the main backend, focusing first on critical paths like user authentication and task creation.
-   **Source:** `ADR-backend-testing-strategy.md`

### 3. Ensure Transactional Bidding
-   **Problem:** The bidding logic may be vulnerable to race conditions.
-   **Impact:** Potential for data corruption and an unfair auction if two bids are placed simultaneously.
-   **Action:** Review the backend bidding logic and ensure the read-validate-write process is wrapped in a database transaction.
-   **Source:** `ROADMAP-store-service-improvements.md`

---

## P2: Recommended Before Launch

*These are important for stability and maintainability. It is strongly recommended to address them before a wider public launch, but they are not immediate blockers for a small, controlled user test.*

### 1. Refactor Frontend Chat into a Reusable Component ✅ **RESOLVED**
-   **Problem:** The chat UI and logic are duplicated in multiple places.
-   **Impact:** This creates a significant maintenance burden.
-   **Action:** Create a single, reusable `ChatWindow.vue` component and use it in both the Task and Store views. This should be done as part of fixing the P1 chat state issue.
-   **Source:** `TODO-chat-functionality-review.md`
-   **Status:** ✅ **RESOLVED** - January 3, 2026
-   **Fixed:** Created ChatWindow.vue component (430 lines) and integrated into StoreItemView and TaskDetailView
-   **Eliminated:** 193 lines of duplicated chat code
-   **Details:** See `../03-completed/FIXLOG-p1-p2-chat-refactoring.md`

### 2. Address Backend Scalability & Performance
-   **Problem:** The chat service cannot scale beyond a single instance, and the database may be missing key indexes.
-   **Impact:** The system may face performance issues under moderate load.
-   **Action:**
    -   (Chat) Plan for integrating a Redis adapter for Socket.IO.
    -   (Chat & Store) Review and add necessary database indexes to foreign key columns and common query filters.
-   **Source:** `TODO-chat-functionality-review.md`, `ROADMAP-store-service-improvements.md`

---

## P3: Post-MVP / Future Enhancements

*These items are valuable features and architectural improvements for future iterations.*

-   Implement advanced auction features (proxy bidding, anti-sniping).
-   Add an event-driven architecture to synchronize user data between services.
-   Implement soft deletes instead of hard deletes.
-   Add a background job for auction expiration.
-   Offload image serving to a CDN.
-   Implement inventory management (quantity) for store items.
-   Add outbid notifications.

---

## Recommendation Summary

For the MVP user test, you must address all **P0 Blockers** and **P1 Critical** issues. This will ensure you have a deployable application where the core user journeys—creating and viewing items, and communicating about them—are functional and performant enough for initial feedback.
