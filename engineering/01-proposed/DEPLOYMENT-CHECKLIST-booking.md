# Deployment Checklist: Unified Booking & Messaging

**Feature:** Unified Booking & Messaging Flow
**Date Prepared:** January 4, 2026
**Status:** Backend Complete, Frontend In Progress

---

## Pre-Deployment Checklist

### 1. Environment Configuration

#### Generate Secure Keys
- [ ] Generate INTERNAL_API_KEY:
  ```bash
  openssl rand -hex 32
  ```
- [ ] Store key securely (password manager, secrets vault)
- [ ] Never commit to git

#### Chat Service Environment Variables
- [ ] Create/update `.env` from `.env.example`
- [ ] Set `INTERNAL_API_KEY=<generated-key>`
- [ ] Set `STORE_API_URL=http://localhost:8081/api/v1` (dev)
- [ ] Set `STORE_API_URL=https://store-api.yourdomain.com/api/v1` (prod)
- [ ] Verify `JWT_SECRET` matches other services
- [ ] Verify `DB_CONNECTION` is correct

#### Store Service Environment Variables
- [ ] Create/update `.env` from `.env.example`
- [ ] Set `INTERNAL_API_KEY=<same-key-as-chat-service>`
- [ ] Set `CHAT_API_URL=http://localhost:8082/api/v1` (dev)
- [ ] Set `CHAT_API_URL=https://chat-api.yourdomain.com/api/v1` (prod)
- [ ] Verify `JWT_SECRET` matches other services
- [ ] Verify `DB_CONNECTION` is correct

#### Docker Compose (if applicable)
- [ ] Update `docker-compose.yml` with environment variables
- [ ] Add `INTERNAL_API_KEY` to both services
- [ ] Add service URLs (use service names: `http://chat-websocket-service:8082`)

---

### 2. Database Migrations

#### Chat Service
- [ ] Backup database before migration:
  ```bash
  pg_dump -U postgres -d my_guy_chat > backup_chat_$(date +%Y%m%d).sql
  ```
- [ ] Run migration:
  ```bash
  cd chat-websocket-service
  npm run migrate
  ```
- [ ] Verify migration success:
  ```sql
  -- Check if metadata column exists
  SELECT column_name, data_type
  FROM information_schema.columns
  WHERE table_name = 'messages' AND column_name = 'metadata';

  -- Should return: metadata | jsonb
  ```
- [ ] Check indexes:
  ```sql
  SELECT indexname FROM pg_indexes WHERE tablename = 'messages';
  -- Should include: idx_messages_metadata_booking_id, idx_messages_metadata_item_id
  ```

#### Store Service
- [ ] Backup database before migration:
  ```bash
  pg_dump -U postgres -d my_guy_store > backup_store_$(date +%Y%m%d).sql
  ```
- [ ] Migration runs automatically on startup (GORM AutoMigrate)
- [ ] Verify columns added:
  ```sql
  SELECT column_name, data_type
  FROM information_schema.columns
  WHERE table_name = 'booking_requests'
  AND column_name IN ('chat_notified', 'notification_attempts', 'last_notification_attempt');

  -- Should return all 3 columns
  ```

---

### 3. Code Deployment

#### Backend Services
- [ ] Pull latest code:
  ```bash
  git pull origin main
  ```
- [ ] Install dependencies:
  ```bash
  # Chat service
  cd chat-websocket-service && npm install

  # Store service (if needed)
  cd store-service && go mod download
  ```
- [ ] Build services:
  ```bash
  docker-compose build
  ```

#### Frontend
- [ ] Pull latest code
- [ ] Install dependencies:
  ```bash
  cd frontend && npm install
  ```
- [ ] Run type check:
  ```bash
  npm run type-check
  ```
- [ ] Build for production:
  ```bash
  npm run build
  ```

---

### 4. Testing Before Deployment

#### Unit Tests
- [ ] Run chat service tests:
  ```bash
  cd chat-websocket-service
  npm test
  ```
- [ ] Run store service tests:
  ```bash
  cd store-service
  make test
  ```
- [ ] Run frontend tests:
  ```bash
  cd frontend
  npm run test:unit
  ```

#### Integration Testing (Local)
- [ ] Start all services:
  ```bash
  docker-compose up
  ```
