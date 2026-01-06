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

-   **Real-time Messaging**: Instant message delivery, typing indicators, and online presence.
-   **Conversation Management**: Groups messages by context (task, store item) and tracks unread counts.
-   **Message Lifecycle**: Supports editing (with history), soft deletion, and read receipts.
-   **Content Filtering**: Automatically removes URLs, emails, and phone numbers from messages to ensure user privacy.
-   **Automated Deletion**: A scheduler (`node-cron`) automatically deletes old messages based on predefined rules (e.g., 6 months after task completion).

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

Authentication is performed by passing a JWT in the `auth.token` field upon connection.

### Key Events (Client → Server)
-   `join:conversation`: Join a room for a specific task or item (`{ taskId: 1 }` or `{ itemId: 2 }`).
-   `message:send`: Send a message (`{ recipientId: 1, content: 'Hello!', ... }`).
-   `message:edit`: Edit a message (`{ messageId: 1, content: 'New content' }`).
-   `message:delete`: Soft-delete a message (`{ messageId: 1 }`).
-   `conversation:read`: Mark all messages in a conversation as read (`{ taskId: 1 }`).
-   `conversations:list`: Request the list of all conversations for the user.
-   `messages:get`: Request a paginated history of messages for a conversation.
-   `typing:start` / `typing:stop`: Manage typing indicators.

### Key Events (Server → Client)
-   `message:new`: A new message has arrived.
-   `message:edited` / `message:deleted`: A message was changed.
-   `user:typing` / `user:stopped-typing`: Typing indicator updates.
-   `conversations:list`: The user's list of conversations.
-   `error`: An error occurred (`{ message: 'Error description' }`).

## 8. REST API

-   `GET /health`: Health check endpoint to verify the service is running.
-   `GET /api/v1/deletion-warnings`: Get pending deletion warnings for the authenticated user.
-   `POST /api/v1/deletion-warnings/:id/shown`: Mark a warning as acknowledged.

## 9. Message Lifecycle

1.  **Creation**: A client sends `message:send`. The server filters content, saves to the `messages` table, and emits `message:new` to the recipient.
2.  **Editing**: A client sends `message:edit`. The server verifies ownership, updates the record, and emits `message:edited`.
3.  **Deletion**: A client sends `message:delete`. The server soft-deletes the message (replaces content with "[Message deleted]") and emits `message:deleted`.
4.  **Auto-Deletion**: A daily cron job checks for old conversations tied to completed/inactive tasks and schedules them for permanent deletion, notifying users 30 days in advance.

## 10. Security & Filtering

-   **Authentication**: All socket connections and REST endpoints are protected and require a valid JWT.
-   **Authorization**: Business logic verifies that users can only access or modify their own messages and conversations.
-   **Content Filtering**: To protect user privacy, the following patterns are automatically removed from message content before storage:
    -   URLs (e.g., `http://example.com`)
    -   Emails (e.g., `user@example.com`)
    -   Phone numbers
    -   Social media handles (`@username`)
    The original, unfiltered content is stored separately for auditing but is never exposed to clients.
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
