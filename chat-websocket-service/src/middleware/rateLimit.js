const { rateLimit, ipKeyGenerator } = require('express-rate-limit');
const { verifyToken } = require('./auth');

/**
 * Who a request counts against: the signed-in user when the request carries
 * a valid session token, otherwise the client IP. Keying by user keeps one
 * person from exhausting a limit shared by everyone behind the same proxy
 * (nginx forwards from one address).
 */
function requestKey(req) {
  const header = req.headers.authorization || '';
  if (header.startsWith('Bearer ')) {
    const claims = verifyToken(header.slice(7));
    if (claims && claims.user_id) return `user:${claims.user_id}`;
  }
  return `ip:${ipKeyGenerator(req.ip || '')}`;
}

/**
 * Rate limit for the HTTP API. Generous enough for the app's own polling and
 * page loads; it exists to stop one client hammering the service. Internal
 * service-to-service calls (/internal/*) are authenticated by the shared key
 * and come in bursts (one per affected applicant), so they are not limited.
 */
const apiRateLimit = rateLimit({
  windowMs: 60 * 1000,
  limit: 300,
  standardHeaders: 'draft-7',
  legacyHeaders: false,
  keyGenerator: requestKey,
  skip: (req) => req.path.startsWith('/internal/'),
  message: { error: 'Too many requests, please slow down.' }
});

module.exports = { apiRateLimit, requestKey };