- [ ] Test booking creation:
  ```bash
  curl -X POST http://localhost:8081/api/v1/items/1/booking-request \
    -H "Authorization: Bearer YOUR_JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"message": "Test booking"}'
  ```
- [ ] Check chat service logs for notification:
  ```
  Expected: "✅ Chat service notified successfully for booking X"
  ```
- [ ] Verify system message in database:
  ```sql
  SELECT id, message_type, content, metadata
  FROM messages
  WHERE message_type = 'booking_request'
  ORDER BY created_at DESC LIMIT 1;
  ```
- [ ] Test approve action:
  ```bash
  curl -X POST http://localhost:8082/api/v1/booking-action \
    -H "Authorization: Bearer SELLER_JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"bookingId": 1, "action": "approve"}'
  ```
- [ ] Verify booking status updated in store DB
- [ ] Verify approval message created in chat DB

#### Frontend Testing
- [ ] Login as buyer
- [ ] Book an item
- [ ] Verify redirect to /messages
- [ ] Verify booking request message appears
- [ ] Login as seller (different browser/incognito)
- [ ] Open messages
- [ ] Verify notification badge
- [ ] Click conversation
- [ ] Verify [Approve] [Decline] buttons visible
- [ ] Click [Approve]
- [ ] Verify both users see approval message
- [ ] Test [Decline] flow similarly

---

## Deployment Steps

### Development Environment

1. **Stop Running Services:**
   ```bash
   docker-compose down
   ```

2. **Pull Latest Code:**
   ```bash
   git pull origin main
   ```

3. **Update Environment Files:**
   ```bash
   cp chat-websocket-service/.env.example chat-websocket-service/.env
   cp store-service/.env.example store-service/.env
   # Edit .env files with correct values
   ```

4. **Run Migrations:**
   ```bash
   cd chat-websocket-service
   npm run migrate
   cd ..
   ```

5. **Start Services:**
   ```bash
   docker-compose up --build
   ```

6. **Verify Health:**
   ```bash
   # Chat service
   curl http://localhost:8082/health

   # Store service
   curl http://localhost:8081/api/v1/items
   ```

---

### Production Environment

#### Pre-Deployment

- [ ] **Schedule maintenance window** (30-60 minutes)
- [ ] **Notify users** of potential downtime
- [ ] **Backup all databases**
- [ ] **Tag release in git:**
  ```bash
  git tag -a v1.1.0-booking-messaging -m "Unified booking & messaging feature"
  git push origin v1.1.0-booking-messaging
  ```

#### Deployment Sequence

1. **Deploy Database Migrations (Zero-Downtime):**
   - [ ] Migrations are additive (safe to run before code deploy)
   - [ ] Run chat service migration:
     ```bash
     ssh production-chat-server
     cd /app/chat-websocket-service
     npm run migrate
     ```
   - [ ] Store service migration runs on startup (deploy store first)

2. **Deploy Store Service:**
   - [ ] Build new image:
     ```bash
     docker build -t store-service:v1.1.0 .
     ```
   - [ ] Push to registry
   - [ ] Update deployment config with new env vars
   - [ ] Rolling update:
     ```bash
     kubectl set image deployment/store-service store-service=store-service:v1.1.0
     # OR
     docker-compose up -d --no-deps --build store-service
     ```
   - [ ] Verify health endpoint
   - [ ] Check logs for errors

3. **Deploy Chat Service:**
   - [ ] Build new image:
     ```bash
     docker build -t chat-service:v1.1.0 .
     ```
   - [ ] Push to registry
   - [ ] Update deployment config with new env vars
   - [ ] Rolling update:
     ```bash
     kubectl set image deployment/chat-service chat-service=chat-service:v1.1.0
     # OR
     docker-compose up -d --no-deps --build chat-websocket-service
     ```
   - [ ] Verify health endpoint
   - [ ] Check logs for WebSocket connections

4. **Deploy Frontend:**
   - [ ] Build production bundle:
     ```bash
     cd frontend
     npm run build
     ```
   - [ ] Upload to CDN/static hosting
   - [ ] Update cache invalidation
   - [ ] Verify assets loading

5. **Smoke Tests:**
   - [ ] Create test booking (staging account)
   - [ ] Verify notification in chat
   - [ ] Test approve action
   - [ ] Check both databases for correct data
   - [ ] Verify WebSocket real-time updates

---

## Post-Deployment Checklist

