# Chat Service Issues Report
**Date:** January 2, 2026
**Status:** Critical Issues Found
**Reporter:** Claude Code Analysis

---

## Executive Summary

The chat service has **4 critical issues** preventing store messages from working. All issues stem from code attempting to access tables/columns that don't exist in the separated `my_guy_chat` database.

**Impact:** Store messaging completely broken (500 errors)
**Root Cause:** Service code still has cross-database dependencies not fixed in Phase 3

---

## Critical Issues

### Issue #1: getStoreMessages - Missing Users Table ❌

**Location:** `src/services/messageService.js:536-552`

**Problem:**
```javascript
async getStoreMessages(itemId, userId) {
  const query = `
    SELECT
      m.*,
      s.username as sender_username,  // ❌ JOINs with users table
      r.username as recipient_username // ❌ JOINs with users table
    FROM messages m
    LEFT JOIN users s ON m.sender_id = s.id      // ❌ users table doesn't exist
    LEFT JOIN users r ON m.recipient_id = r.id   // ❌ users table doesn't exist
    WHERE m.store_item_id = $1
      AND (m.sender_id = $2 OR m.recipient_id = $2)
    ORDER BY m.created_at ASC
  `;
```

**Error:**
```
relation "users" does not exist
```

**Why It Fails:**
- `users` table exists in `my_guy` database (Main Backend)
- Chat service uses `my_guy_chat` database (separated)
- No cross-database JOINs possible

**Impact:**
- GET `/api/v1/store-messages/:itemId` returns 500 error
- Cannot load message history

**Solution Needed:**
- Remove JOIN with users table
- Return only user IDs (sender_id, recipient_id)
- Frontend should fetch usernames via Main API (`GET /api/v1/users/:id`)

---

### Issue #2: createStoreMessage - Missing Column ❌

**Location:** `src/services/messageService.js:557-594`

**Problem:**
```javascript
async createStoreMessage({ store_item_id, sender_id, recipient_id, content }) {
  const messageQuery = `
    INSERT INTO messages (
      store_item_id,
      sender_id,
      recipient_id,
      content,
      original_content,  // ❌ Column doesn't exist
      created_at
    )
    VALUES ($1, $2, $3, $4, $5, NOW())
    RETURNING *
  `;
```

**Error:**
```
column "original_content" of relation "messages" does not exist
```

**Current Schema:**
```sql
-- messages table columns:
- id
- sender_id
- recipient_id
- task_id
- application_id
- store_item_id
- content           ✅ EXISTS
- message_type
- is_read
- read_at
- is_edited
- edited_at
- is_deleted
- deleted_at
- deletion_scheduled_at
- created_at
- updated_at

-- ❌ NO original_content column
```

**Why It Fails:**
- Migration `001_chat_schema.sql` doesn't include `original_content` column
- Code assumes column exists for audit trail

**Impact:**
- POST `/api/v1/store-messages` returns 500 error
- Cannot send store messages

**Solution Needed:**
Either:
1. **Option A:** Add `original_content` column to migration
2. **Option B:** Remove `original_content` from INSERT (only use `content`)

---

### Issue #3: getBookingStatus - Missing Tables ❌

**Location:** `src/services/messageService.js:613-638`

**Problem:**
```javascript
async getBookingStatus(itemId, userId) {
  const query = `
    SELECT status
    FROM booking_requests           // ❌ Table doesn't exist in my_guy_chat
    WHERE item_id = $1
    AND (requester_id = $2 OR item_id IN (
      SELECT id FROM store_items    // ❌ Table doesn't exist in my_guy_chat
      WHERE seller_id = $2
    ))
    ORDER BY created_at DESC
    LIMIT 1
  `;
```

**Error:**
```
relation "booking_requests" does not exist
relation "store_items" does not exist
```

**Why It Fails:**
- `booking_requests` table exists in `my_guy_store` database
- `store_items` table exists in `my_guy_store` database
- Chat service uses `my_guy_chat` database (separated)

**Impact:**
- Cannot determine message limits (defaults to 3 messages)
- Booking approval doesn't increase limit to 10

**Solution Needed:**
- Use ValidationService to check booking status via Store API
- Call `GET /api/v1/store/bookings/:itemId/status` endpoint
- Cache result to reduce API calls

