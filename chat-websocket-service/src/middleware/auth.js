const jwt = require('jsonwebtoken');
const logger = require('../utils/logger');

const JWT_SECRET = process.env.JWT_SECRET || 'your-secret-key';

/**
 * Verify JWT token and extract user information
 * @param {string} token - JWT token
 * @returns {object|null} - Decoded token payload or null if invalid
 */
const verifyToken = (token) => {
  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    return decoded;
  } catch (error) {
    logger.error('JWT verification failed:', error.message);
    return null;
  }
};

/**
 * Socket.IO authentication middleware
 * @param {object} socket - Socket.IO socket instance
 * @param {function} next - Next middleware function
 */
const authenticateSocket = async (socket, next) => {
  try {
    const token = socket.handshake.auth.token || socket.handshake.headers.authorization?.replace('Bearer ', '');
    
    if (!token) {
      return next(new Error('Authentication token required'));
    }

    const decoded = verifyToken(token);
    if (!decoded) {
      return next(new Error('Invalid authentication token'));
    }

    // Attach user info to socket
    socket.userId = decoded.user_id;
    socket.userEmail = decoded.email;
    socket.userName = decoded.name;
    
    logger.info('Socket authenticated', { userId: socket.userId, socketId: socket.id });
    
    next();
  } catch (error) {
    logger.error('Socket authentication error:', error);
    next(new Error('Authentication failed'));
  }
};

/**
 * Express middleware for HTTP endpoints
 * @param {object} req - Express request object
 * @param {object} res - Express response object
 * @param {function} next - Next middleware function
 */
const authenticateHTTP = (req, res, next) => {
  try {
    const authHeader = req.headers.authorization;
    
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({ error: 'Authorization header required' });
    }

    const token = authHeader.replace('Bearer ', '');
    const decoded = verifyToken(token);
    
    if (!decoded) {
      return res.status(401).json({ error: 'Invalid token' });
    }

    req.user = {
      id: decoded.user_id,
      email: decoded.email,
      name: decoded.name
    };
    
    next();
  } catch (error) {
    logger.error('HTTP authentication error:', error);
    res.status(401).json({ error: 'Authentication failed' });
  }
};

module.exports = {
  verifyToken,
  authenticateSocket,
  authenticateHTTP
};