### Immediate (Within 1 Hour)

- [ ] **Monitor Logs:**
  ```bash
  # Chat service
  docker-compose logs -f chat-websocket-service | grep -i "booking\|error"

  # Store service
  docker-compose logs -f store-service | grep -i "booking\|error"
  ```

- [ ] **Check Error Rates:**
  - [ ] Chat service /internal/booking-created endpoint (should be 200)
  - [ ] Chat service /booking-action endpoint
  - [ ] Store service booking creation

- [ ] **Verify First Production Booking:**
  - [ ] Watch for first real booking notification
  - [ ] Verify message created correctly
  - [ ] Check notification_attempts = 0, chat_notified = true

- [ ] **Database Health:**
  ```sql
  -- Chat service: Check for booking messages
  SELECT COUNT(*) FROM messages WHERE message_type IN ('booking_request', 'booking_approved', 'booking_declined');

  -- Store service: Check notification status
  SELECT
    COUNT(*) as total,
    SUM(CASE WHEN chat_notified THEN 1 ELSE 0 END) as notified,
    SUM(CASE WHEN NOT chat_notified THEN 1 ELSE 0 END) as failed
  FROM booking_requests
  WHERE created_at > NOW() - INTERVAL '1 hour';
  ```

### Within 24 Hours

- [ ] **Review Metrics:**
  - [ ] Booking notification success rate (target: >95%)
  - [ ] Average notification latency (target: <100ms)
  - [ ] Failed notification count (investigate if >0)
  - [ ] Booking approval rate via chat

- [ ] **Check Failed Notifications:**
  ```sql
  SELECT * FROM booking_requests
  WHERE chat_notified = false
  AND created_at > NOW() - INTERVAL '24 hours'
  ORDER BY created_at DESC;
  ```

- [ ] **User Feedback:**
  - [ ] Monitor support tickets for booking issues
  - [ ] Check user reports of "missing" notifications
  - [ ] Verify sellers are finding booking requests

- [ ] **Performance Impact:**
  - [ ] Check API response times (should not increase significantly)
  - [ ] Verify WebSocket connection stability
  - [ ] Monitor database query performance

### Within 1 Week

- [ ] **Analytics Review:**
  - [ ] Compare booking completion rate (before vs. after)
  - [ ] Measure time from booking to first seller response
  - [ ] Track approval/decline ratios

- [ ] **Failed Notification Investigation:**
  - [ ] Review logs for any INTERNAL_API_KEY errors
  - [ ] Check network connectivity issues between services
  - [ ] Implement retry job if needed

- [ ] **User Experience:**
  - [ ] Survey sellers about new booking flow
  - [ ] Gather feedback on chat integration
  - [ ] Identify UX improvements

---

## Rollback Plan

### If Critical Issues Occur

#### Immediate Rollback (< 5 minutes)

1. **Revert to Previous Version:**
   ```bash
   # Docker Compose
   docker-compose down
   git checkout <previous-tag>
   docker-compose up -d

   # Kubernetes
   kubectl rollout undo deployment/store-service
   kubectl rollout undo deployment/chat-service
   ```

2. **Verify Services Running:**
   ```bash
   curl http://localhost:8082/health
   curl http://localhost:8081/api/v1/items
   ```

3. **Notify Team & Users:**
   - [ ] Post in team Slack/communication channel
   - [ ] Update status page if applicable

#### Database Rollback (If Needed)

**⚠️ WARNING:** Only rollback DB if absolutely necessary (data loss risk)

```sql
-- Chat service: Remove metadata column (DESTRUCTIVE)
ALTER TABLE messages DROP COLUMN IF EXISTS metadata;

-- Store service: Remove notification columns (DESTRUCTIVE)
ALTER TABLE booking_requests
  DROP COLUMN IF EXISTS chat_notified,
  DROP COLUMN IF EXISTS notification_attempts,
  DROP COLUMN IF EXISTS last_notification_attempt;
```

**Better approach:** Keep new columns, disable feature via feature flag

---

## Monitoring & Alerts

### Key Metrics to Track

1. **Booking Notification Success Rate:**
   ```sql
   SELECT
     COUNT(*) as total_bookings,
     SUM(CASE WHEN chat_notified THEN 1 ELSE 0 END) as successful_notifications,
     ROUND(100.0 * SUM(CASE WHEN chat_notified THEN 1 ELSE 0 END) / COUNT(*), 2) as success_rate
   FROM booking_requests
   WHERE created_at > NOW() - INTERVAL '1 day';
   ```
   - **Alert if:** Success rate < 95%

