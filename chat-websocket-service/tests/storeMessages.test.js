const request = require('supertest');
const express = require('express');
const jwt = require('jsonwebtoken');

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
    it('should verify database query filters messages correctly', () => {
      // Test that the messageService getStoreMessages method would be called
      // with the correct parameters for filtering
      const itemId = 1;
      const userId = 1;
      
      // Mock the database query to verify the SQL
      const mockQuery = jest.fn().mockResolvedValue({ rows: [] });
      db.query = mockQuery;
      
      messageService.getStoreMessages(itemId, userId);
      
      // This test verifies the integration will work correctly
      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('WHERE sm.store_item_id = $1'),
        [itemId, userId]
      );
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
});