require('dotenv').config();
const express = require('express');
const { createServer } = require('http');
const { Server } = require('socket.io');
const cors = require('cors');
const helmet = require('helmet');

const logger = require('./utils/logger');
const { authenticateSocket, authenticateHTTP } = require('./middleware/auth');
const SocketHandlers = require('./handlers/socketHandlers');
const messageService = require('./services/messageService');
const schedulerService = require('./services/schedulerService');

// Initialize Express app
const app = express();
const httpServer = createServer(app);

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());

// Initialize Socket.IO
const io = new Server(httpServer, {
  cors: {
    origin: process.env.CLIENT_URL || 'http://localhost:5173',
    credentials: true
  },
  pingTimeout: 60000,
  pingInterval: 25000
});

// Socket.IO authentication middleware
io.use(authenticateSocket);

// Initialize socket handlers
const socketHandlers = new SocketHandlers(io);

// Socket.IO connection handling
io.on('connection', (socket) => {
  socketHandlers.handleConnection(socket);
});

// HTTP API Routes

// Health check
app.get('/health', (req, res) => {
  res.json({ 
    status: 'ok', 
    service: 'chat-websocket-service',
    version: '1.0.0',
    uptime: process.uptime()
  });
});


// Get deletion warnings for user
app.get('/api/v1/deletion-warnings', authenticateHTTP, async (req, res) => {
  try {
    const warnings = await messageService.getUserDeletionWarnings(req.user.id);
    res.json(warnings);
  } catch (error) {
    logger.error('Error getting deletion warnings:', error);
    res.status(500).json({ error: 'Failed to get deletion warnings' });
  }
});

// Mark warning as shown
app.post('/api/v1/deletion-warnings/:id/shown', authenticateHTTP, async (req, res) => {
  try {
    await messageService.markWarningAsShown(req.params.id);
    res.json({ success: true });
  } catch (error) {
    logger.error('Error marking warning as shown:', error);
    res.status(500).json({ error: 'Failed to mark warning as shown' });
  }
});

// Task message endpoints

// Get task messages
app.get('/api/v1/tasks/:taskId/messages', authenticateHTTP, async (req, res) => {
  try {
    const taskId = parseInt(req.params.taskId);
    const userId = req.user.id;
    
    if (isNaN(taskId)) {
      return res.status(400).json({ error: 'Invalid task ID' });
    }
    
    const messages = await messageService.getMessages(taskId, userId);
    
    // Format messages to include proper sender/recipient objects
    const formattedMessages = messages.map(msg => ({
      ...msg,
      sender: {
        id: msg.sender_id,
        username: msg.sender_name
      },
      recipient: {
        id: msg.recipient_id,
        username: msg.recipient_name
      }
    }));
    
    res.json(formattedMessages);
  } catch (error) {
    logger.error('Error getting task messages:', error);
    res.status(500).json({ error: 'Failed to get task messages' });
  }
});

// Send task message
app.post('/api/v1/tasks/:taskId/messages', authenticateHTTP, async (req, res) => {
  try {
    const taskId = parseInt(req.params.taskId);
    const { recipient_id, content } = req.body;
    const senderId = req.user.id;
    
    if (isNaN(taskId)) {
      return res.status(400).json({ error: 'Invalid task ID' });
    }
    
    if (!recipient_id || !content || !content.trim()) {
      return res.status(400).json({ error: 'Missing required fields' });
    }
    
    // Check message limit
    const messageCount = await messageService.getUserTaskMessageCount(taskId, senderId);
    const messageLimit = await messageService.getTaskMessageLimit(taskId, senderId);
    
    if (messageCount >= messageLimit) {
      const limitMessage = messageLimit === 15 ? 
        'You\'ve reached your message limit for this gig (15 messages).' : 
        'You\'ve reached your message limit for this gig (3 messages). The limit increases to 15 once you\'re assigned to the task.';
      return res.status(403).json({ 
        error: limitMessage,
        limit: messageLimit,
        count: messageCount
      });
    }
    
    const message = await messageService.sendMessage({
      taskId,
      senderId,
      recipientId: parseInt(recipient_id),
      content: content.trim()
    });
    
    // Get user information to format the response
    const senderQuery = 'SELECT username FROM users WHERE id = $1';
    const recipientQuery = 'SELECT username FROM users WHERE id = $1';
    
    const db = require('./config/database');
    const senderResult = await db.query(senderQuery, [senderId]);
    const recipientResult = await db.query(recipientQuery, [parseInt(recipient_id)]);
    
    // Format message with sender/recipient info
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: senderResult.rows[0]?.username || 'Unknown'
      },
      recipient: {
        id: parseInt(recipient_id),
        username: recipientResult.rows[0]?.username || 'Unknown'
      }
    };
    
    res.status(201).json(formattedMessage);
  } catch (error) {
    logger.error('Error sending task message:', error);
    res.status(500).json({ error: 'Failed to send task message' });
  }
});

