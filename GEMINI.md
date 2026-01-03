# MyGuy Project Context (Gemini CLI)

## Architecture Overview
MyGuy is a microservices platform for a task marketplace.
- **Service Independence**: Services communicate only via IDs. No cross-service DB queries.
- **Shared Auth**: All services use the same `JWT_SECRET`.
- **User Privacy**: Chat service automatically filters PII (emails, phone, URLs).

## Service Map
| Service | Language/Framework | Port | DB Name |
| :--- | :--- | :--- | :--- |
| **Backend** | Go (Gin) | 8080 | `my_guy` |
| **Store** | Go (Gin) | 8081 | `my_guy_store` |
| **Chat** | Node.js (Express) | 8082 | `my_guy_chat` |
| **Frontend** | Vue 3 (TS) | 5173 | N/A |

## Project-Specific Rules for Gemini
- **Strict Separation**: If I ask for a feature involving two services, plan the API changes in each service separately.
- **Testing Standard**: Use `store-service` as the "Gold Standard" for testing logic. When writing Go code for the `backend` service, replicate the pattern found in `store-service/Makefile` and `internal/services/`.
- **Privacy Filtering**: When working on the Chat Service, ensure any new message handling logic includes the content filtering middleware.

## Dev Commands Reference
- **All Services**: `docker-compose up --build`
- **Store Tests**: `cd store-service && make test-coverage`
- **Chat Migrations**: `cd chat-websocket-service && npm run migrate`

## Implementation Checklist (Standard Workflow)
1. Verify `JWT_SECRET` alignment across `.env` files.
2. If adding an endpoint, check the corresponding local user cache logic.
3. For Chat updates, ensure the `messages` table schema supports the new context.