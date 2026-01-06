# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

MyGuy is a microservices-based task marketplace platform where users can post tasks, apply for tasks, communicate in real-time, and buy/sell items. The architecture separates concerns into distinct services: task management (Go), real-time chat (Node.js), and a store/bidding marketplace (Go), each with its own PostgreSQL database.

---

## 🏗 Architecture & Service Topology

| Service | Language/Framework | Port | Database | Purpose |
|---------|-------------------|------|----------|---------|
| **Backend** | Go (Gin) | 8080 | `my_guy` | Core task marketplace API (users, tasks, applications, reviews) |
| **Store Service** | Go (Gin) | 8081 | `my_guy_store` | Marketplace for items with fixed-price and auction bidding |
| **Chat Service** | Node.js (Express + Socket.IO) | 8082 | `my_guy_chat` | Real-time WebSocket messaging service |
| **Frontend** | Vue 3 + TypeScript (Vite) | 5173 | - | Single-page application |
| **Database** | PostgreSQL 15 | 5432 (exposed as 5433) | - | Shared database server with multiple databases |

---

## 📜 Critical Architecture Principles

### 1. Database Isolation
**Services never query each other's databases directly.** Each service has its own database. The frontend is responsible for fetching associated data (like user or task details) from the appropriate services.

### 2. Shared Authentication
**All services use a unified `JWT_SECRET`** for independent validation. Store and chat services perform automatic user synchronization via JWT middleware to maintain local user caches.

### 3. User Privacy & Content Filtering
**The Chat Service must automatically filter PII:** URLs, emails, phone numbers, and social media handles are stripped from messages to protect user privacy.

### 4. Service Blueprint Pattern
**Use `store-service` (92%+ test coverage) as the architectural blueprint** for any new Go development, testing strategies, or service patterns. This service demonstrates:
- Comprehensive test coverage (unit + integration)
- Clean architecture (handlers → services → repositories)
- Proper error handling and validation
- CI/CD with coverage enforcement

### 5. Unified Message Table
The chat service uses a single `messages` table for all message types (tasks, store items, applications), distinguished by foreign key columns (`task_id`, `store_item_id`, `application_id`).

---

## 📂 Engineering Documentation & Status Tracking

**ALWAYS check `engineering/` directory first for project health and priorities:**

### Start Here:
- **`engineering/❗-current-focus.md`** - Current Q1 2026 priorities, P0/P1/P2 status
- **`engineering/01-proposed/ROADMAP-mvp-prioritization.md`** - Detailed MVP roadmap

### Reference:
- **`engineering/01-proposed/`** - ADRs (Architecture Decision Records), RFCs, TODOs
- **`engineering/02-reference/`** - Architecture diagrams, patterns, guides
- **`engineering/03-completed/`** - Historical fixes, implementation logs, FIXLOGs

### Current Top Priorities (Q1 2026):
1. **P1: Backend Filtering for Store Items** - Prevent performance collapse
2. **P1: Backend Testing Foundation** - Reduce regression risk (0% coverage → use store-service blueprint)
3. **P2: TypeScript Type Errors** - 62 errors tracked in `01-proposed/TODO-typescript-errors.md`

---

## 🛠 Common Development Commands

### Running the Full Platform
```bash
# Start all services (recommended)
docker-compose up --build

# View logs for specific service
docker-compose logs -f [api|store-service|chat-websocket-service|frontend]

# Stop all services
docker-compose down
```

### Backend (Go - Main API)
```bash
cd backend

# Run locally
go run cmd/api/main.go

# Build
go build -o backend cmd/api/main.go

# Note: Currently 0% test coverage - TOP PRIORITY
# See: engineering/01-proposed/ADR-backend-testing-strategy.md
```

### Store Service (Go - THE BLUEPRINT)
```bash
cd store-service

# Testing (92%+ coverage - use as reference for backend)
make test                    # Run all tests
make test-unit              # Unit tests only
make test-integration       # Integration tests only
make test-coverage          # Generate coverage report with HTML
make test-coverage-check    # Verify coverage ≥70%
make test-watch             # Watch mode (requires entr)

# Other commands
make build                  # Build binary
make lint                   # Lint code
make fmt                    # Format code
make help                   # Show all available commands
```

### Chat Service (Node.js)
```bash
cd chat-websocket-service

# Install dependencies
npm install

# Development
npm run dev

# Production
npm start

# Database migrations
npm run migrate                    # Run pending migrations
npm run migrate:create <name>      # Create new migration

# Testing
npm test
npm run lint
```

### Frontend (Vue 3 + TypeScript)
```bash
cd frontend

# Development
npm install
npm run dev

# Build & Preview
npm run build
npm run preview

# Testing
npm run test:unit          # Vitest unit tests
npm run test:e2e           # Playwright E2E tests

# Code Quality
npm run lint               # ESLint with auto-fix
npm run format             # Prettier formatting
npm run type-check         # TypeScript type checking
```

---

## ⚙️ Environment Configuration

