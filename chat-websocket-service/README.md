# Chat WebSocket Service - MyGuy Platform

This real-time messaging microservice for the MyGuy platform handles WebSocket connections, message delivery and management, conversation state, and content filtering.

## Table of Contents

1.  [Architecture](#1-architecture)
2.  [Features](#2-features)
3.  [Technology Stack](#3-technology-stack)
4.  [Getting Started (Development)](#4-getting-started-development)
5.  [Configuration](#5-configuration)
6.  [Database and Migrations](#6-database-and-migrations)
7.  [WebSocket API](#7-websocket-api)
8.  [REST API](#8-rest-api)
9.  [Message Lifecycle](#9-message-lifecycle)
10. [Security & Filtering](#10-security--filtering)
11. [Troubleshooting & Common Issues](#11-troubleshooting--common-issues)

---

## 1. Architecture

The service operates as a standalone Node.js application, managing all real-time communication between clients. It connects to its own dedicated PostgreSQL database (`my_guy_chat`) and uses a shared JWT secret for stateless authentication.

**Key Design Principles:**
-   **Unified Message Table**: A single `messages` table handles all message types (for tasks, store items, etc.), distinguished by foreign key columns (`task_id`, `store_item_id`).
-   **Service Independence**: The chat service does not query other services' databases directly. It communicates only through IDs, with the frontend responsible for fetching associated data (like user or task details).
-   **Real-time & REST**: Provides a rich WebSocket API for real-time events and a REST API for stateless operations like health checks.

## 2. Features

-   **Real-time Messaging**: Instant delivery, typing indicators, and read receipts — sent only to the two participants of a conversation.
-   **Conversations**: A conversation is a context (task, application or store item) **plus the other person**, so a seller's chats with different buyers of one item, or a task's chats with different applicants, stay separate. Unread counts are tracked per conversation.
-   **Task events**: The main API posts system messages (new application, accepted, declined, cancelled) into the owner↔applicant task conversation. They are stored as `system_alert` and can't be edited or deleted.
-   **Booking messages**: The store service posts booking requests, which participants approve, decline and rate from the chat.
-   **Message Lifecycle**: Editing, soft deletion, and read receipts.
-   **Content Filtering**: Masks URLs, emails, phone numbers and social handles in messages until the two people are matched (accepted gig application or approved booking).
-   **Automated Deletion**: A scheduler flags old messages on finished tasks for deletion, warning users 30 days ahead.
-   **Privacy**: Online status isn't broadcast; "last seen" is only shared with people you've exchanged messages with. Application chats are limited to the task owner and the applicant (checked against the main API).

## 3. Technology Stack

-   **Runtime**: Node.js 18+
-   **Framework**: Express 4.18+
-   **WebSocket**: Socket.IO 4.7+
-   **Database**: PostgreSQL with `node-postgres` (pg)
-   **Authentication**: JSON Web Tokens (JWT)
-   **Scheduling**: `node-cron`
-   **Logging**: Winston

## 4. Getting Started (Development)

### Prerequisites
-   Node.js 18+
-   PostgreSQL 12+
-   Docker (recommended)

### Local Setup
1.  **Clone Repository:**
    ```bash
    git clone <repository-url>
    cd chat-websocket-service
    ```
2.  **Install Dependencies:**
    ```bash
    npm install
    ```
3.  **Configure Environment:**
    -   Create a `.env` file from `.env.example`.
    -   Set `DATABASE_URL` to your `my_guy_chat` database instance.
    -   Set `JWT_SECRET` to match the other services.
4.  **Run Migrations:**
    ```bash
    npm run migrate
    ```
5.  **Start the Service:**
    ```bash
    npm run dev
    ```

### Docker Development
The service is included in the project's root `docker-compose.yml`.
```bash
# From project root
docker-compose up --build chat-websocket-service
```

## 5. Configuration

Key environment variables are defined in `.env`:

-   `PORT`: The port for the service to run on (e.g., 8082).
-   `DATABASE_URL`: Connection string for the PostgreSQL database.
-   `JWT_SECRET`: The shared secret for validating JWTs.
-   `CLIENT_URL`: The URL of the frontend client for CORS configuration.
-   `LOG_LEVEL`: Logging verbosity (e.g., `info`, `debug`).

## 6. Database and Migrations

The service connects to its own `my_guy_chat` database. The schema is managed by `node-pg-migrate`, which tracks executed migrations in a database table named `pgmigrations`.

### Key Tables
-   **`messages`**: A unified table for all messages. The message context is determined by which foreign key column (`task_id`, `store_item_id`, etc.) is populated.
-   **`user_activity`**: Tracks user presence and last seen status.
-   **`message_deletion_warnings`**: Logs upcoming automated message deletions.

### Migrations
Migrations are handled via npm scripts. They run automatically on service startup.

-   **Run pending migrations:**
    ```bash
    npm run migrate
    ```
-   **Create a new migration:**
    ```bash
    npm run migrate:create <migration_name>
    ```

## 7. WebSocket API

Authentication is performed by passing a JWT in the `auth.token` field upon connection. Each user joins a personal room (`user:<id>`), and all message events are delivered there — never to shared conversation rooms.

A conversation is addressed by its context (`taskId`, `applicationId` or `itemId`) and `otherUserId`.

### Key Events (Client → Server)
-   `message:send`: `{ taskId | applicationId | itemId, recipientId, content }`. Application messages must be between the task owner and the applicant.
-   `message:edit` / `message:delete`: `{ messageId, content? }` (own messages only; never system messages).
-   `messages:get`: `{ <context>, otherUserId, limit, offset }` — a page of history.
-   `conversation:read`: `{ <context>, otherUserId }` — mark it read.
-   `conversations:list`: the user's conversations with unread counts.
-   `typing:start` / `typing:stop`: `{ <context>, recipientId }`.
-   `user:lastseen`: `{ userId }` — answered only for people you've chatted with.

### Key Events (Server → Client)
-   `message:new` / `message:sent`: a message (including system and booking messages).
-   `message:edited` / `message:deleted` / `message:updated` / `message:read`.
-   `messages:list`: `{ <context>, otherUserId, messages, offset, totalCount }`.
-   `conversation:marked-read`, `conversations:list`, `conversations:refresh`.
-   `user:typing` / `user:stopped-typing`: `{ userId, <context> }`.
-   `error`: `{ message }`.

## 8. REST API

-   `GET /health`
-   `GET /api/v1/conversations` — conversations (HTTP fallback).
-   `GET|POST /api/v1/tasks/:taskId/messages`, `GET|POST /api/v1/applications/:applicationId/messages`, `GET|POST /api/v1/store-messages…` — message history and sending over HTTP.
-   `GET /api/v1/users/:id/last-seen` — only for people you've chatted with.
-   `GET /api/v1/deletion-warnings`, `POST /api/v1/deletion-warnings/:id/shown`.
-   `POST /api/v1/booking-action` — approve/decline/confirm/rate a booking from chat.

### Internal (service-to-service, `X-Internal-API-Key: $INTERNAL_API_KEY`)
-   `POST /api/v1/internal/booking-created` — from the store service.
-   `POST /api/v1/internal/task-message` — from the main API: `{ task_id, sender_id, recipient_id, content }`, stored as `system_alert`.
-   `POST /api/v1/internal/store-message` — from store-service: `{ store_item_id, sender_id, recipient_id, content }`, stored as `system_alert` in that item's conversation (a listing made for someone's request).

## 9. Message Lifecycle

1.  **Creation**: A client sends `message:send`. The server filters content, saves to the `messages` table, and emits `message:new` to the recipient's and the sender's personal rooms.
2.  **Editing**: A client sends `message:edit`. The server verifies ownership, updates the record, and emits `message:edited`.
3.  **Deletion**: A client sends `message:delete`. The server soft-deletes the message (replaces content with "[Message deleted]") and emits `message:deleted`.
4.  **Auto-Deletion**: A daily cron job checks for old conversations tied to completed/inactive tasks and schedules them for permanent deletion, notifying users 30 days in advance.

## 10. Security & Filtering

-   **Authentication**: All socket connections and REST endpoints are protected and require a valid JWT.
-   **Authorization**: Users only ever receive and read messages they sent or received; application chats are verified against the main API (`GET /applications/:id/participants`), failing closed if it is unreachable.
-   **Content Filtering**: Until two people are matched, these are masked in message content before storage:
    -   URLs (e.g., `http://example.com`, `www.shop.ug`, `john.dev/x`)
    -   Emails (e.g., `user@example.com`)
    -   Phone numbers, local and international (e.g., `0772 123 456`, `+256 772 123456`) — prices like `120,000` are left alone
    -   Social media handles (`@username`)
    The patterns are tested against `shared/contact-filter-cases.json`, which the backend's gig-text check also uses.
-   **Contact Unlocks**: When the poster accepts an application (the backend sends `unlock_contacts: true` to `/internal/task-message`) or a seller approves a booking (`/booking-action`), the pair is recorded in `contact_unlocks` and their messages in that conversation are no longer filtered. Application chats never unlock.
-   **Input Validation**: Message length and payload structure are validated.

## 11. Troubleshooting & Common Issues

### Service Won't Start

**Symptom**: Service crashes immediately after migrations complete, or Docker container shows `Exited` status.

**Common Causes**:

1. **Module Import Errors**
   - **Issue**: Incorrect import paths (e.g., `require('../db')` instead of `require('../config/database')`)
   - **Fix**: Verify all imports point to existing modules with correct relative paths
   - **Note**: Node 18+ has built-in `fetch` - don't import `node-fetch`

2. **Middleware Export Mismatches**
   - **Issue**: Importing non-existent exports (e.g., `authenticateJWT` vs `authenticateHTTP`)
   - **Fix**: Check `src/middleware/auth.js` exports: `authenticateHTTP`, `authenticateSocket`, `verifyToken`

3. **Docker Build Cache**
   - **Issue**: Code changes not reflected in running container
   - **Fix**: Always rebuild after code changes: `docker-compose up -d --build chat-websocket-service`

**Debugging Steps**:
```bash
# Check service status
docker-compose ps chat-websocket-service

# View detailed logs
docker-compose logs chat-websocket-service

# Rebuild and restart
docker-compose up -d --build chat-websocket-service
```

### Frontend Connection Errors

**Symptom**: Browser console shows repeated WebSocket connection failures:
```
Chat connection attempt [N] failed: websocket error
⚠️ Chat service unavailable after multiple connection attempts
```

**Causes & Solutions**:

1. **Service Not Running**: Check `docker-compose ps` - ensure chat-websocket-service is `Up`
2. **Wrong WebSocket URL**: Verify `VITE_CHAT_WS_URL=http://localhost:8082` in frontend `.env`
3. **Invalid JWT Token**: Check browser localStorage for valid token, re-login if needed
4. **CORS Issues**: Ensure `CLIENT_URL` environment variable matches frontend URL

### Messages Not Saving

**Symptom**: Messages appear in UI but don't persist after refresh.

**Debugging**:
```bash
# Check database connection
docker-compose exec postgres-db psql -U postgres -d my_guy_chat -c "SELECT COUNT(*) FROM messages;"

# View recent messages
docker-compose exec postgres-db psql -U postgres -d my_guy_chat -c "SELECT id, content, created_at FROM messages ORDER BY created_at DESC LIMIT 10;"
```

**Common Fixes**:
- Verify `DATABASE_URL` points to `my_guy_chat` database
- Check migration status: `npm run migrate`
- Review service logs for database errors

### For More Details

See engineering documentation:
- **Architecture**: [../engineering/02-reference/ARCH-chat-service-architecture.md](../engineering/02-reference/ARCH-chat-service-architecture.md)
- **Recent Fixes**: [../engineering/03-completed/FIXLOG-chat-service-startup-failure.md](../engineering/03-completed/FIXLOG-chat-service-startup-failure.md)

---

**Last Updated**: January 5, 2026
