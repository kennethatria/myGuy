require('./tracing'); // must be first — instruments http/express before they load
require('dotenv').config();
const express = require('express');
const { createServer } = require('http');
const { Server } = require('socket.io');
const cors = require('cors');
const helmet = require('helmet');

const logger = require('./utils/logger');
const { authenticateSocket, authenticateHTTP } = require('./middleware/auth');
const { apiRateLimit } = require('./middleware/rateLimit');
const SocketHandlers = require('./handlers/socketHandlers');
const validationService = require('./services/validationService');
const { healthHandler } = require('./api/health');
const messageService = require('./services/messageService');
const schedulerService = require('./services/schedulerService');
const { configureRedisAdapter, getRedisHealth } = require('./config/redis');

// Initialize Express app
const app = express();
const httpServer = createServer(app);

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());
app.use('/api/v1', apiRateLimit);

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

// Health check: status only, details to the log
app.get('/health', healthHandler({ db: require('./config/database'), getRedisHealth, logger }));



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

    // Format message with sender/recipient IDs only
    // Frontend will handle fetching usernames from main API
    // (Avoids cross-database query - users table is in my_guy, not my_guy_chat)
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: 'User' // Frontend should replace with actual username
      },
      recipient: {
        id: parseInt(recipient_id),
        username: 'User' // Frontend should replace with actual username
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
    const { content, recipient_id } = req.body;
    const senderId = req.user.id;

    if (isNaN(applicationId)) {
      return res.status(400).json({ error: 'Invalid application ID' });
    }

    if (!content || !content.trim()) {
      return res.status(400).json({ error: 'Message content is required' });
    }

    // Only the task owner and the applicant may chat about an application,
    // and only with each other; the main API owns that data.
    let recipientId;
    try {
      recipientId = await validationService.resolveApplicationRecipient(
        applicationId, senderId, req.headers.authorization.replace('Bearer ', '')
      );
    } catch (error) {
      logger.error('Application authorization check failed:', error.message);
      return res.status(503).json({ error: 'Could not verify the application, try again' });
    }
    if (!recipientId || (recipient_id && parseInt(recipient_id) !== recipientId)) {
      return res.status(403).json({ error: 'You cannot message about this application' });
    }

    const message = await messageService.sendMessage({
      applicationId,
      senderId,
      recipientId,
      content: content.trim()
    });

    // Format message with sender/recipient IDs only
    // Frontend will handle fetching usernames from main API
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: 'User' // Frontend should replace with actual username
      },
      recipient: {
        id: recipientId,
        username: 'User' // Frontend should replace with actual username
      }
    };

    // Emit WebSocket events to notify connected clients
    if (io) {
      // Deliver only to the two participants (all their tabs); conversation
      // rooms are joinable by anyone, so private messages never go there.
      io.to(`user:${senderId}`).to(`user:${recipientId}`).emit('message:new', formattedMessage);

      // Refresh conversations list for both sender and recipient
      io.to(`user:${senderId}`).emit('conversations:refresh');
      io.to(`user:${recipientId}`).emit('conversations:refresh');
    }

    // Return message with IDs only - frontend fetches usernames separately
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

    // Format message with sender/recipient IDs only
    // Frontend will handle fetching usernames from main API
    // (Avoids cross-database query - users table is in my_guy, not my_guy_chat)
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: 'User' // Frontend should replace with actual username
      },
      recipient: {
        id: parseInt(recipient_id),
        username: 'User' // Frontend should replace with actual username
      }
    };
    
    // Emit WebSocket events to notify connected clients
    if (io) {
      // Deliver only to the two participants (all their tabs). The item room
      // holds every buyer of the item, so it must not carry private messages.
      io.to(`user:${senderId}`).to(`user:${recipient_id}`).emit('message:new', formattedMessage);
      
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
    const otherUserId = parseInt(req.params.id);
    if (isNaN(otherUserId)) {
      return res.status(400).json({ error: 'Invalid user ID' });
    }
    // Only people who have chatted with this user may see when they were last on.
    if (!(await messageService.haveConversed(req.user.id, otherUserId))) {
      return res.status(404).json({ error: 'Not found' });
    }
    const lastSeen = await messageService.getUserLastSeen(otherUserId);
    res.json({ userId: otherUserId, lastSeen });
  } catch (error) {
    logger.error('Error getting last seen:', error);
    res.status(500).json({ error: 'Failed to get last seen' });
  }
});

// Booking notification routes
const bookingNotifications = require('./api/bookingNotifications');
app.use('/api/v1', bookingNotifications);

// Make io instance available to routes
app.set('io', io);

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

    // Configure Redis adapter for horizontal scaling (if Redis is configured)
    const redisConfigured = await configureRedisAdapter(io);
    if (redisConfigured) {
      logger.info('Multi-instance mode enabled via Redis adapter');
    } else {
      logger.info('Single-instance mode (set REDIS_URL or REDIS_HOST to enable multi-instance)');
    }

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