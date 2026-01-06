# Chat Service Architecture - How It All Works
**Date:** January 2, 2026
**Status:** Documentation

---

## Overview

The chat service is a **standalone microservice** that operates independently from the Tasks and Store services. It's "invisible" to those services because they don't directly interact with it - all integration happens through the **frontend** and **shared authentication**.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         FRONTEND (Vue 3)                         │
│                      Port: 5173 (dev)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ TaskView.vue │  │StoreView.vue │  │  Chat.ts     │         │
│  │              │  │              │  │  (Pinia)     │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                  │                  │                  │
└─────────┼──────────────────┼──────────────────┼─────────────────┘
          │                  │                  │
          │ HTTP             │ HTTP             │ WebSocket
          │ (REST API)       │ (REST API)       │ (Socket.IO)
          │                  │                  │
┌─────────▼──────────┐ ┌─────▼──────────┐ ┌───▼─────────────────┐
│  Main Backend      │ │ Store Service  │ │  Chat Service       │
│  (Go - Port 8080)  │ │(Go - Port 8081)│ │ (Node - Port 8082)  │
├────────────────────┤ ├────────────────┤ ├─────────────────────┤
│ • Tasks API        │ │ • Store Items  │ │ • WebSocket Server  │
│ • Applications API │ │ • Bookings API │ │ • HTTP API          │
│ • Users API        │ │ • Bids API     │ │ • Message Storage   │
│ • Auth (JWT)       │ │ • Auth (JWT)   │ │ • Auth (JWT)        │
└────────┬───────────┘ └────────┬───────┘ └─────────┬───────────┘
         │                      │                    │
         │                      │                    │
    ┌────▼──────────────────────▼────────────────────▼─────┐
    │           PostgreSQL Database (Port 5432)            │
    ├──────────────────────────────────────────────────────┤
    │  ┌────────────┐  ┌──────────────┐  ┌──────────────┐ │
    │  │  my_guy    │  │ my_guy_store │  │ my_guy_chat  │ │
    │  │            │  │              │  │              │ │
    │  │ • users    │  │ • items      │  │ • messages   │ │
    │  │ • tasks    │  │ • bookings   │  │ • activity   │ │
    │  │ • apps     │  │ • bids       │  │ • warnings   │ │
    │  └────────────┘  └──────────────┘  └──────────────┘ │
    └──────────────────────────────────────────────────────┘
```

---

## How Services Communicate

### 1. **Service Independence** 🔒

**Key Principle:** Services do NOT call each other directly.

```
❌ DOES NOT HAPPEN:
Main Backend ──X──> Chat Service
Store Service ──X──> Chat Service
Chat Service  ──X──> Main Backend
Chat Service  ──X──> Store Service
```

**Why?**
- Services are decoupled
- No circular dependencies
- Each can scale independently
- Services can be deployed separately

---

### 2. **Frontend as Integration Layer** 🌐

The **frontend** is the only component that talks to all services:

```
✅ WHAT ACTUALLY HAPPENS:

Frontend ────────────> Main Backend (HTTP)
         │             (Get tasks, users, applications)
         │
         └──────────> Store Service (HTTP)
         │             (Get store items, bookings)
         │
         └──────────> Chat Service (WebSocket + HTTP)
                       (Real-time messages, message history)
```

---

## Authentication Flow

### Shared JWT Secret 🔑

All three services use the **same JWT secret**, enabling decentralized authentication:

**docker-compose.yml:**
```yaml
services:
  api:
    environment:
      - JWT_SECRET=your-secret-key-here  # ← Same secret

  store-service:
    environment:
      - JWT_SECRET=your-secret-key-here  # ← Same secret

  chat-websocket-service:
    environment:
      - JWT_SECRET=your-secret-key-here  # ← Same secret
```

### How It Works:

```
1. User Login
   Frontend ──→ Main Backend
                └─→ Generate JWT token
                    (signed with JWT_SECRET)
                └─→ Return token to frontend

