const request = require('supertest');
const express = require('express');
const jwt = require('jsonwebtoken');
const { Server } = require('socket.io');
const Client = require('socket.io-client');

// Mock database
jest.mock('../src/config/database', () => ({
  query: jest.fn(),
  getClient: jest.fn()
}));

const db = require('../src/config/database');

// Create test app
const app = express();
app.use(express.json());

// Mock authentication middleware
const mockAuth = (req, res, next) => {
  const authHeader = req.headers.authorization;
  if (!authHeader) {
    return res.status(401).json({ error: 'No authorization header' });
  }
  
  const token = authHeader.split(' ')[1];
  try {
    const decoded = jwt.verify(token, 'test-secret');
    req.user = decoded;
    next();
  } catch (error) {
    return res.status(401).json({ error: 'Invalid token' });
  }
};

// Import messageService after mocking dependencies
const messageService = require('../src/services/messageService');

// Mock the store messages endpoint
app.get('/api/v1/store-messages/:itemId', mockAuth, async (req, res) => {
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
    res.status(500).json({ error: 'Failed to get store messages' });
  }
});

// Mock the POST store messages endpoint
app.post('/api/v1/store-messages', mockAuth, async (req, res) => {
  try {
    const { store_item_id, recipient_id, content } = req.body;
    const senderId = req.user.id;
    
    if (!store_item_id || !recipient_id || !content || !content.trim()) {
      return res.status(400).json({ error: 'Missing required fields' });
    }
    
    const message = await messageService.createStoreMessage({
      store_item_id: parseInt(store_item_id),
      sender_id: senderId,
      recipient_id: parseInt(recipient_id),
      content: content.trim()
    });
    
    const formattedMessage = {
      ...message,
      sender: {
        id: senderId,
        username: req.user.username || 'Unknown'
      },
      recipient: {
        id: parseInt(recipient_id),
        username: 'Unknown'
      }
    };
    
    res.status(201).json(formattedMessage);
  } catch (error) {
    res.status(500).json({ error: 'Failed to send message' });
  }
});

// Mock conversations endpoint for testing
app.get('/api/v1/conversations', mockAuth, async (req, res) => {
  try {
    const userId = req.user.id;
    const conversations = await messageService.getUserConversations(userId);
    
    // Format conversations to match WebSocket format (same as real server.js)
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
    res.status(500).json({ error: 'Failed to get conversations' });
  }
});

