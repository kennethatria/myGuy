# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

MyGuy is a microservices task marketplace: users post tasks, apply, chat in real time, and buy/sell items. Each service owns its own PostgreSQL database.

| Service | Language/Framework | Port | Database | Purpose |
|---------|-------------------|------|----------|---------|
| **Backend** | Go (Gin) | 8080 | `my_guy` | Core task marketplace API (users, tasks, applications, reviews) |
| **Store Service** | Go (Gin) | 8081 | `my_guy_store` | Marketplace items, fixed-price and auction bidding |
| **Chat Service** | Node.js (Express + Socket.IO) | 8082 | `my_guy_chat` | Real-time WebSocket messaging |
| **Frontend** | Vue 3 + TypeScript (Vite) | 5173 | - | Single-page application |
| **Database** | PostgreSQL 15 | 5432 (exposed 5433) | - | Shared server, multiple databases |
| **Redis** | Redis 7 | 6379 | - | Optional Socket.IO adapter for multi-instance chat |

## Critical Architecture Principles

1. **Database Isolation** — Services never query each other's databases directly. The frontend fetches associated data (user/task details) from the owning service.
2. **Shared Authentication** — All services validate against one `JWT_SECRET`. Store and chat perform automatic user sync via JWT middleware into local caches.
3. **User Privacy & Content Filtering** — The Chat Service must strip URLs, emails, phone numbers, and social handles from messages.
4. **Service Blueprint Pattern** — Use `store-service` (92%+ coverage) as the architectural blueprint for new Go development: handlers → services → repositories, with tests.
5. **Unified Message Table** — Chat uses a single `messages` table for all message types, distinguished by `task_id` / `store_item_id` / `application_id`.

## Engineering Docs

Check `engineering/❗-current-focus.md` first for current priorities, then `engineering/01-proposed/` (ADRs/RFCs/TODOs), `engineering/02-reference/` (architecture), `engineering/03-completed/` (fix logs).

## Common Development Commands

```bash
docker-compose up --build          # run everything
docker-compose logs -f <service>   # api | store-service | chat-websocket-service | frontend
```

- **Backend**: `cd backend && go run cmd/api/main.go`. No test suite yet — critical priority, use `store-service` as the pattern reference.
- **Store Service** (blueprint): `cd store-service && make test-coverage-check` (enforces ≥70%). `make help` for all targets.
- **Chat Service**: `cd chat-websocket-service && npm run dev`. Migrations: `npm run migrate:create <name>` then `npm run migrate`.
- **Frontend**: `cd frontend && npm run dev`. `npm run test:unit`, `npm run test:e2e`, `npm run type-check`.

## Environment

Each service has a `.env.example` to copy. The one constraint that isn't visible from any single file:

- **`JWT_SECRET`** must be identical across backend, store-service, and chat-websocket-service, or cross-service auth breaks.
- **`INTERNAL_API_KEY`** must match between store-service and chat-websocket-service — it's what lets store-service notify chat of new bookings (`POST /internal/booking-created`). Without it, booking requests never appear in Messages.

## Important Notes

- **Security**: CORS currently allows all origins (dev only — restrict before production). Passwords are bcrypt-hashed in the backend.
- **Image Storage**: Store service images live on local filesystem at `./uploads/store/`; migrate to cloud storage (S3/GCS) before scaling.
- **Message Auto-Deletion**: A daily cron flags messages on completed/inactive tasks for deletion; users are notified 30 days ahead.
