# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

MyGuy is a microservices-based task marketplace platform where users can post tasks, apply for tasks, communicate in real-time, and buy/sell items. The architecture separates concerns into distinct services: task management (Go), real-time chat (Node.js), and a store/bidding marketplace (Go), each with its own PostgreSQL database.

## Architecture

### Service Topology

| Service | Language/Framework | Port | Database | Purpose |
|---------|-------------------|------|----------|---------|
| **Backend** | Go (Gin) | 8080 | `my_guy` | Core task marketplace API (users, tasks, applications, reviews) |
| **Store Service** | Go (Gin) | 8081 | `my_guy_store` | Marketplace for items with fixed-price and auction bidding |
| **Chat Service** | Node.js (Express + Socket.IO) | 8082 | `my_guy_chat` | Real-time WebSocket messaging service |
| **Frontend** | Vue 3 + TypeScript (Vite) | 5173 | - | Single-page application |
| **Database** | PostgreSQL 15 | 5432 (exposed as 5433) | - | Shared database server with multiple databases |

### Critical Architecture Principles

1. **Database Separation**: Each service has its own database. Services do NOT query each other's databases directly.

2. **Service Independence**: The chat service communicates only through IDs. The frontend is responsible for fetching associated data (like user or task details) from the appropriate services.

3. **Unified Message Table**: The chat service uses a single `messages` table for all message types (tasks, store items), distinguished by foreign key columns (`task_id`, `store_item_id`).

4. **JWT Authentication**: All services share the same `JWT_SECRET`. The store and chat services perform automatic user synchronization via JWT middleware to maintain local user caches.

5. **Content Filtering**: The chat service automatically strips URLs, emails, phone numbers, and social media handles from messages to protect user privacy.

## Development Commands

### Running the Full Platform

```bash
# Start all services with Docker Compose (recommended)
docker-compose up --build

# View logs for a specific service
docker-compose logs -f [api|store-service|chat-websocket-service]

# Stop all services
docker-compose down
```

### Backend (Go - Main API)

```bash
cd backend

# Run the service locally
go run cmd/api/main.go

# Build
go build -o backend cmd/api/main.go

# Note: Backend currently has 0% test coverage (priority item)
```

### Store Service (Go)

```bash
cd store-service

# Run the service locally
go run cmd/api/main.go

# Build
make build

# Testing (92%+ coverage - use as blueprint for backend tests)
make test                    # Run all tests
make test-unit              # Unit tests only
make test-integration       # Integration tests only
make test-coverage          # Generate coverage report with HTML
make test-coverage-check    # Verify coverage ≥70%
make test-watch             # Watch mode (requires entr)

# Other commands
make lint                   # Lint code
make fmt                    # Format code
make help                   # Show all available commands
```

### Chat Service (Node.js)

```bash
cd chat-websocket-service

# Install dependencies
npm install

# Run in development mode
npm run dev

# Run in production mode
npm start

# Database migrations
npm run migrate             # Run pending migrations
npm run migrate:create <name>  # Create new migration

# Testing
npm test                    # Run tests
npm run lint               # Lint code
```

### Frontend (Vue 3 + TypeScript)

```bash
cd frontend

# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Testing
npm run test:unit          # Unit tests (Vitest)
npm run test:e2e           # E2E tests (Playwright)

# Code quality
npm run lint               # ESLint with auto-fix
npm run format             # Prettier formatting
npm run type-check         # TypeScript type checking
```

## Environment Configuration

Each service requires a `.env` file. Copy from `.env.example` (if available) or create:

### Backend
```env
PORT=8080
JWT_SECRET=your-secret-key-here
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
JWT_SECRET=your-secret-key-here
DB_CONNECTION=host=localhost user=postgres password=mysecretpassword dbname=my_guy_store port=5432 sslmode=disable
```

### Chat Service
```env
PORT=8082
NODE_ENV=development
JWT_SECRET=your-secret-key-here
DB_CONNECTION=postgresql://postgres:mysecretpassword@localhost:5432/my_guy_chat
DATABASE_URL=postgresql://postgres:mysecretpassword@localhost:5432/my_guy_chat
CLIENT_URL=http://localhost:5173
MAIN_API_URL=http://localhost:8080/api/v1
STORE_API_URL=http://localhost:8081/api/v1
```

### Frontend
```env
VITE_API_URL=http://localhost:8080/api/v1
VITE_STORE_API_URL=http://localhost:8081/api/v1
VITE_CHAT_API_URL=http://localhost:8082/api/v1
VITE_CHAT_WS_URL=http://localhost:8082
```

