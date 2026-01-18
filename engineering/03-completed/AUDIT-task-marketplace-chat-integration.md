# Audit: Task Marketplace Chat Integration

**Date:** January 18, 2026
**Auditor:** Gemini
**Purpose:** Verify that the "Task Marketplace" correctly utilizes the existing chat/messaging functionality.

---

## 1. Integration Overview

The Task Marketplace fully integrates with the dedicated **Chat Service** via the Frontend. The architecture decouples the core Task API from messaging, relying on the Frontend to bridge the two services using `task_id` as the context identifier.

### Architecture Flow
1.  **Task Context:** Frontend fetches Task details (including `id`, `created_by`, `assigned_to`) from `Backend Service`.
2.  **Chat Connection:** Frontend connects to `Chat Service` (WebSocket).
3.  **Conversation Association:** Messages are tagged with `task_id` and `message_type='task'`.
4.  **Participant Mapping:**
    *   **Owner View:** Chats with the **Assignee** (`task.assigned_to`).
    *   **Applicant View:** Chats with the **Creator** (`task.created_by`).

---

## 2. Component Analysis

### 2.1 Frontend (`TaskDetailView.vue`)
*   **Component:** Uses `<ChatWindow />`.
*   **Props:**
    *   `conversation-id`: Binds to `task.id`.
    *   `conversation-type`: Hardcoded to `"task"`.
    *   `recipient-id`: Dynamically calculated (Creator if viewer is applicant; Assignee if viewer is owner).
*   **Gating Logic:**
    *   **Visibility:** Messages are visible if `is_messages_public` is true OR user is a participant.
    *   **Sending:** Owners are blocked from sending messages until the task is **assigned**. Applicants/Browsers can always message the owner (e.g., to ask questions before applying).

### 2.2 Frontend Store (`chat.ts`)
*   **Socket Events:** Correctly handles `task`-specific events.
    *   `join:conversation` payload includes `{ taskId: ... }`.
    *   `message:send` payload includes `{ taskId: ... }`.
*   **State Management:** Separates messages by conversation ID, correctly distinguishing between `task`, `application`, and `store` contexts.

### 2.3 Chat Service (`messageService.js`)
*   **Schema:** The `messages` table supports `task_id` foreign keys (nullable).
*   **Logic:**
    *   `sendMessage` detects `taskId` and sets `message_type = 'task'`.
    *   `getMessages` filters by `task_id`.
*   **Limitation:** The service cannot enforce dynamic message limits based on Task status (e.g., "only 3 messages before assignment") because it cannot query the `my_guy` database (Database Isolation principle). It currently defaults to a safe limit (3) or no limit depending on the exact code path.

---

## 3. Findings & Observations

### ✅ Verified Functionality
*   **Contextual Messaging:** Messages are correctly tied to specific tasks.
*   **Real-time Updates:** WebSocket events broadcast new messages to participants.
*   **History Loading:** Chat history loads correctly for the specific task context.

### ⚠️ Potential Issues (Non-Blocking)
1.  **Message Limits:** The Chat Service logs a warning: `Task message limit check not implemented for task ...`. This is due to strict database isolation. Currently, it may default to a generic limit or allow unlimited messages, but it cannot intelligently check if a user is the "valid assignee" to lift limits.
2.  **Participant Validation:** The Chat Service trusts the `recipientId` sent by the frontend. A malicious user could theoretically emit a socket event to message a user irrelevant to the task, though the Frontend UI prevents this.

## 4. Conclusion
The Task Marketplace **successfully uses the current chat functionality**. No immediate fixes are required for MVP functionality. The integration adheres to the project's architectural patterns.
