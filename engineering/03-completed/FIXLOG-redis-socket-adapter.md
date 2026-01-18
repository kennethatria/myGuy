# Fix Log: Redis Adapter for Socket.IO

**Date:** January 18, 2026
**Priority:** P2 (Recommended Before Launch)
**Status:** ✅ COMPLETED

## Problem

The chat service could not scale beyond a single instance because Socket.IO's default in-memory adapter does not share state across multiple server instances. This means:

- WebSocket connections are tied to specific instances
- Messages sent on one instance aren't visible to clients connected to other instances
- No horizontal scaling capability for the chat service
- Single point of failure for real-time messaging

## Solution

Implemented the Redis adapter for Socket.IO, enabling horizontal scaling by sharing Socket.IO state (rooms, connections, messages) across multiple instances via Redis pub/sub.

### Key Features

1. **Optional Configuration**: Redis is optional - falls back to in-memory adapter if not configured
2. **Graceful Degradation**: If Redis connection fails, service continues in single-instance mode
3. **Health Monitoring**: Redis status exposed via `/health` endpoint
4. **Docker Integration**: Redis service added to docker-compose.yml

---

## Implementation Details

### New Files

**`chat-websocket-service/src/config/redis.js`**

Redis configuration module providing:
- `isRedisConfigured()` - Check if Redis is configured via environment
- `createRedisClients()` - Create pub/sub clients for the adapter
- `configureRedisAdapter(io)` - Configure Socket.IO with Redis adapter
- `getRedisHealth()` - Get Redis health status for monitoring

### Modified Files

| File | Changes |
|------|---------|
| `chat-websocket-service/package.json` | Added `@socket.io/redis-adapter` and `redis` dependencies |
| `chat-websocket-service/src/server.js` | Import Redis config, configure adapter on startup, add Redis to health check |
| `chat-websocket-service/.env.example` | Added Redis configuration options |
| `docker-compose.yml` | Added Redis service, updated chat-websocket-service to use Redis |

---

## Environment Configuration

### Option 1: Full URL (Recommended for Cloud)

```env
REDIS_URL=redis://username:password@hostname:6379/0
```

Supports cloud Redis providers:
- Redis Cloud
- Upstash
- AWS ElastiCache
- Azure Cache for Redis
- DigitalOcean Managed Redis

### Option 2: Individual Settings (Local Development)

```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your-password
REDIS_DB=0
```

### No Configuration (Single Instance Mode)

If neither `REDIS_URL` nor `REDIS_HOST` is set, the service runs in single-instance mode using Socket.IO's default in-memory adapter.

---

## Docker Compose Changes

Added Redis service:

```yaml
redis:
  image: redis:7-alpine
  ports:
    - "6379:6379"
  volumes:
    - redis-data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 5s
    retries: 5
  command: redis-server --appendonly yes
```

Chat service now depends on Redis and includes `REDIS_URL` environment variable.

---

## Health Check Response

The `/health` endpoint now includes Redis status:

```json
{
  "status": "ok",
  "service": "chat-websocket-service",
  "redis": {
    "configured": true,
    "connected": true,
    "mode": "redis (multi-instance)"
  }
}
```

Possible Redis modes:
- `in-memory (single instance)` - Redis not configured
- `redis (multi-instance)` - Redis connected and working
- `redis (disconnected)` - Redis configured but connection failed

---

## Scaling the Chat Service

### With Docker Compose

```bash
# Scale to 3 instances
docker-compose up --scale chat-websocket-service=3

# Note: You'll need a load balancer (nginx, traefik) to distribute traffic
```

### Load Balancer Configuration (nginx example)

```nginx
upstream chat_servers {
    ip_hash;  # Sticky sessions for WebSocket
    server chat-websocket-service-1:8082;
    server chat-websocket-service-2:8082;
    server chat-websocket-service-3:8082;
}

server {
    location /socket.io/ {
        proxy_pass http://chat_servers;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### With Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chat-websocket-service
spec:
  replicas: 3  # Multiple instances
  template:
    spec:
      containers:
      - name: chat
        env:
        - name: REDIS_URL
          value: "redis://redis-service:6379/0"
```

---

## Testing

### Verify Redis Connection

```bash
# Check health endpoint
curl http://localhost:8082/health | jq '.redis'

# Expected output when Redis is connected:
# {
#   "configured": true,
#   "connected": true,
#   "mode": "redis (multi-instance)"
# }
```

### Test Multi-Instance Messaging

1. Start multiple chat service instances
2. Connect clients to different instances
3. Send message from client on instance 1
4. Verify message received by client on instance 2

---

## Dependencies Added

| Package | Version | Purpose |
|---------|---------|---------|
| `@socket.io/redis-adapter` | ^8.3.0 | Socket.IO Redis adapter |
| `redis` | ^4.6.12 | Redis client for Node.js |

---

## Startup Logs

**With Redis configured:**
```
[INFO] Connecting to Redis for Socket.IO adapter...
[INFO] Redis pub client connected
[INFO] Redis sub client connected
[INFO] Redis clients connected successfully - multi-instance mode enabled
[INFO] Socket.IO Redis adapter configured successfully
[INFO] Multi-instance mode enabled via Redis adapter
[INFO] Chat WebSocket service running on port 8082
```

**Without Redis:**
```
[INFO] Redis not configured - using in-memory Socket.IO adapter (single instance mode)
[INFO] Single-instance mode (set REDIS_URL or REDIS_HOST to enable multi-instance)
[INFO] Chat WebSocket service running on port 8082
```

---

## Related Documentation

- `engineering/01-proposed/ROADMAP-mvp-prioritization.md` - P2 scalability item
- `engineering/02-reference/ARCH-chat-service-architecture.md` - Chat service architecture
- `chat-websocket-service/.env.example` - Environment configuration
- `docker-compose.yml` - Docker deployment configuration

---

**Document Version:** 1.0
**Created:** January 18, 2026