---

### Issue #4: getTaskMessageLimit - Missing Tasks Table ❌

**Location:** `src/services/messageService.js:671-707`

**Problem:**
```javascript
async getTaskMessageLimit(taskId, userId) {
  const taskQuery = `
    SELECT created_by, assigned_to
    FROM tasks                      // ❌ Table doesn't exist in my_guy_chat
    WHERE id = $1
  `;
```

**Error:**
```
relation "tasks" does not exist
```

**Why It Fails:**
- `tasks` table exists in `my_guy` database (Main Backend)
- Chat service uses `my_guy_chat` database (separated)

**Impact:**
- Cannot determine task message limits
- Task messaging may fail

**Solution Needed:**
- Use ValidationService to check task ownership via Main API
- Call `GET /api/v1/tasks/:id` endpoint
- Cache result to reduce API calls

---

## Additional Issues Found

### Issue #5: Application Messages - Multiple Cross-DB Queries ⚠️

**Location:** `src/server.js:320-387`

**Problems:**
```javascript
// Line 327: Queries applications table (doesn't exist)
const appQuery = 'SELECT task_id, applicant_id FROM applications WHERE id = $1';

// Line 338: Queries tasks table (doesn't exist)
const taskQuery = 'SELECT creator_id FROM tasks WHERE id = $1';

// Line 364-368: Queries users table (doesn't exist)
const senderQuery = 'SELECT username FROM users WHERE id = $1';
const recipientQuery = 'SELECT username FROM users WHERE id = $1';
```

**Impact:**
- Application messaging completely broken
- Cannot send messages in application context

---

## Database Separation Status

### Tables Available in my_guy_chat ✅
```
✅ messages
✅ user_activity
✅ message_deletion_warnings
✅ schema_migrations
```

### Tables NOT Available (Different DBs) ❌
```
❌ users              → my_guy database
❌ tasks              → my_guy database
❌ applications       → my_guy database
❌ store_items        → my_guy_store database
❌ booking_requests   → my_guy_store database
```

---

## Summary Table

| Issue | File | Line | Problem | Impact | Severity |
|-------|------|------|---------|--------|----------|
| #1 | messageService.js | 536 | JOIN users table | Can't load messages | 🔴 Critical |
| #2 | messageService.js | 557 | Missing original_content column | Can't send messages | 🔴 Critical |
| #3 | messageService.js | 613 | JOIN booking_requests/store_items | Wrong message limits | 🟡 High |
| #4 | messageService.js | 671 | Query tasks table | Task limits broken | 🟡 High |
| #5 | server.js | 320 | Query applications/tasks/users | App messages broken | 🔴 Critical |

---

## Root Cause Analysis

### Why These Issues Exist

**Phase 3 Incomplete:**
- Phase 3 fixed `getUserConversations`, `getUserDeletionWarnings`, `getMessagesForDeletion`
- **Did NOT fix:**  - Store message methods
  - Application message methods
  - Task message limit methods

**Pattern:**
All unfixed methods follow same anti-pattern:
```javascript
// ❌ BAD: Direct JOIN/query to tables in other databases
SELECT ... FROM messages m
JOIN users u ON m.sender_id = u.id

// ✅ GOOD: Return IDs, fetch details via API
SELECT m.*, m.sender_id, m.recipient_id FROM messages m
// Then: Frontend calls GET /api/v1/users/:id
```

---

## Recommended Fixes

### Priority 1: Make Store Messages Work (Critical)

**Fix #1 - getStoreMessages:**
```javascript
// Remove users JOIN, return IDs only
async getStoreMessages(itemId, userId) {
  const query = `
    SELECT
      m.*
    FROM messages m
    WHERE m.store_item_id = $1
      AND (m.sender_id = $2 OR m.recipient_id = $2)
    ORDER BY m.created_at ASC
  `;
  return await db.query(query, [itemId, userId]);
}
```

**Fix #2 - createStoreMessage:**
```javascript
// Remove original_content column
const messageQuery = `
  INSERT INTO messages (
    store_item_id,
    sender_id,
    recipient_id,
    content,
    message_type,
    created_at
  )
  VALUES ($1, $2, $3, $4, 'store', NOW())
  RETURNING *
