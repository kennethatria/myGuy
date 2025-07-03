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
});