2. **Notification Latency:**
   - Track time between `booking_requests.created_at` and first message creation
   - **Alert if:** Average > 500ms

3. **Failed Notifications with Retries:**
   ```sql
   SELECT COUNT(*) FROM booking_requests
   WHERE chat_notified = false
   AND notification_attempts >= 3;
   ```
   - **Alert if:** Count > 0

4. **API Error Rates:**
   - Monitor `/internal/booking-created` endpoint
   - Monitor `/booking-action` endpoint
   - **Alert if:** Error rate > 1%

### Recommended Alerts

- [ ] Set up Sentry/error tracking for both services
- [ ] Configure Slack/email alerts for failed notifications
- [ ] Set up uptime monitoring for new endpoints
- [ ] Create dashboard for booking flow metrics

---

## Security Checklist

- [ ] **INTERNAL_API_KEY is secure:**
  - [ ] At least 32 characters
  - [ ] Random generated (not a password)
  - [ ] Different from JWT_SECRET
  - [ ] Stored in secrets manager (not in git)
  - [ ] Same key in both services

- [ ] **HTTPS enabled in production:**
  - [ ] Inter-service communication uses HTTPS
  - [ ] Valid SSL certificates
  - [ ] No mixed content warnings

- [ ] **Authorization checks:**
  - [ ] Only seller can approve/decline bookings
  - [ ] JWT validation on all authenticated endpoints
  - [ ] No information leakage in error messages

- [ ] **Rate limiting:**
  - [ ] Consider rate limits on booking creation
  - [ ] Rate limit /booking-action to prevent abuse

---

## Documentation Updates

- [ ] Update API documentation with new endpoints
- [ ] Update README with new environment variables
- [ ] Add troubleshooting guide for common issues
- [ ] Document rollback procedures
- [ ] Create runbook for on-call engineers

---

## Success Criteria

✅ **Deployment is successful if:**
- All services start without errors
- Migrations complete successfully
- First booking creates chat notification
- Approve/decline actions work
- No increase in error rates
- Users can successfully complete booking flow
- WebSocket updates work in real-time

---

## Troubleshooting

### Issue: Chat notifications not being sent

**Symptoms:** Bookings created but no messages in chat

**Debug Steps:**
1. Check store service logs for "Notifying chat service" messages
2. Verify CHAT_API_URL is correct
3. Verify INTERNAL_API_KEY matches between services
4. Test endpoint manually:
   ```bash
   curl -X POST http://localhost:8082/api/v1/internal/booking-created \
     -H "X-Internal-API-Key: YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "bookingId": 1,
       "itemId": 2,
       "itemTitle": "Test",
       "buyerId": 3,
       "sellerId": 4
     }'
   ```

**Fix:**
- Ensure both services can communicate (network connectivity)
- Check firewall rules
- Verify environment variables loaded correctly

### Issue: Approve/decline buttons not working

**Symptoms:** Clicking buttons shows error or nothing happens

**Debug Steps:**
1. Check browser console for errors
2. Verify JWT token is valid
3. Check chat service logs for /booking-action requests
4. Verify STORE_API_URL is correct in chat service

**Fix:**
- Clear browser cache and reload
- Re-login to get fresh JWT token
- Verify API endpoints are accessible

### Issue: Database migration failed

**Symptoms:** Services won't start, migration errors in logs

**Fix:**
1. Check database connectivity
2. Verify database user has CREATE permission
3. Manually run SQL from migration file
4. Check for conflicting column names

---

## Contact & Escalation

- **Primary Contact:** [Your Name/Team]
- **Escalation:** [Tech Lead/Manager]
- **Emergency Rollback Authority:** [CTO/VP Engineering]

---

## Checklist Sign-off

- [ ] **Pre-deployment checks complete** - Signed: __________ Date: __________
- [ ] **Deployment executed** - Signed: __________ Date: __________
- [ ] **Post-deployment verification** - Signed: __________ Date: __________
- [ ] **24-hour monitoring complete** - Signed: __________ Date: __________

---

**Last Updated:** January 4, 2026
**Version:** 1.0
**Feature:** Unified Booking & Messaging Flow
