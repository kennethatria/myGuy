# MyGuy - Task Marketplace Platform

A modern task marketplace application where users can create tasks (gigs) for others to complete, negotiate fees, communicate about requirements, and leave reviews.

## Current Status

### Architecture
- **Microservices-based** with clean separation of concerns
- **Main Backend**: Task management, user authentication, reviews
- **Chat Service**: Real-time messaging via WebSocket 
- **Store Service**: Item marketplace with bidding system

### Recent Updates
- ✅ Messaging functionality separated to dedicated chat microservice
- ✅ Clean backend architecture with no messaging dependencies
- ✅ **Store Functionality Fixed**: Listing visibility and access control implemented
- ✅ **Currency Conversion**: Complete migration from USD to UGX (Uganda Shillings)
- ✅ **JSON API Support**: Store service now supports both JSON and form data requests
- ⚠️ **Testing Required**: Backend has zero test coverage (see `improvements/`)
- 📋 **Roadmap Available**: See `improvements/` folder for enhancement plans

## Features

### Core Functionality
- **User Authentication**: Secure registration and login system with JWT tokens
- **Task Management**: Create, browse, and manage tasks through their complete lifecycle
- **Advanced Search & Filtering**: Search tasks by title/description with filters for status, price range, and deadline
- **Fee Negotiation**: Applicants can propose their fees when applying for tasks
- **Real-time Communication** (Chat Microservice): 
  - WebSocket-based instant messaging with Socket.IO
  - Task-level messaging between creators and assignees
  - Application-specific messaging for pre-assignment communication
  - Message editing and soft deletion
  - Read receipts and typing indicators
  - Automatic content filtering (removes URLs, emails, phone numbers)
  - Message lifecycle management with automatic deletion
- **Store Marketplace** (Store Microservice): 
  - List items for sale with fixed prices or bidding
  - Auction system with starting bids and increments
  - Item categories and condition tracking
  - **Access Control**: Users can view their own listings but cannot bid/purchase their own items
  - **Listing Visibility**: All active listings are visible to users, with proper status filtering
  - **Owner Indicators**: Clear visual indicators when viewing own listings
  - **UGX Currency**: All prices displayed in Uganda Shillings (UGX)
- **Review System**: Both parties can review each other after task completion with 1-5 star ratings
- **User Profiles**: View user profiles with ratings and review history

### Key Features by User Role

#### As a Task Creator
- Post tasks with title, description, deadline, and budget
- View and manage applications with proposed fees
- Accept or decline applications
- Message with assignees during task execution (via Chat Service)
- Mark tasks as complete and review assignees