// Get message limits and status for a task
app.get('/api/v1/tasks/:taskId/message-limits', authenticateHTTP, async (req, res) => {
  try {
    const taskId = parseInt(req.params.taskId);
    const userId = req.user.id;
    
    if (isNaN(taskId)) {
      return res.status(400).json({ error: 'Invalid task ID' });
    }
    
    const messageCount = await messageService.getUserTaskMessageCount(taskId, userId);
    const messageLimit = await messageService.getTaskMessageLimit(taskId, userId);
    
    res.json({
      messageCount,
      messageLimit,
      canSendMore: messageCount < messageLimit,
      remaining: messageLimit - messageCount
    });
  } catch (error) {
    logger.error('Error getting message limits:', error);
    res.status(500).json({ error: 'Failed to get message limits' });
  }
});

// Get application messages
app.get('/api/v1/applications/:applicationId/messages', authenticateHTTP, async (req, res) => {
  try {
    const applicationId = parseInt(req.params.applicationId);
    const userId = req.user.id;
    
    if (isNaN(applicationId)) {
      return res.status(400).json({ error: 'Invalid application ID' });
    }
    
    // Get messages for application (using task_id null and application_id)
    const query = `
      SELECT 
        m.*,
        s.username as sender_name,
        r.username as recipient_name
      FROM messages m
      LEFT JOIN users s ON m.sender_id = s.id
      LEFT JOIN users r ON m.recipient_id = r.id
      WHERE m.application_id = $1
        AND (m.sender_id = $2 OR m.recipient_id = $2)
      ORDER BY m.created_at ASC
    `;
    
    const db = require('./config/database');
    const result = await db.query(query, [applicationId, userId]);
    
    // Format messages to include proper sender/recipient objects
    const formattedMessages = result.rows.map(msg => ({
      ...msg,
      sender: {
        id: msg.sender_id,
        username: msg.sender_name
      },
      recipient: {
        id: msg.recipient_id,
        username: msg.recipient_name
      }
    }));
    
    res.json(formattedMessages);
  } catch (error) {
    logger.error('Error getting application messages:', error);
    res.status(500).json({ error: 'Failed to get application messages' });
  }
});

