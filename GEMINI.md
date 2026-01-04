# Project: MyGuy Platform Context

## 🏗 Architecture & Service Topology
MyGuy is a microservices-based task marketplace. Refer to the specific service READMEs for implementation details.

- **Backend**: Go (Gin) on Port 8080. Core marketplace API (Users, Tasks).
- **Store Service**: Go (Gin) on Port 8081. Fixed-price and auction bidding.
- **Chat Service**: Node.js (Express + Socket.IO) on Port 8082. Real-time messaging.
- **Frontend**: Vue 3 + TypeScript on Port 5173.

## 📜 Critical Engineering Principles
- **Database Isolation**: Services never query each other's databases directly.
- **Shared Auth**: All services use a unified `JWT_SECRET` for independent validation.
- **User Privacy**: The Chat Service must automatically filter PII (emails, phone numbers, URLs).
- **Service Blueprint**: Use the `store-service` (92%+ test coverage) as the architectural blueprint for any new Go development or testing strategies.

## 📂 Documentation & Status Tracking
Always reference the `engineering/` directory for project health and historical context:
- **Priorities**: Check `engineering/❗-current-focus.md` for the current Q1 2026 testing push.
- **Decisions**: Refer to `engineering/01-proposed/` for ADRs (Architecture Decision Records).
- **History**: See `engineering/03-completed/` for previous fixes and investigations.

## 🛠 Common Commands
- **Launch Platform**: `docker-compose up --build`
- **Store Tests**: `cd store-service && make test-coverage`
- **Chat Migrations**: `cd chat-websocket-service && npm run migrate`