2. User Access Tasks
   Frontend ──→ Main Backend
   Header: Authorization: Bearer <token>
           └─→ Verify token (using JWT_SECRET)
           └─→ Return task data

3. User Access Store
   Frontend ──→ Store Service
   Header: Authorization: Bearer <token>
           └─→ Verify token (using JWT_SECRET)
           └─→ Return store data

4. User Connect to Chat
   Frontend ──→ Chat Service
   Header: Authorization: Bearer <token>
           └─→ Verify token (using JWT_SECRET)
           └─→ Establish WebSocket connection
```

**Result:** No service needs to call another service to validate users!

---

## How Chat Integrates With Tasks

### Task Messaging Flow

```
┌─────────────────────────────────────────────────────────────┐
│ User viewing TaskDetailView.vue                              │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
         ┌────────────────────────────────┐
         │ 1. Load Task Details           │
         │    GET /api/v1/tasks/:id       │
         │    ──→ Main Backend (8080)     │
         └────────────────┬───────────────┘
                          │ Returns task data
                          ▼
         ┌────────────────────────────────┐
         │ 2. Connect to Chat             │
         │    WebSocket to Chat Service   │
         │    ──→ Chat Service (8082)     │
         │    Auth: Bearer <JWT token>    │
         └────────────────┬───────────────┘
                          │ Socket authenticated
                          ▼
         ┌────────────────────────────────┐
         │ 3. Join Task Conversation      │
         │    emit('join:conversation',   │
         │         { taskId: 123 })       │
         │    ──→ Chat Service            │
         └────────────────┬───────────────┘
                          │
                          ▼
         ┌────────────────────────────────┐
         │ 4. Load Message History        │
         │    GET /tasks/:id/messages     │
         │    ──→ Chat Service (8082)     │
         └────────────────┬───────────────┘
                          │ Returns messages
                          ▼
         ┌────────────────────────────────┐
         │ 5. Display Chat Interface      │
         │    Shows messages + input      │
         └────────────────────────────────┘
```

### Key Points:

1. **Task data** comes from Main Backend (8080)
2. **Message data** comes from Chat Service (8082)
3. **No direct communication** between backends
4. Frontend **coordinates** both data sources

---

## How Chat Integrates With Store

### Store Messaging Flow

```
┌─────────────────────────────────────────────────────────────┐
│ User viewing StoreItemView.vue                               │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
         ┌────────────────────────────────┐
         │ 1. Load Store Item             │
         │    GET /api/v1/store/:id       │
         │    ──→ Store Service (8081)    │
         └────────────────┬───────────────┘
                          │ Returns item data
                          ▼
         ┌────────────────────────────────┐
         │ 2. Load Store Messages         │
         │    GET /store-messages/:id     │
         │    ──→ Chat Service (8082)     │
         │    Auth: Bearer <JWT token>    │
         └────────────────┬───────────────┘
                          │ Returns messages
                          ▼
         ┌────────────────────────────────┐
         │ 3. Display Chat + Item         │
         │    Item details from Store     │
         │    Messages from Chat          │
         └────────────────────────────────┘
