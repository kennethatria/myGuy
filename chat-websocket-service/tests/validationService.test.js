jest.mock('../src/utils/logger', () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
}));

jest.mock('axios');

const axios = require('axios');
const validationService = require('../src/services/validationService');

describe('ValidationService', () => {
  beforeEach(() => {
    validationService.clearCache();
    jest.clearAllMocks();
  });

  describe('validateUser', () => {
    it('returns true for a 200 response', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      const result = await validationService.validateUser(1, 'token123');
      expect(result).toBe(true);
    });

    it('returns false for a 404 response', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateUser(2, 'token123');
      expect(result).toBe(false);
    });

    it('returns true (fail open) for a non-404 error response', async () => {
      const error = new Error('Server Error');
      error.response = { status: 500 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateUser(3, 'token123');
      expect(result).toBe(true);
    });

    it('returns true (fail open) for a network error with no response', async () => {
      const error = new Error('Network error');
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateUser(4, 'token123');
      expect(result).toBe(true);
    });

    it('returns cached value on repeated calls (only one API request)', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      await validationService.validateUser(10, 'token');
      await validationService.validateUser(10, 'token');
      expect(axios.get).toHaveBeenCalledTimes(1);
    });

    it('caches false for 404 response', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      await validationService.validateUser(11, 'token');
      const result = await validationService.validateUser(11, 'token');
      expect(result).toBe(false);
      expect(axios.get).toHaveBeenCalledTimes(1);
    });
  });

  describe('validateTask', () => {
    it('returns true for a 200 response', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      const result = await validationService.validateTask(10, 'token');
      expect(result).toBe(true);
    });

    it('returns false for a 404 response', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateTask(11, 'token');
      expect(result).toBe(false);
    });

    it('returns true (fail open) for other errors', async () => {
      const error = new Error('Server error');
      error.response = { status: 503 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateTask(12, 'token');
      expect(result).toBe(true);
    });

    it('uses cache to avoid repeated API calls', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      await validationService.validateTask(20, 'token');
      await validationService.validateTask(20, 'token');
      expect(axios.get).toHaveBeenCalledTimes(1);
    });

    it('returns true (fail open) for network error without response', async () => {
      const error = new Error('Connection refused');
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateTask(13, 'token');
      expect(result).toBe(true);
    });
  });

  describe('validateApplication', () => {
    it('always returns true (implementation skipped — managed by main API)', async () => {
      const result = await validationService.validateApplication(1, 'token');
      expect(result).toBe(true);
    });

    it('returns true for any application ID', async () => {
      const result = await validationService.validateApplication(999, 'token');
      expect(result).toBe(true);
    });
  });

  describe('resolveApplicationRecipient', () => {
    const participants = { data: { application_id: 7, task_id: 3, applicant_id: 2, task_owner_id: 1 } };

    it('returns the other participant for the applicant and the owner', async () => {
      axios.get.mockResolvedValue(participants);

      expect(await validationService.resolveApplicationRecipient(7, 2, 'tok')).toBe(1);
      expect(await validationService.resolveApplicationRecipient(7, 1, 'tok')).toBe(2);
      expect(axios.get).toHaveBeenCalledWith(
        expect.stringContaining('/applications/7/participants'),
        expect.objectContaining({ headers: { Authorization: 'Bearer tok' } })
      );
      expect(axios.get).toHaveBeenCalledTimes(1); // second call served from cache
    });

    it('rejects someone who is not a participant, even with cached data', async () => {
      axios.get.mockResolvedValue(participants);
      await validationService.resolveApplicationRecipient(7, 2, 'tok');

      expect(await validationService.resolveApplicationRecipient(7, 99, 'tok')).toBeNull();
    });

    it('treats not found / forbidden as not allowed', async () => {
      axios.get.mockRejectedValue({ response: { status: 404 } });
      expect(await validationService.resolveApplicationRecipient(8, 2, 'tok')).toBeNull();
    });

    it('fails closed when the main API is unreachable', async () => {
      axios.get.mockRejectedValue(new Error('ECONNREFUSED'));
      await expect(validationService.resolveApplicationRecipient(9, 2, 'tok')).rejects.toThrow('ECONNREFUSED');
    });
  });

  describe('validateStoreItem', () => {
    it('returns true for a 200 response', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      const result = await validationService.validateStoreItem(20, 'token');
      expect(result).toBe(true);
    });

    it('returns false for a 404 response', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateStoreItem(21, 'token');
      expect(result).toBe(false);
    });

    it('returns true (fail open) for other errors', async () => {
      const error = new Error('Service unavailable');
      error.response = { status: 503 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateStoreItem(22, 'token');
      expect(result).toBe(true);
    });

    it('returns true (fail open) for network error without response', async () => {
      const error = new Error('Timeout');
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateStoreItem(23, 'token');
      expect(result).toBe(true);
    });

    it('uses cache to avoid repeated API calls', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      await validationService.validateStoreItem(30, 'token');
      await validationService.validateStoreItem(30, 'token');
      expect(axios.get).toHaveBeenCalledTimes(1);
    });
  });

  describe('validateMessageContext', () => {
    it('returns valid=true when context is empty', async () => {
      const result = await validationService.validateMessageContext({}, 'token');
      expect(result.valid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    it('returns valid=false when task does not exist', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateMessageContext({ taskId: 99 }, 'token');
      expect(result.valid).toBe(false);
      expect(result.errors[0]).toContain('Task 99 not found');
    });

    it('returns valid=true when task exists', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      const result = await validationService.validateMessageContext({ taskId: 5 }, 'token');
      expect(result.valid).toBe(true);
    });

    it('returns valid=true when application is provided (always valid)', async () => {
      const result = await validationService.validateMessageContext({ applicationId: 5 }, 'token');
      expect(result.valid).toBe(true);
    });

    it('returns valid=false when store item does not exist', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateMessageContext({ storeItemId: 88 }, 'token');
      expect(result.valid).toBe(false);
      expect(result.errors[0]).toContain('Store item 88 not found');
    });

    it('accumulates multiple errors for multiple invalid contexts', async () => {
      const error = new Error('Not Found');
      error.response = { status: 404 };
      axios.get.mockRejectedValue(error);
      const result = await validationService.validateMessageContext({ taskId: 1, storeItemId: 2 }, 'token');
      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
    });
  });

  describe('clearCache', () => {
    it('removes all cached entries', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      await validationService.validateUser(50, 'token');
      validationService.clearCache();
      await validationService.validateUser(50, 'token');
      expect(axios.get).toHaveBeenCalledTimes(2);
    });
  });

  describe('getCacheStats', () => {
    it('returns size and entries array', async () => {
      axios.get.mockResolvedValue({ status: 200 });
      await validationService.validateUser(100, 'token');
      const stats = validationService.getCacheStats();
      expect(typeof stats.size).toBe('number');
      expect(stats.size).toBeGreaterThan(0);
      expect(Array.isArray(stats.entries)).toBe(true);
    });

    it('returns empty stats after clearCache', () => {
      const stats = validationService.getCacheStats();
      expect(stats.size).toBe(0);
      expect(stats.entries).toEqual([]);
    });
  });
});
