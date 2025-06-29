# Main Backend Current State

## Overview
The main backend is a Go-based REST API for the MyGuy task marketplace platform. After cleanup, it focuses purely on core business logic with chat functionality separated to a microservice.

## Architecture

### Tech Stack
- **Language**: Go 1.21+
- **Framework**: Gin (HTTP web framework)
- **Database**: PostgreSQL with GORM ORM
- **Authentication**: JWT tokens
- **Architecture**: Clean architecture with handlers, services, and repositories

### Project Structure
```
backend/
├── cmd/api/main.go              # Application entrypoint
├── internal/
│   ├── api/handlers.go          # HTTP handlers
│   ├── middleware/jwt.go        # JWT authentication middleware
│   ├── models/
│   │   ├── user.go             # User model
│   │   └── task.go             # Task, Application, Review models
│   ├── services/
│   │   ├── user_service.go     # User business logic
│   │   ├── task_service.go     # Task business logic
│   │   └── review_service.go   # Review business logic
│   └── repositories/
│       ├── user_repository.go   # User data access
│       ├── task_repository.go   # Task data access
│       ├── application_repository.go # Application data access
│       └── review_repository.go # Review data access
├── go.mod
└── go.sum
```

## API Endpoints

### Public Endpoints
- `GET /api/v1/time` - Current server time
- `GET /api/v1/server-time` - Server time with deadline examples
- `POST /api/v1/register` - User registration
- `POST /api/v1/login` - User authentication

### Protected Endpoints (Require JWT)

#### Task Management
- `POST /api/v1/tasks` - Create new task
- `GET /api/v1/tasks` - List tasks with filtering/search/pagination
- `GET /api/v1/tasks/:id` - Get specific task
- `PUT /api/v1/tasks/:id` - Update task (creator only)
- `PATCH /api/v1/tasks/:id/status` - Update task status
- `DELETE /api/v1/tasks/:id` - Delete task (creator only)

#### Application Management
- `POST /api/v1/tasks/:id/apply` - Apply for task
- `GET /api/v1/tasks/:id/applications` - Get task applications
- `PATCH /api/v1/tasks/:id/applications/:applicationId` - Accept/decline application

#### User Task Views
- `GET /api/v1/user/tasks` - Get user's created tasks
- `GET /api/v1/user/tasks/assigned` - Get user's assigned tasks

#### Review System
- `POST /api/v1/tasks/:id/reviews` - Create review
- `GET /api/v1/users/:id/reviews` - Get user reviews

#### User Management
- `GET /api/v1/users/:id` - Get user details
- `GET /api/v1/profile` - Get current user profile
- `PUT /api/v1/profile` - Update user profile

## Data Models

### User
```go
type User struct {
    ID            uint      `json:"id"`
    Username      string    `json:"username"`
    Email         string    `json:"email"`
    Password      string    `json:"-"`        // Hidden from JSON
    FullName      string    `json:"full_name"`
    PhoneNumber   string    `json:"phone_number"`
    Bio           string    `json:"bio"`
    AverageRating float64   `json:"average_rating"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

### Task
```go
type Task struct {
    ID          uint       `json:"id"`
    Title       string     `json:"title"`
    Description string     `json:"description"`
    Status      string     `json:"status"`      // open, in_progress, completed, cancelled
    CreatedBy   uint       `json:"created_by"`
    AssignedTo  *uint      `json:"assigned_to"`
    Fee         float64    `json:"fee"`
    Deadline    time.Time  `json:"deadline"`
    CompletedAt *time.Time `json:"completed_at"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    
    // Relationships
    Creator      User          `json:"creator"`
    Assignee     *User         `json:"assignee"`
    Applications []Application `json:"applications"`
}
```

### Application
```go
type Application struct {
    ID          uint      `json:"id"`
    TaskID      uint      `json:"task_id"`
    ApplicantID uint      `json:"applicant_id"`
    ProposedFee float64   `json:"proposed_fee"`
    Status      string    `json:"status"`      // pending, accepted, declined
    Message     string    `json:"message"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    
    // Relationships
    Applicant User `json:"applicant"`
    Task      Task `json:"task"`
}
```

### Review
```go
type Review struct {
    ID             uint      `json:"id"`
    TaskID         uint      `json:"task_id"`
    ReviewerID     uint      `json:"reviewer_id"`
    ReviewedUserID uint      `json:"reviewed_user_id"`
    Rating         int       `json:"rating"`      // 1-5 stars
    Comment        string    `json:"comment"`
    CreatedAt      time.Time `json:"created_at"`
    
    // Relationships
    Task         Task `json:"task"`
    Reviewer     User `json:"reviewer"`
    ReviewedUser User `json:"reviewed_user"`
}
```

## Key Features

### Authentication & Authorization
- JWT-based authentication
- Role-based access control (task creators vs assignees)
- Password hashing with bcrypt

### Task Lifecycle Management
- Create → Apply → Accept/Decline → In Progress → Complete → Review
- Status transitions: `open` → `in_progress` → `completed` → `cancelled`
- Deadline validation (minimum 24 hours in future)

### Advanced Search & Filtering
- Search by title/description
- Filter by status, price range, deadline
- Sorting by fee, deadline, creation date
- Pagination support
- User-specific views (created vs assigned tasks)

### Fee Negotiation
- Applicants propose custom fees when applying
- Original task fee vs proposed fee comparison
- Accept/decline applications with proposed fees

### Review System
- Bidirectional reviews (creator ↔ assignee)
- 1-5 star rating system
- Optional text comments
- Average rating calculation for users

## Business Rules

### Task Creation
- Title and description required
- Deadline must be at least 24 hours in future
- Fee must be specified
- Creator cannot apply to own tasks

### Application Process
- Users can apply with proposed fee and message
- One application per user per task
- Task creator can accept/decline applications
- Accepting an application assigns the task and changes status to `in_progress`

### Task Assignment
- Only task creators can accept/decline applications
- Accepting assigns the applicant to the task
- Only assigned users can update task status to `completed`
- Task creators and assignees can update task status

### Review System
- Reviews only allowed after task completion
- Both parties (creator and assignee) can review each other
- One review per user per task
- Rating must be 1-5 stars

## Security Features
- JWT token authentication
- Password hashing with bcrypt
- Input validation and sanitization
- Authorization checks for resource access
- CORS enabled (currently allows all origins)

## Database
- PostgreSQL with GORM ORM
- Auto-migration on startup
- Foreign key relationships
- Proper indexing on primary keys

## Removed Components
The following messaging/chat components were removed during cleanup:
- ❌ `message_handlers.go`
- ❌ `message_service.go`
- ❌ `message_repository.go`
- ❌ Message and ConversationSummary models
- ❌ All message-related API endpoints
- ❌ Message database migration

All chat functionality is now handled by the `chat-websocket-service` microservice.

## Current Status
✅ Clean separation from chat functionality
✅ Core task marketplace functionality complete
✅ Authentication and authorization working
✅ Database relationships properly defined
✅ API endpoints follow REST conventions
✅ Error handling implemented
✅ Input validation in place

Ready for production deployment with recommended improvements from `improvements.md`.