```

### Key Points:

1. **Store item data** comes from Store Service (8081)
2. **Message data** comes from Chat Service (8082)
3. **No cross-service calls** between backends
4. Frontend **merges** data from both sources

---

## Database Separation

### Three Separate Databases

```
┌──────────────────────────────────────────────────────┐
│             PostgreSQL Instance                       │
│                  (Port 5432)                          │
├──────────────────────────────────────────────────────┤
│                                                       │
│  ┌─────────────────┐    ┌─────────────────┐         │
│  │   my_guy        │    │  my_guy_store   │         │
│  ├─────────────────┤    ├─────────────────┤         │
│  │ • users         │    │ • store_items   │         │
│  │ • tasks         │    │ • bookings      │         │
│  │ • applications  │    │ • bids          │         │
│  │ • reviews       │    │ • purchases     │         │
│  └─────────────────┘    └─────────────────┘         │
│           ▲                      ▲                    │
│           │                      │                    │
│           │                      │                    │
│    Main Backend            Store Service              │
│    (Port 8080)             (Port 8081)                │
│                                                       │
│  ┌──────────────────────────────────────┐            │
│  │         my_guy_chat                  │            │
│  ├──────────────────────────────────────┤            │
│  │ • messages                           │            │
│  │   - task_id (reference)              │            │
│  │   - application_id (reference)       │            │
│  │   - store_item_id (reference)        │            │
│  │   - sender_id (reference)            │            │
│  │   - recipient_id (reference)         │            │
│  │   - content                          │            │
│  │   - created_at                       │            │
│  │                                      │            │
│  │ • user_activity                      │            │
│  │ • message_deletion_warnings          │            │
│  └──────────────────────────────────────┘            │
│           ▲                                           │
│           │                                           │
│    Chat Service                                       │
│    (Port 8082)                                        │
│                                                       │
└──────────────────────────────────────────────────────┘
```

### Reference Strategy: IDs Only, No Foreign Keys

The chat service **stores IDs** that reference entities in other databases, but **does NOT use foreign keys**:

**messages table in my_guy_chat:**
```sql
CREATE TABLE messages (
  id SERIAL PRIMARY KEY,

  -- References to entities in OTHER databases (no FK constraints)
  task_id INTEGER,           -- → my_guy.tasks.id
  application_id INTEGER,    -- → my_guy.applications.id
  store_item_id INTEGER,     -- → my_guy_store.store_items.id
  sender_id INTEGER,         -- → my_guy.users.id
  recipient_id INTEGER,      -- → my_guy.users.id

  content TEXT,
  created_at TIMESTAMP
);
```

**Why no foreign keys?**
- Can't create FK constraints across databases
- Services remain independent
- Chat service can be deployed separately
- Validation happens via **API calls** or **frontend logic**

---

## How The "Invisibility" Works

### Tasks Service Perspective

From the Main Backend's point of view:

```go
// backend/internal/api/handlers.go

func (h *Handler) GetTask(c *gin.Context) {
    // Get task from database
    task := h.taskService.GetTask(taskId)

    // Return task data
    c.JSON(200, task)

    // ❌ NO AWARENESS OF CHAT SERVICE
    // ❌ NO CALLS TO CHAT SERVICE
    // ❌ NO MESSAGE DATA INCLUDED
}
```

**The Main Backend:**
- ✅ Manages tasks, users, applications
- ✅ Returns task data via API
- ❌ Knows nothing about messages
- ❌ Never calls chat service

---

### Store Service Perspective

From the Store Service's point of view:

```go
// store-service/internal/handlers/items.go

func (h *ItemHandler) GetItem(c *gin.Context) {
    // Get store item from database
    item := h.itemService.GetItem(itemId)

    // Return item data
    c.JSON(200, item)

    // ❌ NO AWARENESS OF CHAT SERVICE
    // ❌ NO CALLS TO CHAT SERVICE
    // ❌ NO MESSAGE DATA INCLUDED
}
```

**The Store Service:**
- ✅ Manages store items, bookings, bids
- ✅ Returns store data via API
- ❌ Knows nothing about messages
- ❌ Never calls chat service

---

### Chat Service Perspective

From the Chat Service's point of view:

```javascript
// chat-websocket-service/src/services/messageService.js

async getStoreMessages(itemId, userId) {
  // Get messages from my_guy_chat database
  const query = `
    SELECT m.*
    FROM messages m
    WHERE m.store_item_id = $1
      AND (m.sender_id = $2 OR m.recipient_id = $2)
    ORDER BY m.created_at ASC
  `;

  const result = await db.query(query, [itemId, userId]);
  return result.rows;

  // ❌ NO AWARENESS OF STORE SERVICE
  // ❌ NO CALLS TO STORE SERVICE
  // ❌ NO STORE ITEM DATA INCLUDED
}
```

**The Chat Service:**
- ✅ Manages messages, conversations, typing indicators
- ✅ Returns message data via API/WebSocket
- ❌ Knows nothing about tasks or store items
- ❌ Never calls other services
- ❌ Only stores **IDs** that reference other entities

---

## Frontend Integration Code

### How Frontend Coordinates Everything

**Example: TaskDetailView.vue**

```typescript
// Load task data from Main Backend
const task = await fetch('http://localhost:8080/api/v1/tasks/123')
  .then(r => r.json());

