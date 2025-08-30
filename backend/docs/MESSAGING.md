# Messaging in MyGuy

All messaging functionality in MyGuy is handled by the dedicated Chat WebSocket Service located in the `chat-websocket-service` directory.

## Architecture

The chat service is responsible for:

- Real-time messaging via WebSockets
- Message persistence
- Room/conversation management
- Content filtering
- Read receipts
- Message limits and moderation
- Notifications
- Conversation tracking
- Typing indicators
- User presence
- Message editing and deletion

## Message Types

The chat service handles all types of messages in the system:

1. Task Messages
   - Between task owner and applicants/assignee
   - Message limits based on task status

2. Application Messages
   - Between task owner and applicant
   - Communication during application process

3. Store Item Messages
   - Between item seller and potential buyers
   - Message limits based on booking status

## Integration

To integrate with the chat service:

1. WebSocket Events:
```javascript
// Connect to chat service
const socket = io('ws://localhost:8000', {
  auth: { token: userToken }
});

// Join a conversation
socket.emit('join:conversation', {
  taskId: 123 // or applicationId or itemId
});

// Send a message
socket.emit('message:send', {
  taskId: 123,
  recipientId: 456,
  content: "Hello"
});

// Listen for new messages
socket.on('message:new', (message) => {
  // Handle new message
});
```

2. REST API Endpoints:
   - GET /api/v1/store-messages/:itemId
   - POST /api/v1/store-messages
   - GET /api/v1/store-messages/:itemId/limits
   - GET /api/v1/tasks/:taskId/messages
   - POST /api/v1/tasks/:taskId/messages
   - GET /api/v1/applications/:applicationId/messages
   - POST /api/v1/applications/:applicationId/messages

For detailed WebSocket events and REST API documentation, refer to the [Chat Service README](../chat-websocket-service/README.md).

## Security

The chat service implements:
- JWT authentication for both WebSocket and HTTP endpoints
- Content filtering to remove sensitive information
- Rate limiting and message count restrictions
- Private conversations with proper access control