Each service requires a `.env` file. Copy from `.env.example` or create with these variables:

### Backend
```env
PORT=8080
JWT_SECRET=your-secret-key-here  # MUST match across all services
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=mysecretpassword
DB_NAME=my_guy
DB_SSL_MODE=disable
```

### Store Service
```env
PORT=8081
JWT_SECRET=your-secret-key-here  # MUST match across all services
DB_CONNECTION=host=localhost user=postgres password=mysecretpassword dbname=my_guy_store port=5432 sslmode=disable
CHAT_API_URL=http://localhost:8082/api/v1
INTERNAL_API_KEY=your-internal-api-key-change-in-production
```

### Chat Service
```env
PORT=8082
NODE_ENV=development
JWT_SECRET=your-secret-key-here  # MUST match across all services
DB_CONNECTION=postgresql://postgres:mysecretpassword@localhost:5432/my_guy_chat
DATABASE_URL=postgresql://postgres:mysecretpassword@localhost:5432/my_guy_chat
CLIENT_URL=http://localhost:5173
MAIN_API_URL=http://localhost:8080/api/v1
STORE_API_URL=http://localhost:8081/api/v1
INTERNAL_API_KEY=your-internal-api-key-change-in-production
```

### Frontend
```env
VITE_API_URL=http://localhost:8080/api/v1
VITE_STORE_API_URL=http://localhost:8081/api/v1
VITE_CHAT_API_URL=http://localhost:8082/api/v1
VITE_CHAT_WS_URL=http://localhost:8082
```

**CRITICAL:**
- `JWT_SECRET` must be identical across all services for authentication to work
- `INTERNAL_API_KEY` must match between chat and store services for inter-service communication
  - **Required for booking notifications**: Without this, booking requests won't appear in Messages
  - Store service uses this to notify chat service when bookings are created
  - Chat service validates this key before creating notification messages

---

## 📁 Project Structure

```
myGuy/
├── backend/                 # Main Go backend (task marketplace)
│   ├── cmd/api/main.go     # Entrypoint
│   └── internal/
│       ├── api/handlers.go
│       ├── middleware/jwt.go
│       ├── models/         # User, Task, Application, Review
│       ├── services/       # Business logic layer
│       └── repositories/   # Data access layer
│
├── store-service/          # Store marketplace (Go) - BLUEPRINT SERVICE
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── api/handlers/
│   │   ├── middleware/
│   │   ├── models/         # StoreItem, Bid, BookingRequest, User (cached)
│   │   ├── repositories/
│   │   └── services/
│   ├── Makefile           # Comprehensive test commands
│   └── migrations/
│
├── chat-websocket-service/ # Real-time messaging (Node.js)
│   ├── src/
│   │   ├── server.js      # Express + Socket.IO server
│   │   ├── handlers/      # WebSocket event handlers
│   │   ├── services/      # Message business logic (including booking)
│   │   └── api/           # HTTP endpoints (booking actions)
│   ├── migrations/        # node-pg-migrate files
│   └── package.json
│
├── frontend/               # Vue 3 SPA
│   ├── src/
│   │   ├── components/    # Reusable Vue components
│   │   │   └── messages/  # BookingMessageBubble, MessageThread, ChatWidget
│   │   ├── views/         # Page-level components
│   │   ├── stores/        # Pinia state management (auth, chat, user, context, messages)
│   │   ├── router/        # Vue Router config
│   │   └── main.ts
│   └── package.json
│
├── engineering/            # Engineering docs & ADRs
│   ├── ❗-current-focus.md  # Current priorities (START HERE)
│   ├── 01-proposed/        # RFCs, ADRs, TODOs, Roadmaps
│   ├── 02-reference/       # Architecture docs
│   └── 03-completed/       # Historical fixes, implementation logs
│
├── docker-compose.yml      # Orchestrates all services
└── scripts/                # Utility scripts
```

---

## 🔄 Key Data Flows

### Task Lifecycle
1. User creates task → Backend (`POST /api/v1/tasks`)
2. Other users apply → Backend (`POST /api/v1/tasks/:id/apply`)
3. Creator accepts application → Task status: `in_progress`, assignee set
4. Chat messages → Chat Service (via WebSocket, contextual by `task_id`)
5. Task completed → Backend (`PATCH /api/v1/tasks/:id/status`)
6. Reviews → Backend (`POST /api/v1/tasks/:id/reviews`)

### Store Item & Booking Lifecycle (Unified Booking Flow)
1. Create item → Store Service (`POST /api/v1/items`)
2. For auctions: Users bid → Store Service (`POST /api/v1/items/:id/bids`)
3. **Booking request** → Store Service (`POST /api/v1/items/:id/booking-request`)
   - Store creates booking record
   - **Async notification** to Chat Service (`POST /internal/booking-created`)
   - Buyer redirected to `/messages`
4. **Chat displays booking** as system message with approve/decline buttons
5. **Seller action** → Chat Service (`POST /booking-action`) → Store Service updates status
6. Chat messages → Chat Service (contextual by `store_item_id`)

