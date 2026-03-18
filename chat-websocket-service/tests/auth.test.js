jest.mock('../src/utils/logger', () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
}));

const jwt = require('jsonwebtoken');
const { verifyToken, authenticateSocket, authenticateHTTP } = require('../src/middleware/auth');

const TEST_SECRET = 'your-secret-key'; // matches default in auth.js

describe('auth middleware', () => {
  describe('verifyToken', () => {
    it('returns decoded payload for a valid token', () => {
      const payload = { user_id: 1, email: 'test@test.com', name: 'Test User' };
      const token = jwt.sign(payload, TEST_SECRET);
      const result = verifyToken(token);
      expect(result.user_id).toBe(1);
      expect(result.email).toBe('test@test.com');
    });

    it('returns null for an invalid token string', () => {
      const result = verifyToken('this.is.not.valid');
      expect(result).toBeNull();
    });

    it('returns null for an expired token', () => {
      const token = jwt.sign({ user_id: 1 }, TEST_SECRET, { expiresIn: -1 });
      const result = verifyToken(token);
      expect(result).toBeNull();
    });

    it('returns null for a token signed with wrong secret', () => {
      const token = jwt.sign({ user_id: 1 }, 'wrong-secret');
      const result = verifyToken(token);
      expect(result).toBeNull();
    });
  });

  describe('authenticateSocket', () => {
    const makeSocket = (authToken, headerToken) => ({
      handshake: {
        auth: { token: authToken },
        headers: { authorization: headerToken }
      },
      userId: null,
      userEmail: null,
      userName: null,
      id: 'socket-abc'
    });

    it('authenticates with token in handshake.auth.token', async () => {
      const token = jwt.sign({ user_id: 2, email: 'a@b.com', name: 'Alice' }, TEST_SECRET);
      const socket = makeSocket(token, undefined);
      const next = jest.fn();

      await authenticateSocket(socket, next);

      expect(next).toHaveBeenCalledWith();
      expect(socket.userId).toBe(2);
      expect(socket.userEmail).toBe('a@b.com');
      expect(socket.userName).toBe('Alice');
    });

    it('authenticates with Bearer token in authorization header', async () => {
      const token = jwt.sign({ user_id: 3, email: 'b@c.com', name: 'Bob' }, TEST_SECRET);
      const socket = makeSocket(undefined, `Bearer ${token}`);
      const next = jest.fn();

      await authenticateSocket(socket, next);

      expect(next).toHaveBeenCalledWith();
      expect(socket.userId).toBe(3);
    });

    it('strips Bearer prefix from header token', async () => {
      const token = jwt.sign({ user_id: 4, email: 'c@d.com', name: 'Carol' }, TEST_SECRET);
      const socket = makeSocket(undefined, `Bearer ${token}`);
      const next = jest.fn();

      await authenticateSocket(socket, next);

      expect(next).toHaveBeenCalledWith();
      expect(socket.userId).toBe(4);
    });

    it('calls next with error when no token provided', async () => {
      const socket = makeSocket(undefined, undefined);
      const next = jest.fn();

      await authenticateSocket(socket, next);

      expect(next).toHaveBeenCalledWith(expect.any(Error));
      expect(next.mock.calls[0][0].message).toBe('Authentication token required');
    });

    it('calls next with error for invalid token', async () => {
      const socket = makeSocket('bad.token.value', undefined);
      const next = jest.fn();

      await authenticateSocket(socket, next);

      expect(next).toHaveBeenCalledWith(expect.any(Error));
      expect(next.mock.calls[0][0].message).toBe('Invalid authentication token');
    });
  });

  describe('authenticateHTTP', () => {
    const makeReq = (authHeader) => ({
      headers: { authorization: authHeader }
    });

    const makeRes = () => {
      const res = {};
      res.status = jest.fn().mockReturnValue(res);
      res.json = jest.fn().mockReturnValue(res);
      return res;
    };

    it('authenticates valid Bearer token and attaches user to req', () => {
      const token = jwt.sign({ user_id: 5, email: 'e@f.com', name: 'Eve' }, TEST_SECRET);
      const req = makeReq(`Bearer ${token}`);
      const res = makeRes();
      const next = jest.fn();

      authenticateHTTP(req, res, next);

      expect(next).toHaveBeenCalled();
      expect(req.user.id).toBe(5);
      expect(req.user.email).toBe('e@f.com');
      expect(req.user.name).toBe('Eve');
    });

    it('returns 401 when authorization header is missing', () => {
      const req = makeReq(undefined);
      const res = makeRes();
      const next = jest.fn();

      authenticateHTTP(req, res, next);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith(expect.objectContaining({ error: expect.any(String) }));
      expect(next).not.toHaveBeenCalled();
    });

    it('returns 401 when header does not start with Bearer', () => {
      const req = makeReq('Basic dXNlcjpwYXNz');
      const res = makeRes();
      const next = jest.fn();

      authenticateHTTP(req, res, next);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(next).not.toHaveBeenCalled();
    });

    it('returns 401 for an invalid token value', () => {
      const req = makeReq('Bearer not.a.real.token');
      const res = makeRes();
      const next = jest.fn();

      authenticateHTTP(req, res, next);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(next).not.toHaveBeenCalled();
    });

    it('returns 401 for an expired token', () => {
      const token = jwt.sign({ user_id: 1 }, TEST_SECRET, { expiresIn: -1 });
      const req = makeReq(`Bearer ${token}`);
      const res = makeRes();
      const next = jest.fn();

      authenticateHTTP(req, res, next);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(next).not.toHaveBeenCalled();
    });
  });
});
