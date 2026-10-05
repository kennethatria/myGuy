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
- `GET /health` - Health check
- `GET /api/v1/time`, `GET /api/v1/server-time` - Server time
- `POST /api/v1/auth/request-code` - Email a 6-digit sign-in code (`{email}`)
- `POST /api/v1/auth/verify-code` - Verify it (`{email, code}`): returns `{token, user}`, or `{signup_token}` for a new email
- `POST /api/v1/auth/complete-signup` - Create the account (`{signup_token, full_name}`): returns `{token, user}`

### Protected Endpoints (Require JWT)

#### Tasks
- `POST /api/v1/tasks` - Create a task
- `GET /api/v1/tasks` - List tasks (search, filters, sorting, pagination)
- `GET /api/v1/tasks/:id` - Get a task
- `PUT /api/v1/tasks/:id` - Update (creator only)
- `PATCH /api/v1/tasks/:id/status` - Change status
- `DELETE /api/v1/tasks/:id` - Delete (creator only; never-assigned tasks only)

#### Applications
- `POST /api/v1/tasks/:id/apply` - Apply with a proposed fee and message
- `GET /api/v1/tasks/:id/applications` - The creator sees all; an applicant only their own
- `PATCH /api/v1/tasks/:id/applications/:applicationId` - Accept or decline (creator only)
- `GET /api/v1/user/applications` - The current user's applications with each task's status
- `GET /api/v1/applications/:id/participants` - Who may chat about an application (participants only; used by the chat service)

#### Your tasks
- `GET /api/v1/user/tasks` - Tasks you created
- `GET /api/v1/user/tasks/assigned` - Tasks assigned to you

#### Reviews
- `POST /api/v1/tasks/:id/reviews` - Review the other participant of a completed task
- `GET /api/v1/tasks/:id/reviews/mine` - Whether you already reviewed this task (`{reviewed}`)
- `GET /api/v1/users/:id/reviews` - Reviews a user received

#### Users
- `GET /api/v1/users/:id` - A user's public profile (no email or phone unless it's you)
- `GET /api/v1/profile` - Your profile
- `PUT /api/v1/profile` - Update your full name and bio (the email is the sign-in identity and can't be changed here)

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
- Passwordless sign-in with emailed one-time codes (`internal/services/auth_service.go`); codes expire in 10 minutes, allow 5 wrong guesses, and are limited to 5 per email per hour. Only an HMAC of each code is stored.
- JWT sessions shared with the store and chat services; signup tokens use a separate derived key so they never work as sessions.
- Role-based access (task creators vs assignees vs applicants).
- Other users' email and phone numbers are never included in responses.

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
- Applicants propose their own fee when applying
- Accepting an application sets the task's fee to the proposed one

### Review System
- Bidirectional reviews (creator ↔ assignee), 1–5 stars with an optional comment
- Each user's `average_rating` is recomputed after every review (and at startup)
- Profiles combine these with store ratings from the store service into one rating

### Task Events in Messages
Task events are posted as system messages into the owner↔applicant conversation in the chat service (`internal/chatnotify`, best effort, never blocking the action):
- New application → owner
- Accepted → the applicant; everyone else still waiting is told they weren't selected
- Declined → the applicant
- Task cancelled → everyone still waiting

Requires `INTERNAL_API_KEY` (shared with the chat service) and `CHAT_API_URL`.

## Business Rules

### Task Creation
- Title and description required
- Deadline must be at least 24 hours in future
- Fee must be specified
- Creator cannot apply to own tasks

### Application Process
- Apply with a proposed fee and an optional message
- One application per user per task; you can't apply to your own task
- Only open tasks accept applications

### Task Assignment
- Only the task creator can accept or decline, and only pending applications to that task
- Accepting is atomic (only one acceptance can win) and declines every other pending application
- A task is never `in_progress` without an assignee
- Cancelling declines all pending applications
- Tasks that were ever assigned can't be deleted (cancel them instead); deleting removes the task's applications too

### Review System
- Reviews only allowed after task completion
- Both parties (creator and assignee) can review each other
- One review per user per task
- Rating must be 1-5 stars

## Security Features
- JWT authentication; passwordless sign-in (no passwords stored)
- Rate-limited, single-use, hashed sign-in codes
- Input validation and authorization checks on every resource
- Contact details (email, phone) hidden from other users
- CORS currently allows all origins; restrict to the frontend URL before scaling

## Database
- PostgreSQL with GORM ORM
- Auto-migration on startup
- Foreign key relationships
- Proper indexing on primary keys

## Configuration
| Variable | Purpose |
| :--- | :--- |
| `DB_CONNECTION` | PostgreSQL connection string |
| `JWT_SECRET` | Shared with store and chat services |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | Sign-in code email. With `SMTP_HOST` unset, codes are logged instead (local development) |
| `INTERNAL_API_KEY`, `CHAT_API_URL` | Posting task events into Messages via the chat service |
| `ZIPKIN_URL` | Trace export |

## Testing
```bash
go test ./...                       # unit + handler tests (SQLite for repositories)
GOWORK=off GOFLAGS=-mod=mod go test ./internal/... -coverprofile=c.out   # as CI runs it (≥70% required)
TEST_POSTGRES_DSN="postgres://..." go test ./internal/repositories/ -run Postgres   # optional real-Postgres checks
```
