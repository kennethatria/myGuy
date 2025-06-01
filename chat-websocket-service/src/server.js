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