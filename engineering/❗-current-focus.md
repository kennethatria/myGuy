# Engineering Priorities: Q1 2026

This document provides a high-level summary of the current engineering focus, upcoming priorities, and links to relevant documents. It is the single source of truth for "what we are working on."

## 🎯 Top Priority: Backend Test Coverage

- **Status:** 🔴 Not Started
- **Problem:** The main Go backend has 0% test coverage, which is a critical risk for production stability and future development. The `store-service` (87%+ coverage) should be used as the blueprint.
- **Action:** Implement a comprehensive testing suite, including unit and integration tests.
- **Details:** [ADR-backend-testing-strategy.md](./01-proposed/ADR-backend-testing-strategy.md)

---

## ⏳ Next Up

### 1. Booking Requests via Messages
- **Status:** 🟡 Proposed (Critical UX)
- **Goal:** Move booking requests from hidden item pages to the main `/messages` view to ensure sellers actually see and respond to them.
- **Details:** [RFC-booking-via-messages.md](./01-proposed/RFC-booking-via-messages.md)

### 2. Descriptive Conversation Titles
- **Status:** 🟡 Proposed (UX Priority)
- **Goal:** Replace the generic "Conversation" label with the actual Store Item or Task title in the message list to help users distinguish between different chats.
- **Details:** [RFC-conversation-titles.md](./01-proposed/RFC-conversation-titles.md)

### 3. Browser Push Notifications
- **Status:** 🟡 Proposed
- **Goal:** Notify users of new messages even when they are offline or the app is in the background, significantly improving user re-engagement.
- **Details:** [DESIGN-browser-push-notifications.md](./01-proposed/DESIGN-browser-push-notifications.md)

### 3. Dedicated Authentication Service
- **Status:** 🟡 Proposed
- **Goal:** Improve security, scalability, and separation of concerns by extracting authentication into its own microservice.
- **Details:** [ADR-dedicated-auth-service.md](./01-proposed/ADR-dedicated-auth-service.md)

---

## ✅ Recently Completed

### Empty Message Body Fix
- **Status:** ✅ Done
- **Summary:** Fixed a critical bug where the message body appeared empty for store item conversations. Addressed a frontend logic error and type mismatch in the chat store state management.
- **Details:** See `FIXLOG-empty-message-body.md` in the `03-completed` directory.

### MessageCenter Loading Failure
- **Status:** ✅ Done
- **Summary:** Resolved the blocking issue where store item messages failed to load. Removed unused and problematic `StoreMessageHandler` code that contained cross-database queries.
- **Details:** See `FIXLOG-messagecenter-loading-failure.md` in the `03-completed` directory.

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
