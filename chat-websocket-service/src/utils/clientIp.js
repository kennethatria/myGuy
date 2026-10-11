const net = require('net');

// Addresses nginx reaches this service from (it listens on 127.0.0.1)
const LOOPBACK = new Set(['127.0.0.1', '::1', '::ffff:127.0.0.1']);

/**
 * The visitor's IP. nginx sets X-Real-IP from the NodeBalancer's PROXY
 * header (templates/nginx-https.conf.j2), so it's trusted only on requests
 * that came from nginx; anything else (X-Forwarded-For, which nginx passes
 * on from the client) could be chosen by the client. The same rule as the
 * backend's and store-service's X-Real-IP.
 */
function clientIp(req) {
  const peer = req.socket?.remoteAddress || req.ip || '';
  const real = String(req.headers?.['x-real-ip'] || '').trim();
  return LOOPBACK.has(peer) && net.isIP(real) ? real : peer;
}

module.exports = { clientIp };
