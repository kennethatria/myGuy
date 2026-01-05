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

### 5. Core Messaging UX is Unusable ✅ **RESOLVED**
- **Problem:** Users cannot identify who they are talking to ("Unknown User") or what they are talking about ("Conversation" title). This makes the chat feature non-functional for its primary purpose.
- **Impact:** Blocker for any user testing. A chat system without sender and context is unusable.
- **Action:** Implement the frontend enrichment strategies to resolve user and context (task/item) titles.
- **Source:** [RFC-unknown-sender.md](./RFC-unknown-sender.md), [RFC-conversation-titles.md](./RFC-conversation-titles.md)
- **Status:** ✅ **RESOLVED** - January 4, 2026
- **Details:** Created user and context stores for caching. Implemented enrichment in chat store. Updated UI components. See `../03-completed/FIXLOG-p0-messaging-ux.md`

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

### 4. Fix Seller Name Display and Profile Link
-   **Problem:** Seller name not displayed on store item pages; shows empty space. Frontend attempts to access `item.seller.full_name` but Store Service API returns `name` field.
-   **Impact:** Users cannot see who they're buying from - critical trust and transparency issue for marketplace. Generic "View Profile" link provides no context about whose profile they're viewing.
-   **Root Cause:** Field name mismatch between Store Service API (`name`) and Frontend code (`full_name`). Three occurrences in `StoreItemView.vue:54, :585, :604`.
-   **Secondary Issue:** Possible wrong profile display due to data seeding inconsistencies between Backend and Store Service user IDs.
-   **Action:**
    1. Update `StoreItemView.vue` to use `item.seller.name` instead of `item.seller.full_name` (add fallback to `username` if name is missing)
    2. Verify and re-seed database ensuring Store Items are created by users with matching IDs in both services
    3. Consider enhancing link label to "View [Name]'s Profile" for clarity
-   **Source:** [INVESTIGATION-view-seller-link.md](./INVESTIGATION-view-seller-link.md)
-   **Status:** 📋 Tracked, not started
-   **Effort:** Low (simple field name change + data verification)

---

## P2: Recommended Before Launch

*These are important for stability and maintainability. It is strongly recommended to address them before a wider public launch, but they are not immediate blockers for a small, controlled user test.*

### 1. Unified Booking & Messaging Flow ✅ **COMPLETED**
-   **Problem:** Sellers are not notified of booking requests and must manually check each item page. Booking and communication are fragmented across different UIs.
-   **Impact:** Poor seller experience, missed booking requests, lower conversion rate. Especially problematic for sellers with multiple items.
-   **Action:** Integrate booking workflow into the chat system. Buyers get redirected to chat after booking, sellers see booking requests as special messages with approve/decline buttons.
-   **Source:** [RFC-booking-via-messages.md](./RFC-booking-via-messages.md), [PLAN-unified-booking-messaging.md](./PLAN-unified-booking-messaging.md), [USER-FLOW-booking-summary.md](./USER-FLOW-booking-summary.md)
-   **Status:** ✅ **COMPLETED** - January 4, 2026
-   **Backend Status:** ✅ **COMPLETED** - January 4, 2026 (migrations, endpoints, notification logic)
-   **Frontend Status:** ✅ **COMPLETED** - January 4, 2026 (booking message component, chat UI updates, redirect flow)
-   **Details:** See `../03-completed/IMPLEMENTATION-unified-booking-backend.md` and `../03-completed/IMPLEMENTATION-unified-booking-frontend.md`
-   **Deployment:** See `DEPLOYMENT-CHECKLIST-booking.md` for deployment steps

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

### 3. Resolve TypeScript Type Errors
-   **Problem:** 62 pre-existing TypeScript type errors across 10 frontend files (App.vue, TaskDetailView, StoreItemView, etc.).
-   **Impact:** Reduced type safety, potential runtime errors, harder to maintain code, builds succeed but with warnings.
-   **Action:**
    -   **Phase 1 (P1 - Before MVP):** Fix critical errors in App.vue, StoreItemView, TaskDetailView (~40 errors, 4-6 hours)
    -   **Phase 2 (P2 - Before Production):** Standardize property naming, consolidate types, fix null safety (~22 errors, 10-15 hours)
    -   **Phase 3 (P3 - Future):** Enable strict TypeScript mode, prevent regression via CI
-   **Source:** [TODO-typescript-errors.md](./TODO-typescript-errors.md)
-   **Root Causes:** Inconsistent snake_case/camelCase, missing null checks, duplicate type definitions, missing type annotations
-   **Status:** 📋 Tracked, not started

### 4. Fix "Book Now" Redirect Context Loss ✅ **COMPLETED**
-   **Problem:** After clicking "Book Now" on a store item, users are redirected to `/messages` without any context. The Message Center doesn't know which conversation to open, forcing users to manually search for the newly created booking conversation.
-   **Impact:** Poor booking UX - buyers complete the booking action but land on an empty messages page with no indication of what happened. This undermines the unified booking flow completed in P2.
-   **Root Cause:**
    -   `StoreItemView.vue:467` performs a "fire-and-forget" redirect: `router.push('/messages')` with no query parameters
    -   `MessageCenter.vue` has no logic to handle incoming intents (query params like `?conversationId=123` or `?itemId=789`)
-   **Action:**
    1. Modify `StoreItemView.vue` to pass context when redirecting (e.g., `router.push({ path: '/messages', query: { itemId: item.value.id } })`)
    2. Update `MessageCenter.vue` to read query parameters in `onMounted` and auto-open the relevant conversation via `chatStore.joinStoreConversation(itemId)`
-   **Source:** [INVESTIGATION-message-seller-functionality.md](./INVESTIGATION-message-seller-functionality.md)
-   **Status:** ✅ **COMPLETED** - January 5, 2026
-   **Details:** See `../03-completed/FIXLOG-p2-booking-redirect-context.md`
-   **Related:** Complements P2 "Unified Booking & Messaging Flow" (completed Jan 4, 2026)

### 5. Enhanced Booking Request Flow ✅ **COMPLETED**
-   **Problem:** Buyers don't get immediate feedback after booking. Sellers must check individual item pages to see booking requests.
-   **Impact:** Poor UX - buyers unsure if booking worked, sellers miss booking requests, lost sales opportunities.
-   **Solution:**
    1. Show confirmation modal with messaging after "Book Now"
    2. Add visual badges and sort booking requests to top in /messages
    3. Fix async race conditions in conversation joining
    4. Fix backend room name bug preventing seller notifications
-   **Features Implemented:**
    - Booking confirmation modal with embedded chat
    - Retry logic for handling async booking notifications
    - Golden calendar badges for booking conversations
    - 3-tier priority sorting (bookings → unread → recent)
    - Fixed WebSocket room name mismatch bug
-   **Status:** ✅ **COMPLETED** - January 5, 2026
-   **Details:** See `../03-completed/FIXLOG-enhanced-booking-flow.md`
-   **Related:** Builds on P2 "Unified Booking & Messaging Flow"

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
