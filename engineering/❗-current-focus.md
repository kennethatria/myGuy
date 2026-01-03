# Engineering Priorities: Q1 2026

This document provides a high-level summary of the current engineering focus, upcoming priorities, and links to relevant documents. It is the single source of truth for "what we are working on."

## 🚨 CRITICAL: MessageCenter Loading Failure (P0)

- **Status:** 🔴 **BLOCKING** - Discovered 2026-01-03
- **Problem:** Store item messages completely broken in MessageCenter due to reintroduced cross-database query bug. Users cannot read or respond to store-related messages via `/messages` route.
- **Impact:** ~33% of message types non-functional. Users forced to use workaround (viewing messages on item detail pages).
- **Root Cause:** Recent refactor (commit ef8f696) accidentally reintroduced a previously-fixed bug where chat service tries to query `store_items` table that doesn't exist in its database.
- **Action:** Remove cross-database query from `socketHandlers.js` (15 min fix)
- **Details:** [ISSUE-messagecenter-loading-failure-2026-01-03.md](./01-proposed/ISSUE-messagecenter-loading-failure-2026-01-03.md)

---

## 🎯 Top Priority: Backend Test Coverage

- **Status:** 🔴 Not Started
- **Problem:** The main Go backend has 0% test coverage, which is a critical risk for production stability and future development. The `store-service` (87%+ coverage) should be used as the blueprint.
- **Action:** Implement a comprehensive testing suite, including unit and integration tests.
- **Details:** [ADR-backend-testing-strategy.md](./01-proposed/ADR-backend-testing-strategy.md)

---

## ⏳ Next Up

### 1. Browser Push Notifications
- **Status:** 🟡 Proposed
- **Goal:** Notify users of new messages even when they are offline or the app is in the background, significantly improving user re-engagement.
- **Details:** [DESIGN-browser-push-notifications.md](./01-proposed/DESIGN-browser-push-notifications.md)

### 2. Dedicated Authentication Service
- **Status:** 🟡 Proposed
- **Goal:** Improve security, scalability, and separation of concerns by extracting authentication into its own microservice.
- **Details:** [ADR-dedicated-auth-service.md](./01-proposed/ADR-dedicated-auth-service.md)

---

## ✅ Recently Completed

### Chat Service Refactor
- **Status:** ✅ Done
- **Summary:** Successfully refactored the `chat-websocket-service` to use its own dedicated database (`my_guy_chat`). This resolved numerous critical cross-database query bugs and stabilized the messaging functionality.
- **Details:** See logs in the [03-completed/](./03-completed/) directory.

### Message Limits Removal
- **Status:** ✅ Done
- **Summary:** Removed all artificial message limits from the frontend and backend to reduce user friction and encourage communication.
- **Details:** See `FIXLOG-frontend-message-limits-removal.md` and `FIXLOG-backend-message-limits-removal.md` in the `03-completed` directory.

---

## 📚 Key Architectural Documents
For onboarding and reference, these documents describe how the system is currently built.

- **Chat Service Architecture:** [ARCH-chat-service-architecture.md](./02-reference/ARCH-chat-service-architecture.md)
- **Deployment Workflow:** [REF-deployment-workflow.md](./02-reference/REF-deployment-workflow.md)
- **Testing Summary:** [REF-testing-summary.md](./02-reference/REF-testing-summary.md)
