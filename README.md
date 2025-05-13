# MyGuy - Task Management Application

A modern task management application where users can create tasks for others to complete.

## Project Structure

```
MyGuy/
├── backend/           # Go backend server
│   ├── cmd/          # Application entrypoints
│   ├── internal/     # Private application code
│   └── Dockerfile    # Backend container definition
├── myGuy/            # Vue.js frontend
└── docker-compose.yml # Container orchestration
```

## Prerequisites

- Docker and Docker Compose
- Node.js 18+ (for local frontend development)
- Go 1.21+ (for local backend development)
- PostgreSQL (handled by Docker)

## Quick Start

1. Clone the repository
2. Create a `.env` file in the backend directory:

```env
DB_CONNECTION="host=postgres user=postgres password=your_password dbname=myguy port=5432 sslmode=disable"
JWT_SECRET="your-secret-key"
```

3. Start the application using Docker Compose:

```powershell
docker-compose up --build
```

The application will be available at:
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

## Development Setup

### Backend (Go)

1. Navigate to the backend directory:
```powershell
cd backend
```

2. Install dependencies:
```powershell
go mod download
```

3. Run the backend:
```powershell
go run cmd/api/main.go
```

### Frontend (Vue.js)

1. Navigate to the frontend directory:
```powershell
cd myGuy
```

2. Install dependencies:
```powershell
npm install
```

3. Run the development server:
```powershell
npm run dev
```

## Available API Endpoints

### Tasks
- `GET /api/v1/tasks` - List all tasks
- `POST /api/v1/tasks` - Create a new task
- `GET /api/v1/tasks/:id` - Get task details
- `PUT /api/v1/tasks/:id` - Update a task
- `DELETE /api/v1/tasks/:id` - Delete a task
- `POST /api/v1/tasks/:id/apply` - Apply for a task

### Authentication
- `POST /api/v1/register` - Register new user
- `POST /api/v1/login` - Login user

### Messages
- `POST /api/v1/tasks/:id/messages` - Send a message
- `GET /api/v1/tasks/:id/messages` - Get task messages

### Reviews
- `POST /api/v1/tasks/:id/reviews` - Create a review
- `GET /api/v1/users/:id/reviews` - Get user reviews

## Testing

### Backend Tests

Run the Go tests:
```powershell
cd backend
go test ./...
```

### Frontend Tests

Run the Vue.js tests:
```powershell
cd myGuy
npm run test:unit    # Unit tests
npm run test:e2e     # End-to-end tests
```

## Docker Commands

Build and start all services:
```powershell
docker-compose up --build
```

Start in detached mode:
```powershell
docker-compose up -d
```

Stop all services:
```powershell
docker-compose down
```

View logs:
```powershell
docker-compose logs -f
```

## Environment Variables

### Backend (.env)
- `DB_CONNECTION` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for JWT tokens

### Frontend (.env)
- `VITE_API_URL` - Backend API URL (default: http://localhost:8080)

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License.
