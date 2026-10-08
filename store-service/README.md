# Store Service - MyGuy Marketplace

This microservice provides the marketplace for the MyGuy platform: listings and "wanted" requests as short sticky notes, bookings agreed in chat, and secure image handling. Listings carry no price; it goes in the note or is agreed in chat.

The service is built with Go, using the Gin framework for its API, and GORM for database interactions with PostgreSQL.

**Test Coverage:** 92%+ across 110+ test scenarios.

## Table of Contents

1.  [Features](#1-features)
2.  [Architecture](#2-architecture)
3.  [Getting Started (Development)](#3-getting-started-development)
4.  [Testing Guide](#4-testing-guide)
5.  [Key Workflows](#5-key-workflows)
    - [Booking Flow](#51-booking-flow)
    - [Image Storage](#52-image-storage)
6.  [API Documentation](#6-api-documentation)
7.  [Data Models](#7-data-models)
8.  [Deployment](#8-deployment)

---

## 1. Features

-   **Item Management**: Create, update, and delete item listings with rich details.
-   **Image Handling**: Allows multiple image uploads per item.
-   **Booking System**: Users can request to "book" an item, which owners can approve or reject.
-   **User Synchronization**: Automatically creates and updates a local cache of users from incoming JWTs to ensure data integrity and prevent "Unknown User" errors.
-   **Advanced Search**: Provides full-text search with filtering, sorting, and pagination.

---

## 2. Architecture

-   **Language**: Go 1.21+
-   **Framework**: Gin
-   **Database**: PostgreSQL with GORM
-   **Authentication**: JWT middleware with automatic user synchronization.

### Project Structure
```
store-service/
├── cmd/api/main.go              # Application entrypoint
├── internal/
│   ├── api/handlers/          # HTTP handlers
│   ├── middleware/            # JWT authentication middleware
│   ├── models/                # GORM data models
│   ├── repositories/            # Data access layer
│   └── services/                # Business logic
├── migrations/
├── Dockerfile
├── go.mod
└── README.md
```

### JWT Authentication & User Synchronization
The service expects JWTs containing `user_id`, `username`, `email`, and `name`. The JWT middleware automatically performs an "upsert" on a local `users` table, ensuring that user data is available for foreign key relationships (e.g., in booking requests) without direct calls to the main backend.

---

## 3. Getting Started (Development)

### Prerequisites
-   Go 1.21+
-   PostgreSQL 12+
-   Docker (optional, for containerized environment)

### Local Setup
1.  **Clone the repository.**
2.  **Configure Environment:** Create a `.env` file from `.env.example` and set the `DB_CONNECTION` and `JWT_SECRET`.
3.  **Install Dependencies:**
    ```bash
    go mod download
    ```
4.  **Run the Service:**
    ```bash
    go run cmd/api/main.go
    ```
The service will be available at `http://localhost:8081`.

### Docker Development
The project includes a `docker-compose.yml` file for easy setup.
```bash
# Build and run all services
docker-compose up --build store-service
```

---

## 4. Testing Guide

The service has a test coverage of over 92%, validated via CI.

### Running Tests
A `Makefile` provides convenient commands for testing:

| Command                  | Description                                            |
| ------------------------ | ------------------------------------------------------ |
| `make test`              | Run all unit and integration tests.                    |
| `make test-unit`         | Run only the unit tests.                               |
| `make test-integration`  | Run only the integration tests.                        |
| `make test-coverage`     | Run all tests and generate an HTML coverage report.    |
| `make test-coverage-check`| Run tests and fail if coverage is below 90%.         |
| `make test-watch`        | Run tests in watch mode (requires `entr`).             |

### Test Strategy
-   **Unit Tests**: Handlers, services, and repositories are tested in isolation using mocks.
-   **Integration Tests**: End-to-end workflows are tested against a real (in-memory SQLite) database to verify API behavior and data integrity.
-   **Coverage**: CI pipeline enforces a minimum of 80% test coverage.
-   **Key Areas Covered**:
    -   All API endpoints, including success and error cases.
    -   JWT validation and automatic user synchronization.
    -   Booking request workflows (creation, approval, rejection, edge cases).
    -   Database constraints and repository logic.

---

## 5. Key Workflows

### 5.1. Booking Flow
This flow allows a potential buyer to formally express interest in an item, which the seller can then approve or reject.

1.  **Request**: A buyer sends a `POST` request to `/api/v1/items/:id/booking-request`.
2.  **View**: The seller can view all booking requests for their item via `GET /api/v1/items/:id/booking-requests`.
3.  **Action**: The seller can approve or reject a request using `POST /booking-requests/:id/approve` or `POST /booking-requests/:id/reject`.
4.  **Messaging**: Upon approval, the communication channel between buyer and seller may be expanded (this logic is handled by the `chat-websocket-service`).

The API handles all necessary authorization, ensuring only item owners can manage requests.

### 5.2. Image Storage
The service currently stores uploaded images on the **local server filesystem**.

-   **Path**: `./uploads/store/{user_id}/{timestamp}_{index}.{extension}`
-   **Database**: Only the URL paths are stored in the `store_items` table.
-   **Serving**: Images are served statically via the `/uploads/*` route.

**Production Recommendation**: For production, it is strongly recommended to use a cloud storage solution like **AWS S3** or Google Cloud Storage for scalability, reliability, and to enable CDN integration.

---

## 6. API Documentation
*Full details are available in the API handlers and tests.*

### Common Endpoints
-   `GET /items`: Browse items with filtering, sorting, and pagination.
-   `POST /items`: Post a listing: `title` (≤ 5 words) and `description` (≤ 20 words), no contact details, optional `images`. No price fields. It stays up 24 hours unless someone asks to book.
-   `POST /items/:id/repost`: Put the seller's expired listing back up for 24 hours.
-   `GET /requests`, `GET /requests/:id`, `GET /requests/:id/listings`: "Wanted" notes and the listings made for one.
-   `POST /requests`: Post a request (same note rules as a listing). `POST /requests/:id/repost` and `DELETE /requests/:id` for the requester. `GET /user/requests`: your own.
-   A listing created with `request_id` answers that request: the requester gets a message through chat's `/internal/store-message` (needs `INTERNAL_API_KEY` and `CHAT_API_URL`).
-   `GET /items/:id`: Get details for a single item.
-   `PUT /items/:id`: Change a live listing's headline and note (owner only). `PUT /requests/:id` does the same for a request.
-   `POST /items/:id/booking-request`: Request to book an item.
-   `POST /booking-requests/:id/approve|reject|confirm-received|confirm-delivery|rate-seller|rate-buyer`: Booking workflow steps.
-   `GET /users/:id/ratings`: Store ratings a user has received (as seller or buyer). Profiles combine these with task reviews into one rating.

User records embedded in responses (e.g. an item's seller) never include the email address.

---

## 7. Data Models
-   **StoreItem**: Represents an item for sale: title, description, seller, status, images, and its 24-hour deadline. Price and auction columns remain only on older rows (`is_auction` marks old auctions, which can't be bid on or booked).
-   **BookingRequest**: Represents a user's request to book an item.
-   **User**: A local cache of user information, synchronized from JWTs.

---

## 8. Deployment
-   The service is containerized using Docker.
-   Ensure all environment variables in `.env` are configured for production.
-   **Critical**: For production, mount the `./uploads` directory to a persistent volume to prevent data loss on container restarts.
-   Add database indexes on frequently queried fields (`seller_id`, `status`, `category`) for performance.