`;

const messageResult = await client.query(messageQuery, [
  store_item_id,
  sender_id,
  recipient_id,
  filtered  // No original_content parameter
]);
```

**Fix #3 - getBookingStatus:**
```javascript
// Use ValidationService or return default
async getBookingStatus(itemId, userId) {
  try {
    // TODO: Use ValidationService to check via Store API
    // For now, return null (defaults to 3 message limit)
    logger.warn('Booking status check not implemented (separate DB)');
    return null;
  } catch (error) {
    logger.error('Error checking booking status:', error);
    return null;
  }
}
```

### Priority 2: Fix Application Messages (Critical)

**Fix application message endpoints:**
- Remove direct queries to applications/tasks/users tables
- Use ValidationService for validation
- Return IDs, let frontend fetch details

### Priority 3: Add Missing Column (Optional)

**If audit trail needed:**
```sql
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS original_content TEXT;
```

---

## Testing Required

After fixes applied:

### Store Messages
- [ ] GET `/api/v1/store-messages/:itemId` returns 200
- [ ] POST `/api/v1/store-messages` returns 201
- [ ] Messages appear in conversation list
- [ ] No 500 errors in logs

### Application Messages
- [ ] POST `/api/v1/applications/:id/messages` works
- [ ] GET `/api/v1/applications/:id/messages` works

### Task Messages
- [ ] Message limits calculated correctly
- [ ] No errors when sending task messages

---

## Impact on Frontend

### Current Frontend Code (Broken)

**StoreItemView.vue expects:**
```javascript
// Expects username fields that won't exist
messages.map(msg => ({
  sender: {
    id: msg.sender_id,
    username: msg.sender_username  // ❌ Won't exist after fix
  }
}))
```

### Frontend Changes Needed

**After backend fix, frontend must:**
1. Fetch user details separately:
   ```javascript
   async loadUserDetails(userId) {
     const response = await fetch(`${API_URL}/users/${userId}`);
     return response.json();
   }
   ```

2. Enrich messages with user data:
   ```javascript
   const messages = await fetchStoreMessages(itemId);
   for (let msg of messages) {
     msg.senderDetails = await loadUserDetails(msg.sender_id);
     msg.recipientDetails = await loadUserDetails(msg.recipient_id);
   }
   ```

3. Cache user data to avoid repeated API calls

---

## Files Requiring Changes

### Backend
```
✏️  chat-websocket-service/src/services/messageService.js
    - getStoreMessages (line 536)
    - createStoreMessage (line 557)
    - getBookingStatus (line 613)
    - getTaskMessageLimit (line 671)

✏️  chat-websocket-service/src/server.js
    - Application message endpoints (lines 320-387)
    - Store message endpoints (lines 390-512)

✏️  chat-websocket-service/migrations/001_chat_schema.sql (optional)
    - Add original_content column if needed
```

### Frontend (Potentially)
```
✏️  frontend/src/views/StoreItemView.vue
    - Update message loading logic
    - Add user detail fetching
    - Add user data caching

✏️  frontend/src/stores/messages.ts (if exists)
    - Update message enrichment logic
```

---

## Phase 4 Recommendation

**Create Phase 4: Complete Database Separation**

**Scope:**
1. Fix all remaining cross-database queries
2. Integrate ValidationService throughout
3. Update frontend to handle ID-only responses
4. Add proper caching for cross-service data
5. Comprehensive testing of all message types

**Estimated Time:** 1-2 hours

---

## Conclusion

**Current State:**
- ✅ Phase 1: Database separation - Complete
- ✅ Phase 2: Resilience & monitoring - Complete
- ⚠️  Phase 3: Fix cross-DB queries - **Partially Complete**
  - ✅ getUserConversations
  - ✅ getUserDeletionWarnings
  - ❌ Store message methods
  - ❌ Application message methods
  - ❌ Task limit methods

**Next Steps:**
1. Implement fixes for Issues #1-5
2. Test all messaging scenarios
3. Update frontend if needed
4. Document changes

**Urgency:** High - Store messaging completely broken

---

**Report Generated:** 2026-01-02 20:20
**Next Review:** After Phase 4 implementation
