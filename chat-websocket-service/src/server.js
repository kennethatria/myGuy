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
    
    res.status(201).json(formattedMessage);
  } catch (error) {
    logger.error('Error creating store message:', error);
    res.status(500).json({ error: 'Failed to send message' });
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

// Start server
const PORT = process.env.PORT || 8082;

httpServer.listen(PORT, () => {
  logger.info(`Chat WebSocket service running on port ${PORT}`);
  
  // Initialize scheduler
  schedulerService.init();
});

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM received, shutting down gracefully');
  
  schedulerService.stop();
  
  httpServer.close(() => {
    logger.info('HTTP server closed');
    process.exit(0);
  });
});