// Send application message
app.post('/api/v1/applications/:applicationId/messages', authenticateHTTP, async (req, res) => {
  try {
    const applicationId = parseInt(req.params.applicationId);
    const { content } = req.body;
    const senderId = req.user.id;
    
    if (isNaN(applicationId)) {
      return res.status(400).json({ error: 'Invalid application ID' });
    }
    
    if (!content || !content.trim()) {
      return res.status(400).json({ error: 'Message content is required' });
    }
    
    // Get application details to find recipient
    const appQuery = 'SELECT task_id, applicant_id FROM applications WHERE id = $1';
    const taskQuery = 'SELECT creator_id FROM tasks WHERE id = $1';
    
    const db = require('./config/database');
    const appResult = await db.query(appQuery, [applicationId]);
    
    if (appResult.rows.length === 0) {
      return res.status(404).json({ error: 'Application not found' });
    }
    
    const application = appResult.rows[0];
    const taskResult = await db.query(taskQuery, [application.task_id]);
    
    if (taskResult.rows.length === 0) {
      return res.status(404).json({ error: 'Associated task not found' });
    }
    
    const task = taskResult.rows[0];
    
    // Determine recipient based on sender
    let recipientId;
    if (senderId === application.applicant_id) {
      recipientId = task.creator_id;
    } else if (senderId === task.creator_id) {
      recipientId = application.applicant_id;
    } else {
      return res.status(403).json({ error: 'Unauthorized to send message in this application' });
    }
    
    const message = await messageService.sendMessage({
      applicationId,
      senderId,
      recipientId,
      content: content.trim()
    });
    
    // Get user information to format the response
    const senderQuery = 'SELECT username FROM users WHERE id = $1';
    const recipientQuery = 'SELECT username FROM users WHERE id = $1';
    
    const senderResult = await db.query(senderQuery, [senderId]);
    const recipientResult = await db.query(recipientQuery, [recipientId]);
    
    // Format message with sender/recipient info
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: senderResult.rows[0]?.username || 'Unknown'
      },
      recipient: {
        id: recipientId,
        username: recipientResult.rows[0]?.username || 'Unknown'
      }
    };
    
    res.status(201).json(formattedMessage);
  } catch (error) {
    logger.error('Error sending application message:', error);
    res.status(500).json({ error: 'Failed to send application message' });
  }
});

// Store message endpoints

// Get store messages for a specific item
app.get('/api/v1/store-messages/:itemId', authenticateHTTP, async (req, res) => {
  try {
    const itemId = parseInt(req.params.itemId);
    const userId = req.user.id;
    
    if (isNaN(itemId)) {
      return res.status(400).json({ error: 'Invalid item ID' });
    }
    
    const messages = await messageService.getStoreMessages(itemId, userId);
    const messageCount = await messageService.getUserStoreMessageCount(itemId, userId);
    const messageLimit = await messageService.getMessageLimit(itemId, userId);
    const bookingStatus = await messageService.getBookingStatus(itemId, userId);
    
    // Format messages to include proper sender/recipient objects
    const formattedMessages = messages.map(msg => ({
      ...msg,
      sender: {
        id: msg.sender_id,
        username: msg.sender_username
      },
      recipient: {
        id: msg.recipient_id,
        username: msg.recipient_username
      }
    }));
    
    res.json({
      messages: formattedMessages,
      messageCount,
      messageLimit,
      bookingStatus
    });
  } catch (error) {
    logger.error('Error getting store messages:', error);
    res.status(500).json({ error: 'Failed to get store messages' });
  }
});

// Get message limits and status for a store item
app.get('/api/v1/store-messages/:itemId/limits', authenticateHTTP, async (req, res) => {
  try {
    const itemId = parseInt(req.params.itemId);
    const userId = req.user.id;
    
    if (isNaN(itemId)) {
      return res.status(400).json({ error: 'Invalid item ID' });
    }
    
    const messageCount = await messageService.getUserStoreMessageCount(itemId, userId);
    const messageLimit = await messageService.getMessageLimit(itemId, userId);
    const bookingStatus = await messageService.getBookingStatus(itemId, userId);
    
    res.json({
      messageCount,
      messageLimit,
      bookingStatus,
      canSendMore: messageCount < messageLimit
    });
  } catch (error) {
    logger.error('Error getting message limits:', error);
    res.status(500).json({ error: 'Failed to get message limits' });
  }
});