#### As a Task Applicant/Assignee
- Browse available tasks with advanced filtering
- Search tasks by keywords
- Apply for tasks with custom fee proposals
- Communicate with task creators about requirements (via Chat Service)
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
├── backend/              # Go backend server (Task management, Auth, Reviews)
│   ├── cmd/             # Application entrypoints
│   ├── internal/        # Private application code
│   │   ├── api/        # HTTP handlers
│   │   ├── models/     # Database models (User, Task, Application, Review)
│   │   ├── services/   # Business logic
│   │   └── repositories/ # Data access layer
│   └── Dockerfile      # Backend container definition
├── store-service/       # Store microservice (Go)
│   ├── cmd/            # Application entrypoint
│   ├── internal/       # Service implementation
│   └── Dockerfile      # Service container
├── chat-websocket-service/ # Real-time chat service (Node.js)
│   ├── src/            # Service implementation
│   ├── migrations/     # Database migrations
│   └── Dockerfile      # Service container
├── myGuy/              # Vue.js frontend
│   ├── src/
│   │   ├── components/ # Reusable components
│   │   ├── views/     # Page components
│   │   ├── stores/    # Pinia stores
│   │   └── router/    # Route definitions
│   └── package.json
├── improvements/        # Enhancement roadmaps and recommendations
│   ├── improvements.md  # General backend improvements
│   ├── improvements-user-management.md # Auth microservice plan
│   └── improvements-tests.md # Critical testing requirements
├── current_state.md     # Current backend functionality documentation
└── docker-compose.yml  # Container orchestration
```

## API Endpoints

### Main Backend (Port 8080)

#### Authentication
- `POST /api/v1/register` - Register new user
- `POST /api/v1/login` - Login user

#### Tasks
- `GET /api/v1/tasks` - List tasks with pagination, search, and filters
  - Query params: `search`, `status`, `min_fee`, `max_fee`, `deadline_before`, `sort_by`, `sort_order`, `page`, `per_page`
- `POST /api/v1/tasks` - Create a new task
- `GET /api/v1/tasks/:id` - Get task details
- `PUT /api/v1/tasks/:id` - Update a task
- `PATCH /api/v1/tasks/:id/status` - Update task status
- `DELETE /api/v1/tasks/:id` - Delete a task
- `POST /api/v1/tasks/:id/apply` - Apply for a task
- `GET /api/v1/tasks/:id/applications` - Get task applications
- `PATCH /api/v1/tasks/:id/applications/:applicationId` - Accept/decline application

#### User Tasks
- `GET /api/v1/user/tasks` - Get tasks created by current user
- `GET /api/v1/user/tasks/assigned` - Get tasks assigned to current user

#### Reviews
- `POST /api/v1/tasks/:id/reviews` - Create a review
- `GET /api/v1/users/:id/reviews` - Get user reviews

#### Users & Profile
- `GET /api/v1/users/:id` - Get user details
- `GET /api/v1/profile` - Get current user profile
- `PUT /api/v1/profile` - Update current user profile

#### Utility
- `GET /api/v1/server-time` - Get server time and deadline examples

### Chat WebSocket Service (Port 8082)
**All messaging functionality handled by dedicated chat microservice**

#### HTTP API Endpoints
- `GET /api/v1/tasks/:taskId/messages` - Get messages for a specific task
- `POST /api/v1/tasks/:taskId/messages` - Send message to task conversation
  - Body: `{recipient_id: number, content: string}`
- `GET /api/v1/applications/:applicationId/messages` - Get messages for a specific application
- `POST /api/v1/applications/:applicationId/messages` - Send message to application conversation
  - Body: `{content: string}`
- `GET /api/v1/deletion-warnings` - Get pending message deletion warnings
- `POST /api/v1/deletion-warnings/:id/shown` - Mark deletion warning as shown
- `GET /api/v1/users/:id/last-seen` - Get user's last seen timestamp

#### WebSocket Features
- Real-time WebSocket connections
- Task-level messaging
- Application-specific messaging
- Message editing, deletion, read receipts
- Typing indicators and presence
- Automatic content filtering

### Store Service (Port 8081)
**Item marketplace functionality with JSON API support**

#### Store Items
- `GET /api/v1/items` - List all active items (public endpoint)
  - Query params: `search`, `category`, `price_type`, `condition`, `status`, `seller_id`, `min_price`, `max_price`, `sort_by`, `sort_order`, `page`, `per_page`
  - Returns: `{items: [...], total: X, page: Y, per_page: Z}`
- `POST /api/v1/items` - Create new item listing (requires auth)
  - Supports both JSON and form data requests
  - JSON format: `{title, description, price_type: "fixed|bidding", fixed_price?, starting_bid?, min_bid_increment?, category, condition, images[]}`
- `GET /api/v1/items/:id` - Get item details
- `PUT /api/v1/items/:id` - Update item (owner only)
- `DELETE /api/v1/items/:id` - Delete item (owner only)

#### Bidding System
- `POST /api/v1/items/:id/bids` - Place bid on auction item
  - Validation: Cannot bid on own items, minimum bid requirements
- `GET /api/v1/items/:id/bids` - Get bid history for item
- `POST /api/v1/items/:id/bids/:bidId/accept` - Accept winning bid (seller only)

#### Purchase System
- `POST /api/v1/items/:id/purchase` - Purchase fixed-price item
  - Validation: Cannot purchase own items, item must be active

#### User Management
- `GET /api/v1/user/listings` - Get current user's listings
- `GET /api/v1/user/purchases` - Get current user's purchases
- `GET /api/v1/user/bids` - Get current user's bids

#### Access Control Features
- **Frontend**: Bid/purchase buttons hidden for own items
- **Backend**: Server-side validation prevents bidding/purchasing own items
- **Status Filtering**: Only active items shown by default
- **Owner Identification**: Clear visual indicators for own listings

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
- Store Service: http://localhost:8081
- Chat WebSocket Service: http://localhost:8082

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

### Main Backend Database
- `users` - User accounts with authentication and profile data
- `tasks` - Task/gig listings with status tracking
- `applications` - Task applications with proposed fees
- `reviews` - User reviews and ratings

### Microservice Databases
- `messages` - Real-time communication (Chat WebSocket Service)
- `store_items` - Marketplace items and bids (Store Service)

## Testing

### ⚠️ Critical Issue: Backend Testing
**Current Status**: Backend has **ZERO test coverage**

**Required Action**: Implement comprehensive testing before production deployment.
See `improvements/improvements-tests.md` for detailed testing requirements and implementation plan.

### Frontend Tests
```bash
cd myGuy
npm run test:unit    # Unit tests
npm run test:e2e     # End-to-end tests
```

## Microservices

### Quick Start
```bash
# Start all services
docker-compose up -d

# Stop all services  
docker-compose down

# Start specific service
docker-compose up -d chat-websocket-service
docker-compose up -d store-service

