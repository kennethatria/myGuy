const messageService = require('../services/messageService');
const logger = require('../utils/logger');

class SocketHandlers {
  constructor(io) {
    this.io = io;
    this.userSockets = new Map(); // userId -> Set of socket IDs
  }

  /**
   * Handle new socket connection
   */
  handleConnection(socket) {
    logger.info('New socket connection', { 
      socketId: socket.id, 
      userId: socket.userId 
    });

    // Track user's sockets
    this.addUserSocket(socket.userId, socket.id);

    // Join user's personal room
    socket.join(`user:${socket.userId}`);

    // Update user's online status
    this.updateUserPresence(socket.userId, true);

    // Set up event handlers
    this.setupEventHandlers(socket);

    // Handle disconnect
    socket.on('disconnect', () => {
      this.handleDisconnect(socket);
    });
  }

  /**
   * Set up all socket event handlers
   */
  setupEventHandlers(socket) {
    // Join conversation room
    socket.on('join:conversation', (data) => this.handleJoinConversation(socket, data));
    
    // Leave conversation room
    socket.on('leave:conversation', (data) => this.handleLeaveConversation(socket, data));
    
    // Send message
    socket.on('message:send', (data) => this.handleSendMessage(socket, data));
    
    // Edit message
    socket.on('message:edit', (data) => this.handleEditMessage(socket, data));
    
    // Delete message
    socket.on('message:delete', (data) => this.handleDeleteMessage(socket, data));
    
    // Mark as read
    socket.on('message:read', (data) => this.handleMarkAsRead(socket, data));
    
    // Mark conversation as read
    socket.on('conversation:read', (data) => this.handleMarkConversationAsRead(socket, data));
    
    // Typing indicators
    socket.on('typing:start', (data) => this.handleTypingStart(socket, data));
    socket.on('typing:stop', (data) => this.handleTypingStop(socket, data));
    
    // Get conversations list
    socket.on('conversations:list', () => this.handleGetConversations(socket));
    
    // Get messages for conversation
    socket.on('messages:get', (data) => this.handleGetMessages(socket, data));
    
    // Get user's last seen
    socket.on('user:lastseen', (data) => this.handleGetLastSeen(socket, data));
  }

  /**
   * Handle joining a conversation room
   */
  async handleJoinConversation(socket, { taskId, applicationId }) {
    try {
      const roomName = taskId ? `task:${taskId}` : `application:${applicationId}`;
      socket.join(roomName);
      
      logger.info('User joined conversation', { 
        userId: socket.userId, 
        room: roomName 
      });

      // Update user activity
      await messageService.updateUserActivity(socket.userId, taskId || applicationId);

      socket.emit('conversation:joined', { room: roomName });
    } catch (error) {
      logger.error('Error joining conversation:', error);
      socket.emit('error', { message: 'Failed to join conversation' });
    }
  }

  /**
   * Handle leaving a conversation room
   */
  handleLeaveConversation(socket, { taskId, applicationId }) {
    const roomName = taskId ? `task:${taskId}` : `application:${applicationId}`;
    socket.leave(roomName);
    
    logger.info('User left conversation', { 
      userId: socket.userId, 
      room: roomName 
    });

    socket.emit('conversation:left', { room: roomName });
  }

  /**
   * Handle sending a message
   */
  async handleSendMessage(socket, data) {
    try {
      const { taskId, applicationId, recipientId, content } = data;

      // Validate input
      if (!content || (!taskId && !applicationId) || !recipientId) {
        return socket.emit('error', { message: 'Invalid message data' });
      }

      // Send message
      const message = await messageService.sendMessage({
        taskId,
        applicationId,
        senderId: socket.userId,
        recipientId,
        content
      });

      // Get user information for formatting
      const senderInfo = await this.getUserInfo(socket.userId);
      const recipientInfo = await this.getUserInfo(recipientId);

      // Format message with sender/recipient info
      const formattedMessage = {
        ...message,
        sender: {
          id: socket.userId,
          username: senderInfo?.username || 'Unknown'
        },
        recipient: {
          id: recipientId,
          username: recipientInfo?.username || 'Unknown'
        }
      };

      // Emit to sender
      socket.emit('message:sent', formattedMessage);

      // Emit to conversation room
      const roomName = taskId ? `task:${taskId}` : `application:${applicationId}`;
      socket.to(roomName).emit('message:new', formattedMessage);

      // Emit to recipient's personal room (for notifications)
      this.io.to(`user:${recipientId}`).emit('message:notification', {
        message: formattedMessage,
        conversationId: taskId || applicationId
      });

      // Send warning if content was filtered
      if (message.hasRemovedContent) {
        socket.emit('message:filtered', {
          messageId: message.id,
          warning: 'Links and contact information have been removed from your message'
        });
      }

    } catch (error) {
      logger.error('Error sending message:', error);
      socket.emit('error', { message: 'Failed to send message' });
    }
  }

  /**
   * Handle editing a message
   */
  async handleEditMessage(socket, { messageId, content }) {
    try {
      const message = await messageService.editMessage(messageId, socket.userId, content);

      // Get the conversation room
      const roomName = message.task_id ? `task:${message.task_id}` : `application:${message.application_id}`;

      // Emit to all in conversation
      this.io.to(roomName).emit('message:edited', message);

      // Send warning if content was filtered
      if (message.hasRemovedContent) {
        socket.emit('message:filtered', {
          messageId: message.id,
          warning: 'Links and contact information have been removed from your message'
        });
      }

    } catch (error) {
      logger.error('Error editing message:', error);
      socket.emit('error', { message: error.message || 'Failed to edit message' });
    }
  }

