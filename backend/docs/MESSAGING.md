# Messaging in MyGuy

Messaging is owned by the chat service (`chat-websocket-service/`, port 8082), not the backend. Its README is the maintained reference: the WebSocket events, the REST and internal endpoints, conversation identity, contact unlocks and when a chat is locked or ended.

The backend's part:

- **Gig events are messages.** The backend posts each gig event to chat's `POST /api/v1/internal/task-message` (`internal/chatnotify`, best effort, with `INTERNAL_API_KEY`), stored as a `system_alert` with `metadata.event` (`application`, `accepted`, `declined`, `cancelled`, `done`, `not_done`, `completed`). Accepting an application sends `unlock_contacts: true`, which records the pair as matched.
- **Matches made before chat recorded them** are re-sent at startup (`UnlockMatchedChats`).
- **Application chats** (if any exist) are authorized through the backend's `GET /api/v1/applications/:id/participants`.