# View logs
docker-compose logs -f chat-websocket-service
```

### Available Microservices

1. **Main Backend Service** (Port 8080)
   - Core task management
   - User authentication and profiles
   - Application handling
   - Review system
   - **Clean architecture**: No messaging dependencies

2. **Store Service** (Port 8081)
   - Item marketplace
   - Bidding system
   - Purchase management

3. **Chat WebSocket Service** (Port 8082)
   - Real-time messaging
   - WebSocket connections
   - Message lifecycle management
   - Content filtering and moderation

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

Scale a service:
```bash
docker-compose up -d --scale chat-websocket-service=3
```

## Development Roadmap

### Immediate Priorities
1. **Testing Implementation** - See `improvements/improvements-tests.md`
   - Critical: Backend has zero test coverage
   - Implement unit, integration, and security tests

2. **Security Enhancements** - See `improvements/improvements.md`
   - Rate limiting, CORS configuration
   - Input validation improvements
   - Database indexing for performance

3. **Authentication Microservice** - See `improvements/improvements-user-management.md`
   - Extract auth to dedicated service
   - Enable token validation across all services

### Documentation
- `current_state.md` - Complete backend functionality overview
- `improvements/` - Detailed enhancement roadmaps
- Each improvement file contains implementation checklists

## Troubleshooting

### Store Service Issues

#### Listings Not Visible
If store listings are not appearing:

1. **Check Services**: Ensure all Docker containers are running:
   ```bash
   docker ps
   docker-compose up -d  # If containers are stopped
   ```

2. **Verify Database Connection**: Check store service logs:
   ```bash
   docker logs myguy-store-service-1
   ```

3. **Test API Directly**: 
   ```bash
   curl "http://localhost:8081/api/v1/items"
   ```

4. **Check Response Format**: Frontend expects `{items: [...]}` format from backend

#### Common Solutions Applied
- **Fixed Response Parsing**: Frontend now correctly extracts `data.items` from API response
- **Added Status Filtering**: Backend defaults to showing only `active` items
- **JSON API Support**: Store service now supports both JSON and form data requests
- **Access Control**: Proper validation prevents users from bidding/purchasing own items

### General Issues

#### Docker Container Problems
```bash
# Rebuild and restart all services
docker-compose down
docker-compose up --build

# Check specific service logs
docker-compose logs -f store-service
```

#### Database Connection Issues
Ensure PostgreSQL container is healthy before other services start:
```bash
docker-compose logs postgres-db
```

## ☁️ Infrastructure & Deployment

### **Akamai Cloud Infrastructure**

MyGuy is designed for deployment on Akamai Cloud with cost-optimized architecture:

**Production Environment** (~€39/month):
- 1x Linode Standard 2GB Instance (€9/month)
- 1x Managed PostgreSQL 1GB (€15/month)
- 1x Object Storage 250GB (€5/month)
- 1x Load Balancer with SSL (€10/month)

**Staging Environment** (~€7/month):
- 1x Linode Nanode 1GB Instance (€4.50/month)
- Shared PostgreSQL database
- Object Storage 50GB (€2.50/month)

### **Automated Deployment**

GitHub Actions workflows handle automatic deployment with cost optimization:

- **Staging**: 
  - Deploys when PR is opened/updated → `staging.myguy.work`
  - **Automatically destroyed** when PR is closed (saves €7/month)
  - **Automatically recreated** when PR is reopened
- **Production**: 
  - Deploys on merge to `main` → `myguy.work`
  - **Destroys staging first** to prevent double billing
- **Infrastructure**: Terraform manages all cloud resources

### **Quick Deployment Setup**

1. **Configure GitHub Secrets**:
```bash
LINODE_TOKEN=your_api_token
SSH_PUBLIC_KEY=your_ssh_public_key
SSH_PRIVATE_KEY=your_ssh_private_key
JWT_SECRET=your_jwt_secret
POSTGRES_PASSWORD=your_db_password
```

2. **Deploy Infrastructure**:
```bash
cd terraform
terraform init
terraform apply -var-file="environments/production/terraform.tfvars"
```

3. **Configure DNS**:
- Point `myguy.work` to load balancer IP
- Point `staging.myguy.work` to staging instance IP

### **Architecture Benefits**

- **Cost-Optimized**: Only €39/month when production is active (staging auto-destroyed)
- **Development-Friendly**: Staging created/destroyed automatically with PRs
- **Zero-Downtime**: Rolling deployments with health checks
- **Secure**: SSL certificates, VPC isolation, firewall rules
- **Scalable**: Easy to upgrade instances as user base grows
- **Resilient**: Automatic rollback capabilities and backups

For detailed infrastructure documentation, see [`terraform/README.md`](terraform/README.md).

## Contributing

1. Fork the repository
2. Review `improvements/` folder for current priorities
3. Create your feature branch (`git checkout -b feature/amazing-feature`)
4. **Add tests** for new functionality (required)
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Create a new Pull Request

## License

This project is licensed under the MIT License.