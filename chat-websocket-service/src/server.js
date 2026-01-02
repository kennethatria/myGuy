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

// Health check (enhanced with migration status and service info)
app.get('/health', async (req, res) => {
  const health = {
    status: 'ok',
    service: 'chat-websocket-service',
    version: '1.0.0',
    uptime: Math.floor(process.uptime()),
    timestamp: new Date().toISOString(),
    database: 'unknown',
    migrations: {
      status: 'unknown',
      count: 0,
      lastRun: null
    },
    environment: {
      node_env: process.env.NODE_ENV || 'production',
      port: process.env.PORT || 8082
    }
  };

  try {
    // Check database connection
    const db = require('./config/database');
    await db.query('SELECT NOW()');
    health.database = 'connected';

    // Check migration status
    const migrationResult = await db.query(`
      SELECT COUNT(*) as count, MAX(applied_at) as last_run
      FROM schema_migrations
    `);

    if (migrationResult.rows && migrationResult.rows.length > 0) {
      health.migrations = {
        status: 'applied',
        count: parseInt(migrationResult.rows[0].count),
        lastRun: migrationResult.rows[0].last_run
      };
    }

    // Get table counts for additional health info
    const tablesResult = await db.query(`
      SELECT
        (SELECT COUNT(*) FROM messages) as message_count,
        (SELECT COUNT(*) FROM user_activity) as active_users
    `);

    if (tablesResult.rows && tablesResult.rows.length > 0) {
      health.stats = {
        messages: parseInt(tablesResult.rows[0].message_count),
        activeUsers: parseInt(tablesResult.rows[0].active_users)
      };
    }

  } catch (error) {
    logger.error('Health check failed:', error);
    health.status = 'degraded';
    health.database = 'error';
    health.error = error.message;
  }

  // Set HTTP status based on health
  const httpStatus = health.status === 'ok' ? 200 : 503;
  res.status(httpStatus).json(health);
});

// Test endpoint without auth
app.get('/api/v1/test', (req, res) => {
  res.json({ message: 'Test endpoint works', timestamp: new Date() });
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

    // Message limits removed - unlimited messaging allowed
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

    // Message limits removed - unlimited messaging
    res.json({
      messageCount,
      messageLimit: null,
      unlimited: true,
      canSendMore: true,
      remaining: null
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

    // Get messages for application - no cross-database JOINs
    // Frontend should fetch user details via Main API
    const query = `
      SELECT m.*
      FROM messages m
      WHERE m.application_id = $1
        AND (m.sender_id = $2 OR m.recipient_id = $2)
      ORDER BY m.created_at ASC
    `;

    const db = require('./config/database');
    const result = await db.query(query, [applicationId, userId]);

    // Return messages with IDs only - frontend fetches usernames separately
    res.json(result.rows);
  } catch (error) {
    logger.error('Error getting application messages:', error);
    res.status(500).json({ error: 'Failed to get application messages' });
  }
});

// Send application message
app.post('/api/v1/applications/:applicationId/messages', authenticateHTTP, async (req, res) => {
  try {
    const applicationId = parseInt(req.params.applicationId);
    const { content, recipientId } = req.body;
    const senderId = req.user.id;

    if (isNaN(applicationId)) {
      return res.status(400).json({ error: 'Invalid application ID' });
    }

    if (!content || !content.trim()) {
      return res.status(400).json({ error: 'Message content is required' });
    }

    if (!recipientId) {
      return res.status(400).json({ error: 'Recipient ID is required' });
    }

    // applications and tasks tables exist in my_guy database (not accessible)
    // Frontend must provide recipientId based on application context
    // TODO: Use ValidationService to validate application exists via Main API

    const message = await messageService.sendMessage({
      applicationId,
      senderId,
      recipientId,
      content: content.trim()
    });

    // Return message with IDs only - frontend fetches usernames separately
    res.status(201).json(message);
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
    const bookingStatus = await messageService.getBookingStatus(itemId, userId);

    // Return messages with IDs only - frontend fetches usernames via Main API
    // Message limits removed - unlimited messaging
    res.json({
      messages,
      messageCount,
      messageLimit: null,
      unlimited: true,
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
    const bookingStatus = await messageService.getBookingStatus(itemId, userId);

    // Message limits removed - unlimited messaging
    res.json({
      messageCount,
      messageLimit: null,
      unlimited: true,
      bookingStatus,
      canSendMore: true
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

    // Message limits removed - unlimited messaging allowed
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
const runMigration = require('./scripts/migrate-simple');

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