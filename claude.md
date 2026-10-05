# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

MyGuy is a microservices task marketplace: users post tasks, apply, chat in real time, and buy/sell items. Each service owns its own PostgreSQL database.

| Service | Language/Framework | Port | Database | Purpose |
|---------|-------------------|------|----------|---------|
| **Backend** | Go (Gin) | 8080 | `my_guy` | Core API: passwordless sign-in, users, tasks, applications, reviews |
| **Store Service** | Go (Gin) | 8081 | `my_guy_store` | Marketplace items, fixed-price and auction bidding |
| **Chat Service** | Node.js (Express + Socket.IO) | 8082 | `my_guy_chat` | Real-time WebSocket messaging |
| **Frontend** | Vue 3 + TypeScript (Vite) | 5173 | - | Single-page application |
| **Database** | PostgreSQL 15 | 5432 (exposed 5433) | - | Shared server, multiple databases |
| **Redis** | Redis 7 | 6379 | - | Optional Socket.IO adapter for multi-instance chat |

## Critical Architecture Principles

1. **Database Isolation** — Services never query each other's databases directly. The frontend fetches associated data (user/task details) from the owning service.
2. **Shared Authentication** — All services validate session JWTs with one `JWT_SECRET`. Store and chat perform automatic user sync via JWT middleware into local caches.
3. **Passwordless Sign-in** — Login and sign-up are one email-code flow owned by the backend (`internal/services/auth_service.go`, endpoints `/api/v1/auth/request-code|verify-code|complete-signup`). Only an HMAC of each code is stored. Signup tokens are signed with a key *derived* from `JWT_SECRET` so no service can mistake one for a session — keep it that way.
4. **User Privacy & Content Filtering** — The Chat Service must strip URLs, emails, phone numbers, and social handles from messages.
5. **Service Blueprint Pattern** — New Go code follows handlers → services → repositories, with tests. `store-service` is the reference; the backend now follows it too (repository and auth tests run on in-memory SQLite).
6. **Unified Message Table** — Chat uses a single `messages` table for all message types, distinguished by `task_id` / `store_item_id` / `application_id`.
7. **Conversation Identity** — A conversation is *(context type, context id, other participant)*: frontend `conversationKey()` in `stores/chat.ts`, backend `conversationFilter()` in `messageService.js`. Never key by a bare id (task/application/item ids overlap; items and tasks have many counterparts). Deliver message events only to `user:<id>` rooms, never shared conversation rooms.
8. **Events Are Messages** — Task events (new application, accepted, declined, cancelled) are posted by the backend (`internal/chatnotify`, best effort) to chat's `POST /internal/task-message` and stored as `system_alert` (not editable/deletable). The gig page has one poster↔person conversation per pair; there are no per-application threads. Application chats that do exist are authorized via `GET /applications/:id/participants`.

## Engineering Docs

`engineering/` is git-ignored (local working notes only). If present, check `engineering/❗-current-focus.md` first, then `01-proposed/` (ADRs/RFCs/TODOs), `02-reference/` (architecture), `03-completed/` (fix logs). `README.md` is the maintained reference for architecture, security, and deployment.

## Common Development Commands

```bash
podman compose up -d                 # backend services (pre-built images; no frontend service)
podman compose logs -f <service>     # api | store-service | chat-websocket-service
podman compose logs api | grep "login code"   # sign-in codes when SMTP_HOST is unset
```

- **Backend**: `cd backend && go run cmd/api/main.go`. Tests: `go test ./...`. CI enforces ≥70% coverage on `./internal/...` and the backend sits just above it — add tests with new code.
- **Store Service**: `cd store-service && make test-coverage-check` (enforces ≥70%). `make help` for all targets.
- **Chat Service**: `cd chat-websocket-service && npm run dev`. Migrations: `npm run migrate:create <name>` then `npm run migrate`. `npm test` rewrites the committed `coverage/` report — restore it (`git checkout -- coverage`) rather than committing the churn.
- **Frontend**: `cd frontend && npm run dev`. `npm run test:unit`, `npm run test:e2e`, `npm run type-check`. `npm run lint` runs `eslint --fix` and may edit files.
- **Go modules**: `vendor/` is git-ignored; CI runs with `GOWORK=off` and downloads modules. Locally `go.work` hides a stale `vendor/` — to reproduce CI, use `GOWORK=off GOFLAGS=-mod=mod go test ./internal/...`.

## Environment

Each service has a `.env.example` to copy. Constraints not visible from any single file:

- **`JWT_SECRET`** must be identical across backend, store-service, and chat-websocket-service, or cross-service auth breaks.
- **`INTERNAL_API_KEY`** must match between store-service and chat-websocket-service — it's what lets store-service notify chat of new bookings (`POST /internal/booking-created`). Without it, booking requests never appear in Messages.
- **`SMTP_*`** (backend) sends sign-in codes. If `SMTP_HOST` is unset the backend *logs* codes instead — fine locally, but in production nobody could log in.
- **`IMAGE_TAG`** selects the app image tag in `docker-compose.yml` (default `latest`); the deploy sets it to the tag whose Cosign signature it verified.

## Deployment & Infrastructure

- **Pipeline** (`.github/workflows/ci.cd.yml`): push to `main` → tests → build, sign, push images → automatic deploy (`Run ansible`, `scope: app` = `deploy.yml` + frontend upload). Server provisioning changes (`users.yml`, `site.yml`, `security.yml`, `observability.yml`, `monitoring.yml`) need a manual `Run ansible` with `scope: full`. Terraform (`infra/`) is always manual.
- **NodeBalancer ↔ nginx are coupled**: port 443 uses PROXY protocol v2 (`infra/main.tf`), so nginx must `listen 443 ssl proxy_protocol`; port 80 health-checks `/healthcheck/` expecting body `healthcheck` — never redirect it. Real client IPs come from `set_real_ip_from 192.168.255.0/24` (`templates/nginx-common.conf.j2`), which fail2ban depends on.
- **Compose files**: `docker-compose.yml` is production; `docker-compose.override.yml` holds local-only services (Zipkin) and is auto-loaded locally. Production commands must pass `-f docker-compose.yml`. The server runs podman-compose 1.0.6, which ignores compose `profiles`.
- **Ansible**: `users.yml` picks `root` (fresh server) or `myguy` (bootstrapped) via `tasks/bootstrap_login_user.yml` — inventory `ansible_user` overrides `remote_user`. Pass values containing spaces as JSON extra-vars (`-e "k=v"` splits on spaces). Monitoring containers are Podman Quadlet units defined by the `monitoring_containers` list in `monitoring.yml`.
- **Sudo**: Ansible `become` runs `sudo /bin/sh -c ...`; never add shell denylists (`!SHELLS`) to `myguy`'s sudoers — it locks out all provisioning.

## Important Notes

- **Security**: CORS currently allows all origins (restrict before scaling). There is no password login; the `users.password` column is kept but unused.
- **Image Storage**: Store service images live on local filesystem at `./uploads/store/`; migrate to cloud storage (S3/GCS) before scaling.
- **Message Auto-Deletion**: A daily cron flags messages on completed/inactive tasks for deletion; users are notified 30 days ahead.
