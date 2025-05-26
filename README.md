# MyGuy - Task Marketplace Platform

A modern task marketplace application where users can create tasks (gigs) for others to complete, negotiate fees, communicate about requirements, and leave reviews.

## Features

### Core Functionality
- **User Authentication**: Secure registration and login system with JWT tokens
- **Task Management**: Create, browse, and manage tasks through their complete lifecycle
- **Advanced Search & Filtering**: Search tasks by title/description with filters for status, price range, and deadline
- **Fee Negotiation**: Applicants can propose their fees when applying for tasks
- **Real-time Communication**: 
  - Task-level messaging between creators and assignees
  - Application-specific messaging for pre-assignment communication
- **Review System**: Both parties can review each other after task completion with 1-5 star ratings
- **User Profiles**: View user profiles with ratings and review history

### Key Features by User Role

#### As a Task Creator
- Post tasks with title, description, deadline, and budget
- View and manage applications with proposed fees
- Communicate with applicants before accepting
- Accept or decline applications
- Message with assignees during task execution
- Mark tasks as complete and review assignees

#### As a Task Applicant/Assignee
- Browse available tasks with advanced filtering
- Search tasks by keywords
- Apply for tasks with custom fee proposals
- Communicate with task creators about requirements
- Update task progress
- Complete tasks and review creators

## Tech Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin (HTTP web framework)
- **Database**: PostgreSQL with GORM ORM
- **Authentication**: JWT tokens
- **Architecture**: Clean architecture with services, repositories, and handlers

### Frontend
- **Framework**: Vue 3 with Composition API
- **State Management**: Pinia
- **Routing**: Vue Router
- **Styling**: Custom CSS with responsive design
- **Build Tool**: Vite
- **Type Safety**: TypeScript

## Project Structure

```
MyGuy/
├── backend/              # Go backend server
│   ├── cmd/             # Application entrypoints
│   ├── internal/        # Private application code
│   │   ├── api/        # HTTP handlers
│   │   ├── models/     # Database models
│   │   ├── services/   # Business logic
│   │   └── repositories/ # Data access layer
│   └── Dockerfile      # Backend container definition
├── myGuy/              # Vue.js frontend
│   ├── src/
│   │   ├── components/ # Reusable components
│   │   ├── views/     # Page components
│   │   ├── stores/    # Pinia stores
│   │   └── router/    # Route definitions
│   └── package.json
└── docker-compose.yml  # Container orchestration
```

## API Endpoints

### Authentication
- `POST /api/v1/register` - Register new user
- `POST /api/v1/login` - Login user

### Tasks
- `GET /api/v1/tasks` - List tasks with pagination, search, and filters
  - Query params: `search`, `status`, `min_fee`, `max_fee`, `deadline_before`, `sort_by`, `sort_order`, `page`, `per_page`
- `POST /api/v1/tasks` - Create a new task
- `GET /api/v1/tasks/:id` - Get task details
- `PUT /api/v1/tasks/:id` - Update a task
- `PATCH /api/v1/tasks/:id/status` - Update task status
- `DELETE /api/v1/tasks/:id` - Delete a task
- `POST /api/v1/tasks/:id/apply` - Apply for a task
- `PATCH /api/v1/tasks/:id/applications/:applicationId` - Accept/decline application

### User Tasks
- `GET /api/v1/user/tasks` - Get tasks created by current user
- `GET /api/v1/user/tasks/assigned` - Get tasks assigned to current user

### Messages
- `POST /api/v1/tasks/:id/messages` - Send task message
- `GET /api/v1/tasks/:id/messages` - Get task messages
- `POST /api/v1/applications/:id/messages` - Send application message
- `GET /api/v1/applications/:id/messages` - Get application messages

### Reviews
- `POST /api/v1/tasks/:id/reviews` - Create a review
- `GET /api/v1/users/:id/reviews` - Get user reviews

### Users & Profile
- `GET /api/v1/users/:id` - Get user details
- `GET /api/v1/profile` - Get current user profile
- `PUT /api/v1/profile` - Update current user profile

## Quick Start

### Using Docker (Recommended)

1. Clone the repository
2. Create a `.env` file in the backend directory:
```env
DB_CONNECTION="host=postgres user=postgres password=your_password dbname=myguy port=5432 sslmode=disable"
JWT_SECRET="your-secret-key"
```

3. Start the application:
```bash
docker-compose up --build
```

The application will be available at:
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

### Local Development

#### Backend Setup

1. Navigate to the backend directory:
```bash
cd backend
```

2. Install dependencies:
```bash
go mod download
```

3. Set up PostgreSQL database and update `.env`

4. Run the backend:
```bash
go run cmd/api/main.go
```

#### Frontend Setup

1. Navigate to the frontend directory:
```bash
cd myGuy
```

2. Install dependencies:
```bash
npm install
```

3. Create `.env` file:
```env
VITE_API_URL=http://localhost:8080
```

4. Run the development server:
```bash
npm run dev
```

## Database Schema

### Core Tables
- `users` - User accounts with authentication and profile data
- `tasks` - Task/gig listings with status tracking
- `applications` - Task applications with proposed fees
- `messages` - Communication between users (task and application scoped)
- `reviews` - User reviews and ratings

## Testing

### Backend Tests
```bash
cd backend
go test ./...
```

### Frontend Tests
```bash
cd myGuy
npm run test:unit    # Unit tests
npm run test:e2e     # End-to-end tests
```

## Docker Commands

Build and start all services:
```bash
docker-compose up --build
```

Start in detached mode:
```bash
docker-compose up -d
```

Stop all services:
```bash
docker-compose down
```

View logs:
```bash
docker-compose logs -f
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a new Pull Request

## License

This project is licensed under the MIT License.