// Connect to Chat Service
const socket = io('http://localhost:8082', {
  auth: { token: localStorage.getItem('token') }
});

// Join task conversation
socket.emit('join:conversation', { taskId: 123 });

// Load message history from Chat Service
const messages = await fetch('http://localhost:8082/api/v1/tasks/123/messages', {
  headers: { Authorization: `Bearer ${token}` }
}).then(r => r.json());

// Now display both:
// - Task details (from Main Backend)
// - Messages (from Chat Service)
```

**Result:** The user sees a unified interface with task info + messages, but the data comes from **two completely separate services**.

---

## Benefits of This Architecture

### 1. **Service Independence** 🔓
- Chat can be down without affecting tasks/store
- Tasks can be updated without touching chat
- Store can scale independently

### 2. **Database Isolation** 🗄️
- Chat messages in separate database
- Tasks/Store don't slow down from chat queries
- Can optimize each database separately

### 3. **Technology Flexibility** 🛠️
- Chat uses Node.js (good for WebSocket)
- Backend uses Go (good for REST APIs)
- Each service uses best technology for its needs

### 4. **Scalability** 📈
- Can scale chat service independently
- Can add more chat instances for WebSocket connections
- Backend/Store can scale separately

### 5. **Deployment** 🚀
- Deploy chat updates without touching backend
- Roll back chat without affecting other services
- Different release cycles for each service

---

## Message Flow Example (End-to-End)

### Scenario: User sends message about a task

```
1. User types message in TaskDetailView.vue
   └─→ Frontend captures: "Is this still available?"

2. Frontend emits WebSocket event
   socket.emit('message:send', {
     taskId: 123,
     recipientId: 456,
     content: "Is this still available?"
   })
   └─→ Goes to Chat Service (8082)

3. Chat Service receives event
   └─→ Validates JWT token
   └─→ Filters content (removes profanity/PII)
   └─→ Saves to my_guy_chat.messages table
       INSERT INTO messages (task_id, sender_id, recipient_id, content)
       VALUES (123, 789, 456, "Is this still available?")
   └─→ Emits 'message:new' to all sockets in task room

4. Other user's browser receives 'message:new' event
   └─→ Frontend updates messages array
   └─→ New message appears in chat UI

5. Message persisted in chat database
   └─→ Can be retrieved later via GET /tasks/123/messages
```

**Note:** Main Backend never knew this message was sent!

---

## Security Considerations

### 1. **Shared JWT Secret** 🔐

**Risk:** If one service is compromised, all services are at risk.

**Mitigation:**
- Use strong, random JWT secret
- Rotate secret periodically
- Use environment variables (never hardcode)
- Consider asymmetric keys (RS256) for production

### 2. **Cross-Service Validation** ✅

**Problem:** Chat service can't verify if task/item actually exists (different database).

**Current Solution:** Frontend ensures valid IDs before sending.

**Future Enhancement:** ValidationService calls other APIs to verify entities exist.

### 3. **Access Control** 🚫

**Chat Service checks:**
- Is user authenticated? (JWT valid)
- Is user participant in conversation? (sender or recipient)
- Message content safe? (filtered)

**What it CAN'T check:**
- Does task still exist?
- Is task assigned/open?
- Is store item still available?

**Solution:** Frontend handles these checks before showing chat UI.

---

## Configuration Reference

### Environment Variables

**Main Backend (.env):**
```bash
PORT=8080
JWT_SECRET=your-secret-key-here
DB_NAME=my_guy
```

**Store Service (.env):**
```bash
PORT=8081
JWT_SECRET=your-secret-key-here  # ← Must match
DB_NAME=my_guy_store
```

**Chat Service (.env):**
```bash
PORT=8082
JWT_SECRET=your-secret-key-here  # ← Must match
DATABASE_URL=postgresql://...my_guy_chat
MAIN_API_URL=http://api:8080/api/v1      # For future validation
STORE_API_URL=http://store-service:8081/api/v1  # For future validation
```

**Frontend (.env):**
```bash
VITE_API_URL=http://localhost:8080/api/v1
VITE_CHAT_API_URL=http://localhost:8082/api/v1
VITE_CHAT_WS_URL=http://localhost:8082
```

---

## Future Enhancements

### 1. **ValidationService** 🔍

Add service-to-service validation:

```javascript
// chat-websocket-service/src/services/validationService.js