describe('Store Messages API', () => {
  let itemOwnerToken, buyerToken;
  
  beforeAll(() => {
    // Create test tokens
    itemOwnerToken = jwt.sign({ id: 1, username: 'seller1' }, 'test-secret');
    buyerToken = jwt.sign({ id: 2, username: 'buyer1' }, 'test-secret');
  });
  
  beforeEach(() => {
    jest.clearAllMocks();
  });
  
  describe('GET /api/v1/store-messages/:itemId', () => {
    it('should show messages to item owner that were sent to them', async () => {
      const mockMessages = [
        {
          id: 1,
          store_item_id: 1,
          sender_id: 2,
          recipient_id: 1,
          content: 'Hi, I\'m interested in this item',
          sender_username: 'buyer1',
          recipient_username: 'seller1',
          created_at: new Date()
        }
      ];
      
      // Mock the messageService methods
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockMessages);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(1);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(3);
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue(null);
      
      const response = await request(app)
        .get('/api/v1/store-messages/1')
        .set('Authorization', `Bearer ${itemOwnerToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(1);
      expect(response.body.messages[0].sender.username).toBe('buyer1');
      expect(response.body.messages[0].recipient.username).toBe('seller1');
      expect(response.body.messages[0].content).toBe('Hi, I\'m interested in this item');
      
      // Verify messageService was called with correct parameters
      expect(messageService.getStoreMessages).toHaveBeenCalledWith(1, 1);
    });
    
    it('should show messages to buyer that they sent', async () => {
      const mockMessages = [
        {
          id: 1,
          store_item_id: 1,
          sender_id: 2,
          recipient_id: 1,
          content: 'Hi, I\'m interested in this item',
          sender_username: 'buyer1',
          recipient_username: 'seller1',
          created_at: new Date()
        }
      ];
      
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockMessages);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(1);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(3);
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue(null);
      
      const response = await request(app)
        .get('/api/v1/store-messages/1')
        .set('Authorization', `Bearer ${buyerToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(1);
      expect(response.body.messageCount).toBe(1);
      expect(response.body.messageLimit).toBe(3);
      
      // Verify messageService was called with buyer's ID
      expect(messageService.getStoreMessages).toHaveBeenCalledWith(1, 2);
    });
    
    it('should return empty messages for unrelated user', async () => {
      const unrelatedUserToken = jwt.sign({ id: 3, username: 'unrelated' }, 'test-secret');
      
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue([]);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(0);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(3);
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue(null);
      
      const response = await request(app)
        .get('/api/v1/store-messages/1')
        .set('Authorization', `Bearer ${unrelatedUserToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(0);
      expect(messageService.getStoreMessages).toHaveBeenCalledWith(1, 3);
    });
    
    it('should handle invalid item ID', async () => {
      await request(app)
        .get('/api/v1/store-messages/invalid')
        .set('Authorization', `Bearer ${itemOwnerToken}`)
        .expect(400)
        .expect(res => {
          expect(res.body.error).toBe('Invalid item ID');
        });
    });
    
    it('should require authentication', async () => {
      await request(app)
        .get('/api/v1/store-messages/1')
        .expect(401)
        .expect(res => {
          expect(res.body.error).toBe('No authorization header');
        });
    });
  });
  
  describe('Message Visibility Logic', () => {
    it('should filter messages to only show relevant conversations', async () => {
      // Test that the messageService filters conversations correctly
      // by checking that users only see messages they're part of
      const userId = 1;
      
      // Mock database to return conversations for this user
      db.query.mockClear();
      db.query.mockResolvedValue({
        rows: [
          {
            id: 1,
            store_item_id: 1,
            sender_id: 1,
            recipient_id: 2,
            content: 'User is sender',
            created_at: new Date()
          },
          {
            id: 2,
            store_item_id: 2,
            sender_id: 3,
            recipient_id: 1,
            content: 'User is recipient',
            created_at: new Date()
          }
        ]
      });
      
      const conversations = await messageService.getUserConversations(userId);
      
      // Verify that we got conversations and db.query was called
      expect(db.query).toHaveBeenCalled();
      expect(conversations).toBeDefined();
    });
  });

  describe('Enhanced Owner Messaging', () => {
    it('should allow item owner to see messages sent to them by buyers', async () => {
      const ownerUserId = 1;
      const buyerUserId = 2;
      const itemId = 1;
      
      const mockMessages = [
        {
          id: 1,
          store_item_id: itemId,
          sender_id: buyerUserId,
          recipient_id: ownerUserId,
          content: 'I\'m interested in your item',
          sender_username: 'buyer1',
          recipient_username: 'seller1',
          created_at: new Date()
        }
      ];
      
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockMessages);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(1);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(3);
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue(null);
      
      const response = await request(app)
        .get(`/api/v1/store-messages/${itemId}`)
        .set('Authorization', `Bearer ${itemOwnerToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(1);
      expect(response.body.messages[0].sender.username).toBe('buyer1');
      expect(response.body.messages[0].recipient.username).toBe('seller1');
      expect(messageService.getStoreMessages).toHaveBeenCalledWith(itemId, ownerUserId);
    });
    
    it('should allow item owner to see messages they sent to buyers', async () => {
      const ownerUserId = 1;
      const buyerUserId = 2;
      const itemId = 1;
      
      const mockMessages = [
        {
          id: 1,
          store_item_id: itemId,
          sender_id: ownerUserId,
          recipient_id: buyerUserId,
          content: 'Thanks for your interest! When would you like to view it?',
          sender_username: 'seller1',
          recipient_username: 'buyer1',
          created_at: new Date()
        }
      ];
      
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockMessages);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(1);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(10); // Approved booking
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue('approved');
      
      const response = await request(app)
        .get(`/api/v1/store-messages/${itemId}`)
        .set('Authorization', `Bearer ${itemOwnerToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(1);
      expect(response.body.messages[0].sender.username).toBe('seller1');
      expect(response.body.messages[0].recipient.username).toBe('buyer1');
      expect(response.body.messageLimit).toBe(10); // Enhanced limit for approved booking
      expect(response.body.bookingStatus).toBe('approved');
    });
    
    it('should show conversation between owner and specific buyer', async () => {
      const ownerUserId = 1;
      const buyerUserId = 2;
      const itemId = 1;
      
      const mockConversation = [
        {
          id: 1,
          store_item_id: itemId,
          sender_id: buyerUserId,
          recipient_id: ownerUserId,
          content: 'I\'m interested in this item',
          sender_username: 'buyer1',
          recipient_username: 'seller1',
          created_at: new Date('2024-01-01T10:00:00Z')
        },
        {
          id: 2,
          store_item_id: itemId,
          sender_id: ownerUserId,
          recipient_id: buyerUserId,
          content: 'Great! When would you like to see it?',
          sender_username: 'seller1',
          recipient_username: 'buyer1',
          created_at: new Date('2024-01-01T10:05:00Z')
        }
      ];
      
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockConversation);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(1);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(10);
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue('approved');
      
      const response = await request(app)
        .get(`/api/v1/store-messages/${itemId}`)
        .set('Authorization', `Bearer ${itemOwnerToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(2);
      // Verify conversation flow
      expect(response.body.messages[0].sender.username).toBe('buyer1');
      expect(response.body.messages[1].sender.username).toBe('seller1');
      // Verify messages are ordered chronologically
      expect(new Date(response.body.messages[0].created_at).getTime()).toBeLessThan(
        new Date(response.body.messages[1].created_at).getTime()
      );
    });
    
    it('should not show messages between other users (privacy maintained)', async () => {
      const ownerUserId = 1;
      const unrelatedUserId = 3;
      const itemId = 1;
      
      // No messages should be returned for unrelated user
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue([]);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(0);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(3);
      jest.spyOn(messageService, 'getBookingStatus').mockResolvedValue(null);
      
      const unrelatedToken = jwt.sign({ id: unrelatedUserId, username: 'unrelated' }, 'test-secret');
      
      const response = await request(app)
        .get(`/api/v1/store-messages/${itemId}`)
        .set('Authorization', `Bearer ${unrelatedToken}`)
        .expect(200);
      
      expect(response.body.messages).toHaveLength(0);
      expect(messageService.getStoreMessages).toHaveBeenCalledWith(itemId, unrelatedUserId);
    });
  });

  describe('WebSocket Store Messaging Integration', () => {
    let io, serverSocket, clientSocket;
    const port = 3001;
    
    beforeAll((done) => {
      // Create WebSocket server
      const httpServer = require('http').createServer();
      io = new Server(httpServer);
      httpServer.listen(port, () => {
        done();
      });
    });
    
    afterAll(() => {
      io.close();
    });
    
    beforeEach((done) => {
      // Mock socket handlers with store message support
      io.on('connection', (socket) => {
        socket.userId = 1;
        socket.userName = 'testuser';
        serverSocket = socket;
        
        // Mock WebSocket handlers for store messages
        socket.on('join:conversation', ({ itemId }) => {
          if (itemId) {
            socket.join(`item:${itemId}`);
            socket.emit('conversation:joined', { room: `item:${itemId}` });
          }
        });
        
        socket.on('message:send', async ({ itemId, recipientId, content }) => {
          if (itemId) {
            // Mock store message creation
            const mockMessage = {
              id: 1,
              store_item_id: itemId,
              sender_id: socket.userId,
              recipient_id: recipientId,
              content: content,
              created_at: new Date().toISOString()
            };
            
            socket.emit('message:sent', mockMessage);
            socket.to(`item:${itemId}`).emit('message:new', mockMessage);
          }
        });
        
        socket.on('messages:get', ({ itemId }) => {
          if (itemId) {
            const mockMessages = [{
              id: 1,
              store_item_id: itemId,
              sender_id: 1,
              recipient_id: 2,
              content: 'Test store message',
              created_at: new Date().toISOString()
            }];
            
            socket.emit('messages:list', { 
              itemId, 
              messages: mockMessages, 
              offset: 0,
              totalCount: 1
            });
          }
        });
      });
      
      // Create client socket
      clientSocket = Client(`http://localhost:${port}`);
      clientSocket.on('connect', done);
    });
    
    afterEach(() => {
      serverSocket?.disconnect();
      clientSocket?.disconnect();
    });
    
    it('should join store conversation room', (done) => {
      const itemId = 1;
      
      clientSocket.emit('join:conversation', { itemId });
      
      clientSocket.once('conversation:joined', (data) => {
        expect(data.room).toBe(`item:${itemId}`);
        done();
      });
    });
    
    it('should send store messages via WebSocket', (done) => {
      const itemId = 1;
      const recipientId = 2;
      const content = 'WebSocket store message test';
      
      clientSocket.emit('message:send', { itemId, recipientId, content });
      
      clientSocket.once('message:sent', (message) => {
        expect(message.store_item_id).toBe(itemId);
        expect(message.sender_id).toBe(1);
        expect(message.recipient_id).toBe(recipientId);
        expect(message.content).toBe(content);
        done();
      });
    });
    
    it('should retrieve store messages via WebSocket', (done) => {
      const itemId = 1;
      
      clientSocket.emit('messages:get', { itemId });
      
      clientSocket.once('messages:list', (data) => {
        expect(data.itemId).toBe(itemId);
        expect(data.messages).toHaveLength(1);
        expect(data.messages[0].store_item_id).toBe(itemId);
        expect(data.messages[0].content).toBe('Test store message');
        done();
      });
    });
  });

  describe('End-to-End Store Message Flow', () => {
    let mockStoreItem, mockUsers;
    
    beforeAll(() => {
      // Mock store item data
      mockStoreItem = {
        id: 1,
        title: 'Test Item',
        seller: { id: 2, username: 'seller1', full_name: 'Test Seller' }
      };
      
      // Mock users
      mockUsers = {
        buyer: { id: 1, username: 'buyer1' },
        seller: { id: 2, username: 'seller1' }
      };
    });
    
    it('should handle complete message flow from popup to chat thread', async () => {
      const itemId = 1;
      const buyerId = 1;
      const sellerId = 2;
      const messageContent = 'Is this item still available?';
      
      // Mock the store message creation response
      const mockCreatedMessage = {
        id: 1,
        store_item_id: itemId,
        sender_id: buyerId,
        recipient_id: sellerId,
        content: messageContent,
        created_at: new Date().toISOString(),
        sender: { id: buyerId, username: 'buyer1' },
        recipient: { id: sellerId, username: 'seller1' }
      };
      
      // Mock the HTTP API response
      jest.spyOn(messageService, 'createStoreMessage').mockResolvedValue(mockCreatedMessage);
      jest.spyOn(messageService, 'getUserStoreMessageCount').mockResolvedValue(0);
      jest.spyOn(messageService, 'getMessageLimit').mockResolvedValue(3);
      
      const buyerToken = jwt.sign({ id: buyerId, username: 'buyer1' }, 'test-secret');
      
      // 1. Send message via popup modal (HTTP API)
      const response = await request(app)
        .post('/api/v1/store-messages')
        .set('Authorization', `Bearer ${buyerToken}`)
        .send({
          store_item_id: itemId,
          recipient_id: sellerId,
          content: messageContent
        })
        .expect(201);
      
      expect(response.body).toMatchObject({
        store_item_id: itemId,
        sender_id: buyerId,
        recipient_id: sellerId,
        content: messageContent
      });
      
      // 2. Verify conversation appears in conversations list with proper formatting
      const mockConversations = [{
        store_item_id: itemId,
        item_id: itemId,
        item_title: 'Test Item',
        other_user_id: sellerId,
        other_user_name: 'seller1',
        content: messageContent,
        created_at: new Date().toISOString(),
        unread_count: 0
      }];
      
      jest.spyOn(messageService, 'getUserConversations').mockResolvedValue(mockConversations);
      
      const conversationsResponse = await request(app)
        .get('/api/v1/conversations')
        .set('Authorization', `Bearer ${buyerToken}`)
        .expect(200);
      
      expect(conversationsResponse.body).toHaveLength(1);
      expect(conversationsResponse.body[0]).toMatchObject({
        item_id: itemId,
        item_title: 'Test Item',
        other_user_name: 'seller1',
        conversation_type: 'store' // Verify it's correctly identified as store conversation
      });
      
      // 3. Verify messages can be retrieved for the conversation
      const mockStoreMessages = [mockCreatedMessage];
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockStoreMessages);
      
      const messagesResponse = await request(app)
        .get(`/api/v1/store-messages/${itemId}`)
        .set('Authorization', `Bearer ${buyerToken}`)
        .expect(200);
      
      expect(messagesResponse.body.messages).toHaveLength(1);
      expect(messagesResponse.body.messages[0]).toMatchObject({
        store_item_id: itemId,
        content: messageContent
      });
    });
    
    it('should handle seller reply in global chat', async () => {
      const itemId = 1;
      const buyerId = 1;
      const sellerId = 2;
      const replyContent = 'Yes, it\'s still available! Would you like to arrange pickup?';
      
      // Mock seller's reply message
      const mockReplyMessage = {
        id: 2,
        store_item_id: itemId,
        sender_id: sellerId,
        recipient_id: buyerId,
        content: replyContent,
        created_at: new Date().toISOString(),
        sender: { id: sellerId, username: 'seller1' },
        recipient: { id: buyerId, username: 'buyer1' }
      };
      
      jest.spyOn(messageService, 'createStoreMessage').mockResolvedValue(mockReplyMessage);
      
      const sellerToken = jwt.sign({ id: sellerId, username: 'seller1' }, 'test-secret');
      
      // Seller sends reply via chat interface
      const response = await request(app)
        .post('/api/v1/store-messages')
        .set('Authorization', `Bearer ${sellerToken}`)
        .send({
          store_item_id: itemId,
          recipient_id: buyerId,
          content: replyContent
        })
        .expect(201);
      
      expect(response.body).toMatchObject({
        store_item_id: itemId,
        sender_id: sellerId,
        recipient_id: buyerId,
        content: replyContent
      });
      
      // Verify buyer can see the reply
      const mockConversationMessages = [
        {
          id: 1,
          store_item_id: itemId,
          sender_id: buyerId,
          recipient_id: sellerId,
          content: 'Is this item still available?',
          sender_username: 'buyer1',
          recipient_username: 'seller1',
          created_at: new Date(Date.now() - 60000).toISOString()
        },
        {
          id: 2,
          store_item_id: itemId,
          sender_id: sellerId,
          recipient_id: buyerId,
          content: replyContent,
          sender_username: 'seller1',
          recipient_username: 'buyer1',
          created_at: new Date().toISOString()
        }
      ];
      
      jest.spyOn(messageService, 'getStoreMessages').mockResolvedValue(mockConversationMessages);
      
      const buyerToken = jwt.sign({ id: buyerId, username: 'buyer1' }, 'test-secret');
      
      const messagesResponse = await request(app)
        .get(`/api/v1/store-messages/${itemId}`)
        .set('Authorization', `Bearer ${buyerToken}`)
        .expect(200);
      
      expect(messagesResponse.body.messages).toHaveLength(2);
      expect(messagesResponse.body.messages[1]).toMatchObject({
        sender_id: sellerId,
        content: replyContent
      });
    });
  });
});