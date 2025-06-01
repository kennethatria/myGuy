# Chat WebSocket Service - MyGuy Platform

A real-time messaging microservice for the MyGuy platform that handles WebSocket connections, message delivery, read receipts, and content filtering.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Features](#features)
- [Technology Stack](#technology-stack)
- [Database Schema](#database-schema)
- [WebSocket Events](#websocket-events)
- [API Endpoints](#api-endpoints)
- [Authentication](#authentication)
- [Message Lifecycle](#message-lifecycle)
- [Content Filtering](#content-filtering)
- [Message Deletion Policy](#message-deletion-policy)
- [Development Setup](#development-setup)
- [Configuration](#configuration)
- [Testing](#testing)
- [Monitoring & Debugging](#monitoring--debugging)
- [Security Considerations](#security-considerations)
- [Scaling](#scaling)

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Vue.js App    │────▶│  WebSocket      │────▶│   PostgreSQL    │
│  (Socket.IO     │◀────│    Service      │◀────│    Database     │
│    Client)      │     │  (Socket.IO)    │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │   Scheduler     │
                        │   (node-cron)   │
                        └─────────────────┘
```

### Key Components

1. **WebSocket Server**: Handles real-time bidirectional communication
2. **Message Service**: Business logic for message operations
3. **Content Filter**: Removes URLs, emails, and phone numbers
4. **Scheduler Service**: Manages message deletion lifecycle
5. **Database Layer**: PostgreSQL with connection pooling

## Features

### Real-time Messaging
- Instant message delivery via WebSocket
- Typing indicators
- Online/offline presence tracking
- Message notifications across devices

### Message Management
- Edit messages (with edit history tracking)
- Soft delete with "[Message deleted]" placeholder
- Read receipts with timestamps
- Message filtering (removes contact information)
- Original content preservation for audit

### Conversation Features
- Group messages by task/application
- Unread message counters
- Last message preview
- Conversation search and filtering
- Pagination for message history

### Automated Lifecycle
- Automatic message deletion after:
  - 6 months for completed tasks
  - 1 month for cancelled/inactive tasks
- 30-day advance deletion warnings
- User notification system

## Technology Stack

- **Runtime**: Node.js 18+
- **WebSocket**: Socket.IO 4.7+
- **HTTP Server**: Express 4.18+
- **Database**: PostgreSQL with pg driver
- **Authentication**: JWT (jsonwebtoken)
- **Scheduling**: node-cron
- **Logging**: Winston
- **Security**: Helmet, CORS
- **Environment**: dotenv

## Database Schema

### Messages Table (Extended)
```sql
CREATE TABLE messages (
  id SERIAL PRIMARY KEY,
  task_id INTEGER NOT NULL,
  application_id INTEGER,
  sender_id INTEGER NOT NULL,
  recipient_id INTEGER NOT NULL,
  content TEXT NOT NULL,
  original_content TEXT,
  is_read BOOLEAN DEFAULT FALSE,
  read_at TIMESTAMP,
  is_edited BOOLEAN DEFAULT FALSE,
  edited_at TIMESTAMP,
  is_deleted BOOLEAN DEFAULT FALSE,
  deleted_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);
```

### User Activity Table
```sql
CREATE TABLE user_activity (
  user_id INTEGER PRIMARY KEY,
  last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
  last_conversation_id INTEGER,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

### Message Deletion Warnings Table
```sql
CREATE TABLE message_deletion_warnings (
  id SERIAL PRIMARY KEY,
  task_id INTEGER NOT NULL,
  deletion_scheduled_at TIMESTAMP NOT NULL,
  warning_shown BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(task_id)
);
```

## WebSocket Events

### Client → Server Events

#### Connection & Room Management
```javascript
// Join a conversation room
socket.emit('join:conversation', {
  taskId: 123,          // OR
  applicationId: 456
});

// Leave a conversation room
socket.emit('leave:conversation', {
  taskId: 123
});
```

#### Message Operations
```javascript
// Send a message
socket.emit('message:send', {
  taskId: 123,
  recipientId: 456,
  content: "Hello!"
});

// Edit a message
socket.emit('message:edit', {
  messageId: 789,
  content: "Updated message"
});

// Delete a message
socket.emit('message:delete', {
  messageId: 789
});

// Mark message as read
socket.emit('message:read', {
  messageId: 789
});

// Mark entire conversation as read
socket.emit('conversation:read', {
  taskId: 123
});
```

#### Data Fetching
```javascript
// Get conversations list
socket.emit('conversations:list');

// Get messages for a conversation
socket.emit('messages:get', {
  taskId: 123,
  limit: 5,      // optional, default: 5
  offset: 0      // optional, for pagination
});

// Get user's last seen
socket.emit('user:lastseen', {
  userId: 456
});
```

#### Typing Indicators
```javascript
// Start typing
socket.emit('typing:start', {
  taskId: 123
});

// Stop typing
socket.emit('typing:stop', {
  taskId: 123
});
```

### Server → Client Events

#### Message Events
```javascript
// New message received
socket.on('message:new', (message) => {
  // message object with all fields
});

// Message sent confirmation
socket.on('message:sent', (message) => {
  // Confirms message was saved
});

// Message edited
socket.on('message:edited', (message) => {
  // Updated message object
});

// Message deleted
socket.on('message:deleted', ({ messageId }) => {
  // ID of deleted message
});

// Message read receipt
socket.on('message:read', ({ messageId, readAt }) => {
  // Read confirmation
});

// Content filtered warning
socket.on('message:filtered', ({ 
  messageId, 
  warning: "Links and contact information have been removed" 
}) => {
  // Show warning to user
});
```

#### Conversation Events
```javascript
// Conversations list
socket.on('conversations:list', (conversations) => {
  // Array of conversation summaries
});

// Messages list
socket.on('messages:list', ({ taskId, messages, offset }) => {
  // Paginated messages
});

// Conversation marked as read
socket.on('conversation:marked-read', ({ taskId, count }) => {
  // Number of messages marked
});
```

#### User Events
```javascript
// User typing
socket.on('user:typing', ({ userId, userName, conversationId }) => {
  // Show typing indicator
});

// User stopped typing
socket.on('user:stopped-typing', ({ userId, conversationId }) => {
  // Hide typing indicator
});

// User presence update
socket.on('user:presence', ({ userId, isOnline, lastSeen }) => {
  // Update user status
});

// User last seen
socket.on('user:lastseen', ({ userId, lastSeen }) => {
  // Timestamp or null
});
```

#### System Events
```javascript
// Error handling
socket.on('error', ({ message }) => {
  // Handle error
});

// Connection events
socket.on('connect', () => {
  // Connected to server
});

socket.on('disconnect', () => {
  // Disconnected from server
});
```

## API Endpoints

### GET /health
Health check endpoint
```json
{
  "status": "ok",
  "service": "chat-websocket-service",
  "version": "1.0.0",
  "uptime": 12345
}
```

### GET /api/v1/deletion-warnings
Get pending deletion warnings for the authenticated user
```json
[
  {
    "id": 1,
    "task_id": 123,
    "task_title": "Fix login bug",
    "deletion_scheduled_at": "2024-07-15T00:00:00Z",
    "warning_shown": false
  }
]
```

### POST /api/v1/deletion-warnings/:id/shown
Mark a deletion warning as shown
```json
{
  "success": true
}
```

### GET /api/v1/users/:id/last-seen
Get user's last seen timestamp
```json
{
  "userId": 123,
  "lastSeen": "2024-01-15T10:30:00Z"
}
```

## Authentication

### JWT Token Structure
```javascript
{
  "user_id": 123,
  "email": "user@example.com",
  "name": "John Doe",
  "iat": 1234567890,
  "exp": 1234654290
}
```

### Socket Authentication
```javascript
const socket = io('http://localhost:8082', {
  auth: {
    token: 'eyJhbGciOiJIUzI1NiIs...'
  }
});
```

### HTTP Authentication
```http
GET /api/v1/deletion-warnings
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

## Message Lifecycle

### 1. Message Creation
```
Client → Send Message → Content Filter → Save to DB → Emit to Recipients
```

### 2. Message Editing
```
Client → Edit Request → Verify Owner → Content Filter → Update DB → Broadcast Changes
```

### 3. Message Deletion (Soft)
```
Client → Delete Request → Verify Owner → Mark Deleted → Update Content → Broadcast
```

### 4. Automatic Deletion
```
Scheduler → Check Tasks → Create Warnings → Wait 30 Days → Delete Messages → Remove Records
```

## Content Filtering

### Filtered Patterns
1. **URLs**: HTTP(S), FTP, www links
2. **Emails**: Standard email format
3. **Phone Numbers**: 10-15 digit patterns with common formats
4. **Social Handles**: @mentions

### Example Transformations
```
Input:  "Call me at 555-1234 or email john@example.com"
Output: "Call me at [phone removed] or email [email removed]"

Input:  "Check out https://example.com for more info"
Output: "Check out [link removed] for more info"
```

### Implementation
```javascript
const patterns = {
  urls: /(?:https?|ftp|ftps):\/\/[^\s]+|www\.[^\s]+\.[^\s]+/gi,
  emails: /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g,
  phones: /(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}/g
};
```

## Message Deletion Policy

### Deletion Schedule
1. **Completed Tasks**: Messages deleted 6 months after task completion
2. **Cancelled/Inactive Tasks**: Messages deleted 1 month after last activity
3. **Warning Period**: Users notified 30 days before deletion

### Deletion Process
```
Daily at 2 AM: Check for messages to delete
Daily at 3 AM: Create deletion warnings
Daily at 4 AM: Execute scheduled deletions
```

### Warning Display
- Banner shown on login if warnings exist
- Warnings persist until acknowledged
- Shows exact deletion date for each conversation

## Development Setup

### Prerequisites
- Node.js 18+
- PostgreSQL 12+
- Docker (optional)

### Local Development
```bash
# Clone repository
git clone <repository-url>
cd chat-websocket-service

# Install dependencies
npm install

# Create .env file
cp .env.example .env
# Edit .env with your configuration

# Run database migrations
psql -U postgres -d myguy -f migrations/001_message_updates.sql

# Start development server
npm run dev
```

### Docker Development
```bash
# Build and run with docker-compose
docker-compose up chat-websocket-service
```

## Configuration

### Environment Variables
```env
# Server
PORT=8082
NODE_ENV=development
LOG_LEVEL=info

# Database
DB_CONNECTION=postgresql://user:pass@localhost:5432/myguy

# Authentication
JWT_SECRET=your-secret-key

# Client
CLIENT_URL=http://localhost:5173
```

### Configuration Options
- **Connection Pool**: Max 20 connections, 30s idle timeout
- **Socket.IO**: 60s ping timeout, 25s ping interval
- **Message Pagination**: Default 5 messages per request
- **Typing Timeout**: 3 seconds auto-stop
- **Scheduler Timezone**: UTC

## Testing

### Unit Tests
```bash
npm test
```

### Integration Tests
```bash
npm run test:integration
```

### Load Testing
```javascript
// Example Socket.IO load test
const io = require('socket.io-client');
const sockets = [];

for (let i = 0; i < 100; i++) {
  const socket = io('http://localhost:8082', {
    auth: { token: generateToken(i) }
  });
  sockets.push(socket);
}
```

### Testing Scenarios
1. **Message Delivery**: Ensure messages reach all recipients
2. **Read Receipts**: Verify read status updates
3. **Content Filtering**: Test all filter patterns
4. **Concurrent Edits**: Handle race conditions
5. **Reconnection**: Test message queue on disconnect

## Monitoring & Debugging

### Logging
```javascript
// Log levels: error, warn, info, debug
logger.info('Socket connected', { userId, socketId });
logger.error('Database error', { error, query });
```

### Health Checks
```bash
# Check service health
curl http://localhost:8082/health

# Check database connection
docker exec chat-websocket-service node -e "
  const { pool } = require('./src/config/database');
  pool.query('SELECT NOW()').then(console.log);
"
```

### Debug Mode
```env
LOG_LEVEL=debug
NODE_ENV=development
```

### Common Issues
1. **Connection Timeouts**: Check JWT expiration
2. **Missing Messages**: Verify room subscriptions
3. **Slow Queries**: Add database indexes
4. **Memory Leaks**: Monitor socket cleanup

## Security Considerations

### Authentication
- JWT tokens required for all connections
- Tokens validated on each request
- User context attached to socket instances

### Authorization
- Users can only edit/delete own messages
- Message access restricted to participants
- Task/application ownership verified

### Input Validation
- Message length limits (1000 chars)
- Content sanitization
- SQL injection prevention via parameterized queries

### Rate Limiting
- Consider implementing per-user message limits
- Typing indicator throttling
- Connection attempt limits

### Data Privacy
- Original content stored but never exposed
- Soft deletes preserve audit trail
- Automatic deletion for compliance

## Scaling

### Horizontal Scaling
```yaml
# docker-compose.yml
chat-websocket-service:
  scale: 3
  deploy:
    replicas: 3
```

### Redis Adapter (Future)
```javascript
// For multi-instance deployments
const { createAdapter } = require('@socket.io/redis-adapter');
io.adapter(createAdapter(redisClient));
```

### Database Optimization
- Connection pooling configured
- Indexes on frequently queried columns
- Partitioning for large message tables

### Performance Tips
1. Enable database query caching
2. Implement message archiving
3. Use CDN for static assets
4. Consider message compression
5. Optimize database indexes

## Troubleshooting

### Connection Issues
```bash
# Test WebSocket connection
wscat -c ws://localhost:8082 -H "Authorization: Bearer TOKEN"
```

### Database Issues
```sql
-- Check active connections
SELECT count(*) FROM pg_stat_activity;

-- Check message table size
SELECT pg_size_pretty(pg_total_relation_size('messages'));
```

### Memory Usage
```bash
# Monitor Node.js memory
node --inspect src/server.js
```

## API Integration Examples

### Frontend Integration
```javascript
import { io } from 'socket.io-client';

const socket = io('http://localhost:8082', {
  auth: { token: localStorage.getItem('token') }
});

// Send message
socket.emit('message:send', {
  taskId: 123,
  recipientId: 456,
  content: 'Hello!'
});

// Listen for new messages
socket.on('message:new', (message) => {
  console.log('New message:', message);
});
```

### Backend Integration
```go
// Notify chat service of task completion
func markTaskCompleted(taskID uint) error {
    task.CompletedAt = time.Now()
    // Chat service will handle message lifecycle
    return db.Save(&task).Error
}
```

## Contributing

1. Follow existing code style
2. Add tests for new features
3. Update documentation
4. Run linter before committing
5. Keep commits focused and descriptive

## License

This service is part of the MyGuy platform and follows the same license terms.