## Project Structure

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
├── store-service/          # Store marketplace microservice (Go)
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
├── chat-websocket-service/ # Real-time messaging service (Node.js)
│   ├── src/
│   │   ├── server.js      # Express + Socket.IO server
│   │   ├── handlers/      # WebSocket event handlers
│   │   ├── services/      # Message business logic
│   │   └── scripts/       # Migration scripts
│   ├── migrations/        # node-pg-migrate files
│   └── package.json
│
├── frontend/               # Vue 3 SPA
│   ├── src/
│   │   ├── components/    # Reusable Vue components
│   │   ├── views/         # Page-level components
│   │   ├── stores/        # Pinia state management
│   │   ├── router/        # Vue Router config
│   │   └── main.ts
│   └── package.json
│
├── engineering/            # Engineering docs & ADRs
│   ├── ❗-current-focus.md  # Current priorities (START HERE)
│   ├── 01-proposed/        # Proposed changes
│   ├── 02-reference/       # Architecture docs
│   └── 03-completed/       # Historical fixes
│
├── docker-compose.yml      # Orchestrates all services
└── scripts/                # Utility scripts
```

## Key Data Flows

### Task Lifecycle
1. User creates task → Backend (`POST /api/v1/tasks`)
2. Other users apply → Backend (`POST /api/v1/tasks/:id/apply`)
3. Creator accepts application → Task status: `in_progress`, assignee set
4. Chat messages → Chat Service (via WebSocket, contextual by `task_id`)
5. Task completed → Backend (`PATCH /api/v1/tasks/:id/status`)
6. Reviews → Backend (`POST /api/v1/tasks/:id/reviews`)

### Store Item Lifecycle
1. Create item → Store Service (`POST /api/v1/items`)
2. For auctions: Users bid → Store Service (`POST /api/v1/items/:id/bids`)
3. For fixed-price: Users purchase → Store Service (`POST /api/v1/items/:id/purchase`)
4. Booking flow: Request → Approve/Reject → Chat unlocked
5. Chat messages → Chat Service (contextual by `store_item_id`)

### Authentication Flow
1. User registers/logs in → Backend (`POST /api/v1/register`, `/login`)
2. Backend returns JWT with claims: `user_id`, `username`, `email`, `name`
3. Frontend stores JWT and includes in all requests
4. Each service validates JWT independently
5. Store/Chat services automatically upsert user to local cache from JWT claims

## Common Workflows

### Adding a New API Endpoint

**Backend or Store Service:**
1. Define route in `internal/api/handlers.go` or `internal/api/handlers/`
2. Implement handler function (validate input, call service layer)
3. Add business logic in `internal/services/`
4. Add data access in `internal/repositories/`
5. Write tests (use `store-service` as reference)

**Chat Service:**
1. Add WebSocket event handler in `src/handlers/socketHandlers.js`
2. Implement business logic in `src/services/messageService.js`
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

**Store Service (has comprehensive tests):**
```bash
cd store-service
make test-coverage  # Runs all tests with coverage report
```

**Frontend:**
```bash
cd frontend
npm run test:unit   # Vitest unit tests
npm run test:e2e    # Playwright E2E tests
```

**Backend:**
- Currently has 0% test coverage
- Top priority: Implement testing following `store-service` patterns
- See `engineering/01-proposed/ADR-backend-testing-strategy.md`

## Important Notes

### Security
- **CORS**: Currently allows all origins in development. Must restrict to frontend URL in production.
- **JWT_SECRET**: Must be identical across all services. Change default before production.
- **Passwords**: Hashed with bcrypt in backend.
- **Content Filtering**: Chat service automatically removes sensitive data (URLs, emails, phone numbers) from messages.

### Testing Strategy
- **Store Service**: 92%+ coverage - use as the blueprint
- **Backend**: 0% coverage - critical priority
- **Frontend**: Unit tests with Vitest, E2E with Playwright
- CI pipeline enforces minimum 80% coverage for store service

### Image Storage (Store Service)
- Currently stores images on local filesystem at `./uploads/store/`
- Images served via `/uploads/*` static route
- For production: migrate to cloud storage (S3, GCS) for scalability

### Message Auto-Deletion (Chat Service)
- Cron job runs daily to check for old conversations
- Messages tied to completed/inactive tasks are scheduled for deletion
- Users notified 30 days before permanent deletion

## Current Engineering Focus

See `engineering/❗-current-focus.md` for the latest priorities.

**Top Priority (Q1 2026):** Implement comprehensive test coverage for the main backend service using `store-service` as a blueprint.

**Next Up:**
- Browser push notifications for messages
- Dedicated authentication microservice

## Documentation

- **Service READMEs**: Each service directory has detailed documentation
- **Engineering Docs**: `engineering/` contains ADRs, architecture docs, and completed work logs
- **GitHub Copilot Instructions**: `.github/copilot-instructions.md` contains legacy development guidelines