### Authentication Flow
1. User registers/logs in → Backend (`POST /api/v1/register`, `/login`)
2. Backend returns JWT with claims: `user_id`, `username`, `email`, `name`
3. Frontend stores JWT and includes in all requests
4. Each service validates JWT independently
5. Store/Chat services automatically upsert user to local cache from JWT claims

---

## 🛠️ Common Development Workflows

### Adding a New API Endpoint

**Backend or Store Service (Go):**
1. Define route in `internal/api/handlers.go` or `internal/api/handlers/`
2. Implement handler function (validate input, call service layer)
3. Add business logic in `internal/services/`
4. Add data access in `internal/repositories/`
5. **Write tests** (use `store-service` as reference - see `internal/api/handlers/*_test.go`)

**Chat Service (Node.js):**
1. Add HTTP route in `src/api/` OR WebSocket event in `src/handlers/socketHandlers.js`
2. Implement business logic in `src/services/`
3. Update database queries if needed
4. Document in service README

### Database Migrations

**Backend/Store (Go with GORM):**
- Auto-migration runs on startup via `db.AutoMigrate()` in `main.go`
- For complex migrations, create manual SQL scripts in `migrations/`

**Chat Service (Node.js):**
```bash
cd chat-websocket-service
npm run migrate:create <migration_name>
# Edit the new file in migrations/
npm run migrate  # Applies pending migrations
```

### Running Tests

**Store Service (Use as Blueprint):**
```bash
cd store-service
make test-coverage          # Runs all tests with coverage report
make test-coverage-check    # Verifies coverage ≥70%
```

**Frontend:**
```bash
cd frontend
npm run test:unit   # Vitest unit tests
npm run test:e2e    # Playwright E2E tests
npm run type-check  # TypeScript type checking (currently 62 errors tracked)
```

**Backend:**
- Currently **0% test coverage** - CRITICAL PRIORITY
- See `engineering/01-proposed/ADR-backend-testing-strategy.md`
- Use `store-service` test patterns as blueprint

---

## ⚠️ Important Notes

### Security
- **CORS**: Currently allows all origins in development. Must restrict to frontend URL in production.
- **JWT_SECRET**: Must be identical across all services. Change default before production.
- **INTERNAL_API_KEY**: Secures inter-service communication (chat ↔ store). Must match in both services.
- **Passwords**: Hashed with bcrypt in backend.
- **Content Filtering**: Chat service automatically removes sensitive data (URLs, emails, phone numbers) from messages.

### Testing Strategy
- **Store Service**: 92%+ coverage - **use as the blueprint for all Go services**
- **Backend**: 0% coverage - **critical priority** (see ADR)
- **Frontend**: Unit tests with Vitest, E2E with Playwright, 62 TypeScript errors tracked
- CI pipeline enforces minimum 80% coverage for store service

### Image Storage (Store Service)
- Currently stores images on local filesystem at `./uploads/store/`
- Images served via `/uploads/*` static route
- For production: migrate to cloud storage (S3, GCS) for scalability

### Message Auto-Deletion (Chat Service)
- Cron job runs daily to check for old conversations
- Messages tied to completed/inactive tasks are scheduled for deletion
- Users notified 30 days before permanent deletion

### Recent Implementations (January 2026)
- **✅ P0 Complete:** Core messaging UX (user enrichment, conversation titles)
- **✅ P2 Complete:** Unified booking & messaging flow (backend + frontend)
- **📋 Tracked:** 62 TypeScript type errors (see `TODO-typescript-errors.md`)

---

## 📚 Documentation References

- **Current Priorities**: `engineering/❗-current-focus.md`
- **MVP Roadmap**: `engineering/01-proposed/ROADMAP-mvp-prioritization.md`
- **Service READMEs**: Each service directory (`backend/`, `store-service/`, `chat-websocket-service/`, `frontend/`)
- **Engineering Docs**: `engineering/` contains ADRs, RFCs, architecture docs, and completed work logs
- **Deployment**: `engineering/01-proposed/DEPLOYMENT-CHECKLIST-booking.md` (for booking feature)

---

## 🚀 Quick Start for Common Tasks

### To understand current priorities:
```bash
cat engineering/❗-current-focus.md
cat engineering/01-proposed/ROADMAP-mvp-prioritization.md
```

### To add backend tests (following store-service blueprint):
1. Review `store-service/internal/api/handlers/store_item_handler_test.go`
2. Copy test patterns (table-driven tests, mocks, test fixtures)
3. See `engineering/01-proposed/ADR-backend-testing-strategy.md`

### To understand TypeScript errors:
```bash
cat engineering/01-proposed/TODO-typescript-errors.md
npm run type-check  # See current errors
```

### To deploy booking feature:
```bash
cat engineering/01-proposed/DEPLOYMENT-CHECKLIST-booking.md
```

---

**Last Updated:** January 4, 2026
**Document Version:** 2.0 (Aligned with GEMINI.md principles)
