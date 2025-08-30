# Store Message Integration

## Overview
This document outlines how store-related messages are integrated into the main chat service. Messages sent between users about store items appear both in the store interface and the centralized chat system.

## Message Flow

### 1. Store Message Creation
When a user sends a message about a store item:
1. Message is created in unified `messages` table with `store_item_id`
2. WebSocket event is emitted to both the sender and item owner
3. Conversation thread is created/updated in chat service

### 2. Conversation Handling
Store conversations are handled through:
- Direct message thread between user and item owner
- Item context maintained through `store_item_id` reference
- Real-time updates via WebSocket events

### 3. Access Control
- Item owners see all messages about their items
- Users only see their own conversations
- Messages are private between sender and item owner

## Database Schema

### Messages Table
```sql
CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    store_item_id INTEGER REFERENCES store_items(id),
    sender_id INTEGER NOT NULL REFERENCES users(id),
    recipient_id INTEGER NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    is_edited BOOLEAN DEFAULT FALSE,
    edited_at TIMESTAMP,
    original_content TEXT
);

-- Indexes
CREATE INDEX idx_messages_store_item ON messages(store_item_id);
CREATE INDEX idx_messages_store_participants ON messages(store_item_id, sender_id, recipient_id);
```

## WebSocket Events

### Store Message Events
1. `message:send` - Send new store message
   ```javascript
   {
     itemId: number,
     recipientId: number,
     content: string
   }
   ```

2. `message:new` - New store message received
   ```javascript
   {
     id: number,
     store_item_id: number,
     sender: { id: number, username: string },
     recipient: { id: number, username: string },
     content: string,
     created_at: string
   }
   ```

### Room Management
- Users join rooms: `item:{itemId}`
- Item owners automatically join rooms for their items
- Real-time updates sent to all room participants

## Integration Points

### Store Service
- Uses unified message service for all communications
- Maintains item context in messages
- Handles message limits and permissions

### Chat Service
- Shows store conversations in main chat interface
- Maintains message history and read status
- Provides real-time updates

## Frontend Components

### Store Message Components
- Message popup in store item view
- Conversation thread in chat interface
- Unread message indicators

### Chat Integration
- Store conversations appear in main chat list
- Item context shown in conversation header
- Full chat functionality (edit, delete, read receipts)

## Testing Scenarios

1. Store Message Creation
   - Send message from store item view
   - Verify message appears in both store and chat interfaces
   - Check real-time updates for both sender and recipient

2. Item Owner Access
   - Verify item owners see all messages about their items
   - Test message notifications and real-time updates
   - Confirm conversation appears in chat list

3. User Access
   - Test message visibility restrictions
   - Verify conversation continuity
   - Check message history access

## Error Handling

1. Message Failures
   - Failed message delivery retry
   - Error notifications to users
   - Data consistency checks

2. Access Control
   - Invalid recipient handling
   - Unauthorized access prevention
   - Permission verification

## Performance Considerations

1. Message Loading
   - Pagination for long conversations
   - Efficient message retrieval
   - Optimized database queries

2. Real-time Updates
   - WebSocket connection management
   - Event throttling
   - Resource optimization

## Future Improvements

1. Planned Enhancements
   - Group conversations for items
   - Rich media support
   - Advanced search capabilities

2. Scalability
   - Message archiving
   - Performance optimization
   - Load distribution
