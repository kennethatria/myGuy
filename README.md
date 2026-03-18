[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/{owner}/{repo}/badge)](https://scorecard.dev/viewer/?uri=github.com/{owner}/{repo})

# MyGuy - Task Marketplace Platform

MyGuy is a modern, microservices-based task marketplace. It allows users to post tasks they need done, and enables other users to apply, negotiate, and complete those tasks.

The platform is designed with a clean architecture, separating concerns into distinct services for task management, real-time chat, and a store/bidding marketplace.

## Architecture & Tech Stack

The system is composed of several key services that work together:

| Service | Language | Port | Description |
| :--- | :--- | :--- | :--- |
| **Frontend** | TypeScript (Vue.js) | `5173` | The main user interface that communicates with all backend services. |
| **Backend** | Go (Gin) | `8080` | The core API for managing users, tasks, applications, and reviews. |
| **Store Service** | Go (Gin) | `8081` | A marketplace for items with fixed-price and auction-style bidding. |
| **Chat Service** | JavaScript (Node.js) | `8082` | A real-time WebSocket service for all messaging features. |
| **Database** | PostgreSQL | `5432` | Primary data store, with each service connecting to its own database. |

## Quick Start

The entire MyGuy platform can be run easily using Docker.

### Prerequisites
- Docker & Docker Compose
- Git

### Running the Application
1. **Clone the repository:**
   ```sh
   git clone <repository-url>
   cd myguy
   ```
2. **Set up environment variables:**
   - Each service (`backend`, `store-service`, `chat-websocket-service`) has an `.env.example` file. Copy it to `.env` and configure as needed. The default values are generally suitable for local development. A `JWT_SECRET` is required.

3. **Build and run the services:**
   ```sh
   docker-compose up --build
   ```
4. **Access the application:**
   - **Frontend:** [http://localhost:5173](http://localhost:5173)
   - **Backend API:** [http://localhost:8080](http://localhost:8080)
   - **Store Service API:** [http://localhost:8081](http://localhost:8081)
   - **Chat Service:** [http://localhost:8082](http://localhost:8082)

## Documentation

This project contains detailed documentation covering architecture, processes, and service-specific details.

- **[Project Status & Priorities](./engineering/❗-current-focus.md)**: The best place to start. A high-level overview of current engineering priorities and recently completed work.

- **Service Documentation**: Each service has a detailed `README.md` explaining its specific responsibilities, API, and setup instructions.
  - **[Backend README](./backend/README.md)**
  - **[Store Service README](./store-service/README.md)**
  - **[Chat Service README](./chat-websocket-service/README.md)**
  - **[Frontend README](./frontend/README.md)**

- **Engineering & Architecture**: For deeper insights into architectural decisions, processes, and historical fixes.
  - **[Browse Engineering Docs](./engineering/)**