async validateTask(taskId, token) {
  const response = await fetch(`${MAIN_API_URL}/tasks/${taskId}`, {
    headers: { Authorization: `Bearer ${token}` }
  });
  return response.ok;
}
```

**Benefits:**
- Chat service can verify task exists
- Can check if task is still open
- Can enforce business rules

### 2. **Event Bus** 📡

Add shared event system (e.g., Redis Pub/Sub, RabbitMQ):

```
Task Assigned Event:
Main Backend ──→ Event Bus ──→ Chat Service
                              └─→ Update message limits
```

**Benefits:**
- Services stay decoupled
- Real-time updates across services
- Better data consistency

### 3. **API Gateway** 🚪

Add single entry point:

```
Frontend ──→ API Gateway ──→ Main Backend
                         └──→ Store Service
                         └──→ Chat Service
```

**Benefits:**
- Single authentication point
- Rate limiting
- Request routing
- Load balancing

---

## Troubleshooting

### "Messages not showing"

1. Check Chat Service is running: `docker-compose ps chat-websocket-service`
2. Check WebSocket connection: Browser console → Network → WS tab
3. Verify JWT token is valid: Check localStorage.getItem('token')
4. Check chat database: `docker-compose exec postgres-db psql -U postgres -d my_guy_chat -c "SELECT COUNT(*) FROM messages;"`

### "Can't send messages"

1. Verify authenticated: Check socket.connected in console
2. Check JWT token included in socket auth
3. Verify recipient_id is correct
4. Check backend logs: `docker-compose logs chat-websocket-service`

### "Messages from wrong database"

Make sure each service uses correct database:
- Main Backend: `my_guy`
- Store Service: `my_guy_store`
- Chat Service: `my_guy_chat`

---

## Summary

**How Chat Service is "Invisible":**

1. ✅ **No Direct Service Calls** - Services never call each other
2. ✅ **Shared Authentication** - Same JWT secret enables decentralized auth
3. ✅ **Frontend Integration** - Frontend coordinates all services
4. ✅ **ID References Only** - Chat stores IDs, not actual data
5. ✅ **Separate Databases** - Complete data isolation
6. ✅ **Independent Deployment** - Each service deploys separately

**The chat service appears "invisible" to Tasks and Store services because they genuinely don't know it exists!** All integration happens through the frontend, which acts as the orchestrator of all three microservices.

---

## Recent Updates

### January 5, 2026: Service Startup Issues Resolved

Fixed three critical module import failures that prevented the chat service from starting. See [FIXLOG-chat-service-startup-failure.md](../03-completed/FIXLOG-chat-service-startup-failure.md) for details:

1. **Database import path**: Updated `bookingMessageService.js` to use correct path (`../config/database`)
2. **node-fetch removal**: Removed unnecessary import (fetch is built-in to Node 18+)
3. **Auth middleware name**: Fixed `authenticateJWT` → `authenticateHTTP` in `bookingNotifications.js`

**Impact**: Chat service now starts successfully, restoring real-time messaging and booking notifications.

---

**Document Created:** January 2, 2026
**Last Updated:** January 5, 2026
**Architecture Pattern:** Microservices with Frontend Integration Layer
**Authentication:** Shared JWT Secret (Decentralized)
**Communication:** HTTP REST + WebSocket (Socket.IO)