// Send a store message
app.post('/api/v1/store-messages', authenticateHTTP, async (req, res) => {
  try {
    const { store_item_id, recipient_id, content } = req.body;
    const senderId = req.user.id;
    
    // Validate input
    if (!store_item_id || !recipient_id || !content || !content.trim()) {
      return res.status(400).json({ error: 'Missing required fields' });
    }
    
    if (content.length > 500) {
      return res.status(400).json({ error: 'Message too long (max 500 characters)' });
    }
    
    // Check dynamic message limit based on booking status
    const messageCount = await messageService.getUserStoreMessageCount(store_item_id, senderId);
    const messageLimit = await messageService.getMessageLimit(store_item_id, senderId);
    
    if (messageCount >= messageLimit) {
      const limitMessage = messageLimit === 10 ? 
        'Message limit reached (10 messages per item)' : 
        'Message limit reached (3 messages per item). Limit increases to 10 when booking is approved.';
      return res.status(403).json({ 
        error: limitMessage,
        limit: messageLimit,
        count: messageCount
      });
    }
    
    const message = await messageService.createStoreMessage({
      store_item_id: parseInt(store_item_id),
      sender_id: senderId,
      recipient_id: parseInt(recipient_id),
      content: content.trim()
    });
    
    // Get user information to format the response
    const senderQuery = 'SELECT username FROM users WHERE id = $1';
    const recipientQuery = 'SELECT username FROM users WHERE id = $1';
    
    const db = require('./config/database');
    const senderResult = await db.query(senderQuery, [senderId]);
    const recipientResult = await db.query(recipientQuery, [parseInt(recipient_id)]);
    
    // Format message with sender/recipient info
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: senderResult.rows[0]?.username || 'Unknown'
      },
      recipient: {
        id: parseInt(recipient_id),
        username: recipientResult.rows[0]?.username || 'Unknown'
      }
    };
    
    // Emit WebSocket events to notify connected clients
    if (io) {
      // Emit to the item room (for users currently viewing the item)
      io.to(`item:${store_item_id}`).emit('message:new', formattedMessage);
      
      // Emit notification to recipient's personal room
      io.to(`user:${recipient_id}`).emit('message:notification', {
        message: formattedMessage,
        conversationId: store_item_id
      });
      
      // Refresh conversations list for both sender and recipient
      io.to(`user:${senderId}`).emit('conversations:refresh');
      io.to(`user:${recipient_id}`).emit('conversations:refresh');
    }
    
    res.status(201).json(formattedMessage);
  } catch (error) {
    logger.error('Error creating store message:', error);
    res.status(500).json({ error: 'Failed to send message' });
  }
});

// Get user's conversations list
app.get('/api/v1/conversations', authenticateHTTP, async (req, res) => {
  try {
    const conversations = await messageService.getUserConversations(req.user.id);
    
    // Format conversations to match WebSocket format
    const formattedConversations = conversations.map(conv => ({
      task_id: conv.task_id,
      application_id: conv.application_id,
      item_id: conv.store_item_id,
      task_title: conv.task_title,
      task_description: conv.task_description,
      task_status: conv.task_status,
      item_title: conv.item_title,
      last_message: conv.content || '',
      last_message_time: conv.created_at,
      other_user_id: conv.other_user_id,
      other_user_name: conv.other_user_name,
      unread_count: conv.unread_count || 0,
      conversation_type: conv.task_id ? 'task' : 
                        conv.application_id ? 'application' : 
                        conv.store_item_id ? 'store' : 'unknown'
    }));
    
    res.json(formattedConversations);
  } catch (error) {
    logger.error('Error getting conversations:', error);
    res.status(500).json({ error: 'Failed to get conversations' });
  }
});

// Get user's last seen
app.get('/api/v1/users/:id/last-seen', authenticateHTTP, async (req, res) => {
  try {
    const lastSeen = await messageService.getUserLastSeen(req.params.id);
    res.json({ userId: req.params.id, lastSeen });
  } catch (error) {
    logger.error('Error getting last seen:', error);
    res.status(500).json({ error: 'Failed to get last seen' });
  }
});

// Error handling middleware
app.use((err, req, res, next) => {
  logger.error('Unhandled error:', err);
  res.status(500).json({ error: 'Internal server error' });
});

// Run migrations and start server
const runMigration = require('./scripts/migrate');

const startServer = async () => {
  const PORT = process.env.PORT || 8082;

  try {
    // Run database migrations
    await runMigration();

    httpServer.listen(PORT, () => {
      logger.info(`Chat WebSocket service running on port ${PORT}`);
      
      // Initialize scheduler
      schedulerService.init();
    });
  } catch (error) {
    logger.error('Failed to start server:', error);
    process.exit(1);
  }
};

startServer();

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM received, shutting down gracefully');
  
  schedulerService.stop();
  
  httpServer.close(() => {
    logger.info('HTTP server closed');
    process.exit(0);
  });
});