  /**
   * Handle deleting a message
   */
  async handleDeleteMessage(socket, { messageId }) {
    try {
      const message = await messageService.deleteMessage(messageId, socket.userId);

      // Get the conversation room
      const roomName = message.task_id ? `task:${message.task_id}` : `application:${message.application_id}`;

      // Emit to all in conversation
      this.io.to(roomName).emit('message:deleted', { messageId });

    } catch (error) {
      logger.error('Error deleting message:', error);
      socket.emit('error', { message: error.message || 'Failed to delete message' });
    }
  }

  /**
   * Handle marking message as read
   */
  async handleMarkAsRead(socket, { messageId }) {
    try {
      const message = await messageService.markAsRead(messageId, socket.userId);
      
      if (message) {
        // Notify sender that message was read
        this.io.to(`user:${message.sender_id}`).emit('message:read', {
          messageId: message.id,
          readAt: message.read_at
        });
      }

    } catch (error) {
      logger.error('Error marking message as read:', error);
    }
  }

  /**
   * Handle marking entire conversation as read
   */
  async handleMarkConversationAsRead(socket, { taskId }) {
    try {
      const readMessages = await messageService.markConversationAsRead(taskId, socket.userId);
      
      // Emit read receipts for all messages
      readMessages.forEach(msg => {
        this.io.to(`task:${taskId}`).emit('message:read', {
          messageId: msg.id,
          readAt: new Date()
        });
      });

      socket.emit('conversation:marked-read', { taskId, count: readMessages.length });

    } catch (error) {
      logger.error('Error marking conversation as read:', error);
      socket.emit('error', { message: 'Failed to mark conversation as read' });
    }
  }

  /**
   * Handle typing start
   */
  handleTypingStart(socket, { taskId, applicationId }) {
    const roomName = taskId ? `task:${taskId}` : `application:${applicationId}`;
    
    socket.to(roomName).emit('user:typing', {
      userId: socket.userId,
      userName: socket.userName,
      conversationId: taskId || applicationId
    });
  }

  /**
   * Handle typing stop
   */
  handleTypingStop(socket, { taskId, applicationId }) {
    const roomName = taskId ? `task:${taskId}` : `application:${applicationId}`;
    
    socket.to(roomName).emit('user:stopped-typing', {
      userId: socket.userId,
      conversationId: taskId || applicationId
    });
  }

  /**
   * Handle getting conversations list
   */
  async handleGetConversations(socket) {
    try {
      const conversations = await messageService.getUserConversations(socket.userId);
      
      // Format conversations to ensure proper timestamp and structure
      const formattedConversations = conversations.map(conv => ({
        task_id: conv.task_id,
        application_id: conv.application_id,
        task_title: conv.task_title,
        task_description: conv.task_description,
        task_status: conv.task_status,
        last_message: conv.content || '',
        last_message_time: conv.created_at, // Use created_at as the timestamp
        other_user_id: conv.other_user_id,
        other_user_name: conv.other_user_name,
        unread_count: conv.unread_count || 0,
        conversation_type: conv.task_id ? 'task' : 'application'
      }));
      
      socket.emit('conversations:list', formattedConversations);
    } catch (error) {
      logger.error('Error getting conversations:', error);
      socket.emit('error', { message: 'Failed to get conversations' });
    }
  }

  /**
   * Handle getting messages for a conversation
   */
  async handleGetMessages(socket, { taskId, limit = 5, offset = 0 }) {
    try {
      const messages = await messageService.getMessages(taskId, socket.userId, limit, offset);
      
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
      
      socket.emit('messages:list', { taskId, messages: formattedMessages, offset });
    } catch (error) {
      logger.error('Error getting messages:', error);
      socket.emit('error', { message: 'Failed to get messages' });
    }
  }

  /**
   * Handle getting user's last seen
   */
  async handleGetLastSeen(socket, { userId }) {
    try {
      const lastSeen = await messageService.getUserLastSeen(userId);
      socket.emit('user:lastseen', { userId, lastSeen });
    } catch (error) {
      logger.error('Error getting last seen:', error);
      socket.emit('error', { message: 'Failed to get last seen' });
    }
  }

  /**
   * Get user information by ID
   */
  async getUserInfo(userId) {
    try {
      const db = require('../config/database');
      const query = 'SELECT id, username FROM users WHERE id = $1';
      const result = await db.query(query, [userId]);
      return result.rows[0] || null;
    } catch (error) {
      logger.error('Error getting user info:', error);
      return null;
    }
  }

  /**
   * Handle socket disconnect
   */
  handleDisconnect(socket) {
    logger.info('Socket disconnected', { 
      socketId: socket.id, 
      userId: socket.userId 
    });

    this.removeUserSocket(socket.userId, socket.id);

    // If user has no more sockets, update presence
    if (!this.userSockets.has(socket.userId) || this.userSockets.get(socket.userId).size === 0) {
      this.updateUserPresence(socket.userId, false);
    }
  }

  /**
   * Track user's socket connections
   */
  addUserSocket(userId, socketId) {
    if (!this.userSockets.has(userId)) {
      this.userSockets.set(userId, new Set());
    }
    this.userSockets.get(userId).add(socketId);
  }

  /**
   * Remove user's socket
   */
  removeUserSocket(userId, socketId) {
    if (this.userSockets.has(userId)) {
      this.userSockets.get(userId).delete(socketId);
      if (this.userSockets.get(userId).size === 0) {
        this.userSockets.delete(userId);
      }
    }
  }

  /**
   * Update user's presence
   */
  async updateUserPresence(userId, isOnline) {
    try {
      await messageService.updateUserActivity(userId);
      
      // Broadcast presence update
      this.io.emit('user:presence', {
        userId,
        isOnline,
        lastSeen: new Date()
      });
    } catch (error) {
      logger.error('Error updating user presence:', error);
    }
  }
}

module.exports = SocketHandlers;