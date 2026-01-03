# Chat Service Investigation Report
**Date:** January 2, 2026
**Status:** Service Down - Migration Failures

## Executive Summary

The chat-websocket-service is currently non-functional due to database migration errors. The service architecture is well-designed, but migration complexity and database schema conflicts are preventing startup.

**Current Impact:**
- ❌ Real-time messaging unavailable
- ❌ WebSocket connections failing
- ✅ Main app functionality works (tasks, store, auth)
- ⚠️ Frontend shows connection errors but doesn't crash

---

## Critical Issues Identified

### 1. Migration System Failures ⚠️ CRITICAL

**Problem:** Complex SQL migrations incompatible with node-pg-migrate

**Root Causes:**

#### a) Nested DO Blocks Not Supported (Migration 002)
```sql
-- Migration 002: Lines 73-118
DO $$
BEGIN
    -- Outer DO block
    IF EXISTS (...) THEN
        -- Creates function with ANOTHER nested $$
        CREATE OR REPLACE FUNCTION update_store_item_message_count()
        RETURNS TRIGGER AS $$  -- INNER nested delimiter
        BEGIN
            -- Function body
        END;
        $$ LANGUAGE plpgsql;  -- Closes inner $$
    END IF;
END
$$;  -- Closes outer DO block
```

**Error:** `syntax error at or near "BEGIN"` at position 2675

**Why It Fails:**
- node-pg-migrate's SQL parser struggles with nested `$$` delimiters
- The outer DO block contains a CREATE FUNCTION with its own `$$ ... $$` block
- Parser gets confused about which `$$` closes which block

#### b) Explicit Transaction Management (Migrations 003, 004)
```sql
BEGIN;
-- Migration code
COMMIT;
```

**Error:** node-pg-migrate manages transactions automatically

**Why It Fails:**
- node-pg-migrate wraps each migration file in its own transaction
- Explicit BEGIN/COMMIT creates nested transactions
- Can cause "already in transaction" errors

#### c) Migration Order and Redundancy

**Current Order:**
1. `000_initial_schema.sql` ✅ - Creates base tables
2. `001_message_updates.sql` ✅ - Adds columns to messages
3. `002_store_message_integration.sql` ❌ - Adds store_item_id + complex triggers
4. `003_fix_messages_table.sql` ❌ - ALSO adds store_item_id (duplicate!)
5. `004_fix_task_references.sql` ❌ - Renames columns

**Conflicts:**
- Migrations 002 and 003 both try to add `store_item_id` column
- Migration 004 tries to rename `creator_id` to `created_by` but column may not exist
- Both try to create same indexes

---

### 2. Database Schema Conflicts

#### Shared Database Issues
All services use the same PostgreSQL database `my_guy`:
- Main Backend (Go) - Manages: users, tasks, applications, reviews
- Store Service (Go) - Manages: store_items, bids, purchases
- Chat Service (Node) - Manages: messages, user_activity

**Problems:**
1. **Race Conditions**: Multiple services can run migrations simultaneously
2. **Schema Ownership**: Unclear which service "owns" shared tables
3. **Foreign Key Dependencies**: Chat service references tables it doesn't create

#### Missing Table References
Migration 002 references tables that may not exist:
```sql
-- Checks if store_items exists (created by store-service)
IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'store_items')

-- Checks if store_messages exists (obsolete table)
IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'store_messages')
```

**Issue:** These tables are created by different services with uncertain timing

---

### 3. Migration Script Issues

**File:** `chat-websocket-service/src/scripts/migrate.js`

#### Default Database Name Wrong
```javascript
const dbConnection = process.env.DB_CONNECTION ||
  'postgresql://postgres:password@localhost:5432/myguy';  // Wrong DB name
```

**Problem:**
- Default uses `myguy` (no underscore)
- Actual database is `my_guy` (with underscore)
- Will fail if DB_CONNECTION env var not set

#### Environment Variable Confusion
- Migration script reads `DB_CONNECTION` (line 11)
- But Docker entrypoint might set `DATABASE_URL`
- Uses fallback that won't work in production

---

### 4. Frontend Chat Integration Issues

**File:** `frontend/src/stores/chat.ts`

#### No Graceful Degradation
```javascript
function connectSocket() {
  socket.value = io(import.meta.env.VITE_CHAT_WS_URL || 'http://localhost:8082', {
    reconnection: true,
    reconnectionAttempts: 3,  // Only 3 attempts
  });
}
```

**Problems:**
1. **Limited Retries**: Only 3 reconnection attempts, then gives up
2. **No Error State**: Doesn't set a "chat unavailable" flag
3. **Console Spam**: Failed connections keep logging errors
4. **User Experience**: No indication to user that chat is disabled

#### Hardcoded URLs
```javascript
// Line 625 - chat.ts
const response = await fetch('http://localhost:8082/api/v1/deletion-warnings', {
```

**Issues:**
- Bypasses environment variable configuration
- Will fail in production/staging
- Should use `config.CHAT_API_URL`

---

### 5. Docker Entrypoint Issues

**File:** `chat-websocket-service/docker-entrypoint.sh`

#### No Error Handling for Migration Failures
```bash
#!/bin/sh
set -e  # Exits on any error

# Wait for database
echo "Waiting for database..."
# ... wait logic ...

# Run migrations
echo "Running database migrations..."
npm run migrate  # If this fails, container exits

# Start the application
echo "Starting chat websocket service..."
exec node src/server.js  # Never reached if migration fails
```

**Problems:**
1. **No Fallback**: Migration failure = container crash
2. **No Retry Logic**: Doesn't retry failed migrations
3. **Poor Debugging**: Limited logging of what went wrong
4. **Production Risk**: One migration error brings down entire service

#### Database Wait Logic
```bash
timeout=60
while ! nc -z postgres-db 5432; do
  # Wait for DB
done
```

**Issue:** Checks if database *accepts connections*, not if it's *ready for queries*

---

## Architecture Analysis

### What's Working Well ✅

1. **Service Separation**: Clean microservice architecture
2. **Code Quality**: Well-structured Express + Socket.IO implementation
3. **Security**: JWT authentication on both HTTP and WebSocket
4. **Content Filtering**: Removes URLs, emails, phone numbers from messages
5. **Message Lifecycle**: Sophisticated soft-delete and archival system
6. **Logging**: Winston logger with proper levels and formatting

### Design Patterns Observed

1. **Unified Messages Table**: Single table for task/application/store messages
   - ✅ Reduces complexity
   - ✅ Easier to query all user messages
   - ⚠️ Requires complex foreign keys and triggers

2. **Scheduled Cleanup**: node-cron jobs for message deletion
   - ✅ Prevents database bloat
   - ✅ Complies with data retention policies
   - ⚠️ Complex warning system

3. **Shared Database**: All services use one PostgreSQL instance
   - ✅ Simpler infrastructure
   - ✅ Easy cross-service queries
   - ❌ Schema conflicts and race conditions
   - ❌ Harder to scale independently

---

## Detailed Recommendations

### 🔴 IMMEDIATE (Fix Now)

#### 1. Simplify Migration 002
**File:** `chat-websocket-service/migrations/002_store_message_integration.sql`

**Current Approach:** Complex DO block with nested CREATE FUNCTION
**Recommended Approach:** Split into 3 separate migrations

**New Structure:**
```
002a_add_store_columns.sql       - Just add columns and indexes
002b_migrate_store_data.sql      - Data migration (if needed)
002c_store_triggers.sql          - Create functions and triggers separately
```

**Why This Works:**
- node-pg-migrate handles simple SQL better
- Easier to debug which step fails
- Can rollback individual steps
- No nested delimiter issues

**Alternative:** Use raw SQL execution instead of node-pg-migrate for complex migrations

---

#### 2. Remove BEGIN/COMMIT from Migrations 003 & 004
**Files:**
- `003_fix_messages_table.sql` (line 2, line 44)
- `004_fix_task_references.sql` (line 2, line 16)

**Action:** Delete `BEGIN;` and `COMMIT;` statements

**Reason:** node-pg-migrate manages transactions automatically

---

#### 3. Fix Migration Redundancy
**Problem:** Migrations 002 and 003 both add `store_item_id`

**Solution:**
- Remove duplicate column creation from 003
- Keep column in 002 OR 003 (not both)
- Add `IF NOT EXISTS` checks everywhere

**Suggested Approach:**
```sql
-- In 002: Add the column initially
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS store_item_id INTEGER;

-- In 003: Skip this, or just verify it exists
-- (Don't try to add again)
```

---

#### 4. Fix Default Database Name
**File:** `chat-websocket-service/src/scripts/migrate.js` (line 11)

**Change:**
```javascript
// FROM:
const dbConnection = process.env.DB_CONNECTION || 'postgresql://postgres:password@localhost:5432/myguy';

// TO:
const dbConnection = process.env.DB_CONNECTION || process.env.DATABASE_URL ||
  'postgresql://postgres:password@localhost:5432/my_guy';  // Underscore added
```

---

#### 5. Add Environment Variable Fallback
**File:** Same as above

**Enhancement:**
```javascript
const dbConnection = process.env.DATABASE_URL ||
                     process.env.DB_CONNECTION ||
                     'postgresql://postgres:mysecretpassword@postgres-db:5432/my_guy';

// Log which connection is being used
logger.info(`Using database connection: ${dbConnection.replace(/:[^:]*@/, ':****@')}`);
```

---

### 🟡 HIGH PRIORITY (Fix Soon)

#### 6. Add Migration Error Handling
**File:** `chat-websocket-service/docker-entrypoint.sh`

**Current:**
```bash
npm run migrate  # Fails = container exits
exec node src/server.js
```

**Recommended:**
```bash
# Run migrations with retry logic
MAX_RETRIES=3
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  echo "Running database migrations (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)..."
  if npm run migrate; then
    echo "✓ Migrations completed successfully"
    break
  else
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
      echo "✗ Migration failed, retrying in 5 seconds..."
      sleep 5
    else
      echo "✗ Migration failed after $MAX_RETRIES attempts"
      echo "Starting service anyway (migrations can be run manually)"
      # Don't exit - start service in degraded mode
    fi
  fi
done

# Start the application
echo "Starting chat websocket service..."
exec node src/server.js
```

**Benefits:**
- Service can start even if migrations fail
- Better debugging with retry logs
- Production resilience

---

#### 7. Frontend: Add Chat Unavailable State
**File:** `frontend/src/stores/chat.ts`

**Add State Variable:**
```typescript
const chatUnavailable = ref(false);
const connectionError = ref<string | null>(null);
```

**Update Connection Logic:**
```typescript
function connectSocket() {
  if (socket.value?.connected) return;

  try {
    socket.value = io(import.meta.env.VITE_CHAT_WS_URL || 'http://localhost:8082', {
      auth: { token: authStore.token },
      reconnection: true,
      reconnectionDelay: 1000,
      reconnectionAttempts: 10,  // Increase from 3
      reconnectionDelayMax: 5000,
    });

    socket.value.on('connect', () => {
      connected.value = true;
      chatUnavailable.value = false;
      connectionError.value = null;
      console.log('✓ Chat connected');
    });

    socket.value.on('connect_error', (error) => {
      console.warn('Chat connection failed:', error.message);
      connectionError.value = error.message;

      // After multiple failures, mark as unavailable
      if (socket.value?.io.engine?.transports.length === 0) {
        chatUnavailable.value = true;
      }
    });

    socket.value.on('disconnect', () => {
      connected.value = false;
      console.log('Chat disconnected');
    });

  } catch (error) {
    console.error('Failed to initialize chat:', error);
    chatUnavailable.value = true;
  }
}
```

**Add to Return Statement:**
```typescript
return {
  // ... existing exports
  chatUnavailable,
  connectionError,
};
```

---

#### 8. Fix Hardcoded Chat URLs
**File:** `frontend/src/stores/chat.ts` (lines 625, 645)

**Change:**
```javascript
// FROM:
const response = await fetch('http://localhost:8082/api/v1/deletion-warnings', {

// TO:
const response = await fetch(`${config.CHAT_API_URL}/deletion-warnings`, {
```

---

#### 9. Add Migration Order Documentation
**File:** Create `chat-websocket-service/migrations/README.md`

**Content:**
```markdown
# Chat Service Migrations

## Migration Order

1. **000_initial_schema.sql** - Base tables (users, tasks, applications, messages)
2. **001_message_updates.sql** - Message features (read, edit, delete tracking)
3. **002_store_message_integration.sql** - Store item support
4. **003_fix_messages_table.sql** - ⚠️ DEPRECATED - functionality moved to 002
5. **004_fix_task_references.sql** - Column renaming

## Prerequisites

- PostgreSQL 12+
- Tables created by main backend: users, tasks, applications
- Tables created by store service: store_items (optional)

## Manual Migration

If automatic migrations fail:

\`\`\`bash
cd chat-websocket-service
npm run migrate
\`\`\`

## Troubleshooting

**Error: "syntax error at or near BEGIN"**
- Remove explicit BEGIN/COMMIT from migration files
- node-pg-migrate handles transactions automatically

**Error: "relation already exists"**
- Migrations already partially applied
- Check: SELECT * FROM pgmigrations;
- Manually mark as complete or rollback
```

---

### 🟢 MEDIUM PRIORITY (Improve Later)

#### 10. Consider Separate Databases per Service
**Current:** All services share `my_guy` database
**Recommendation:** Give each service its own database

**New Structure:**
```
my_guy_main    - Main backend (users, tasks, applications, reviews)
my_guy_store   - Store service (store_items, bids, purchases)
my_guy_chat    - Chat service (messages, user_activity)
```

**Benefits:**
- ✅ No schema conflicts
- ✅ Independent scaling
- ✅ Clear ownership
- ✅ Easier backups per service

**Challenges:**
- ❌ Cross-database foreign keys not possible
- ❌ Need to duplicate user data
- ❌ More complex joins

**Alternative:** Keep shared DB but document schema ownership clearly

---

#### 11. Implement Migration Version Checking
**Add to:** `chat-websocket-service/src/server.js`

**Before starting server:**
```javascript
const checkMigrations = async () => {
  const client = await pool.connect();
  try {
    const result = await client.query(`
      SELECT id, name, run_on
      FROM pgmigrations
      ORDER BY id DESC
      LIMIT 1
    `);

    const EXPECTED_MIGRATION = '004_fix_task_references';
    const currentMigration = result.rows[0]?.name;

    if (currentMigration !== EXPECTED_MIGRATION) {
      logger.warn(`Migration mismatch! Expected: ${EXPECTED_MIGRATION}, Got: ${currentMigration}`);
      logger.warn('Some features may not work correctly');
    } else {
      logger.info(`✓ Database migrations up to date: ${currentMigration}`);
    }
  } finally {
    client.release();
  }
};

// In server startup
checkMigrations().catch(err => logger.error('Migration check failed:', err));
```

---

#### 12. Add Health Check with Migration Status
**File:** `chat-websocket-service/src/server.js`

**Enhance `/health` endpoint:**
```javascript
app.get('/health', async (req, res) => {
  const health = {
    status: 'ok',
    service: 'chat-websocket-service',
    uptime: process.uptime(),
    timestamp: new Date().toISOString(),
    database: 'unknown',
    migrations: 'unknown',
  };

  try {
    // Check database connection
    await pool.query('SELECT NOW()');
    health.database = 'connected';

    // Check migration status
    const result = await pool.query(`
      SELECT COUNT(*) as count, MAX(run_on) as last_run
      FROM pgmigrations
    `);
    health.migrations = {
      count: result.rows[0].count,
      last_run: result.rows[0].last_run,
    };

  } catch (error) {
    health.status = 'degraded';
    health.error = error.message;
  }

  res.status(health.status === 'ok' ? 200 : 503).json(health);
});
```

---

#### 13. Add Chat Feature Flags
**File:** `frontend/src/config.ts`

**Add:**
```typescript
export default {
  API_URL,
  CHAT_API_URL,
  FEATURES: {
    ENABLE_CHAT: import.meta.env.VITE_ENABLE_CHAT !== 'false',  // Default: enabled
    ENABLE_STORE_MESSAGES: import.meta.env.VITE_ENABLE_STORE_MESSAGES !== 'false',
    CHAT_RETRY_ATTEMPTS: parseInt(import.meta.env.VITE_CHAT_RETRY_ATTEMPTS || '10'),
  },
  // ...
};
```

**Use in frontend:**
```typescript
// In chat store
if (!config.FEATURES.ENABLE_CHAT) {
  console.log('Chat feature disabled via configuration');
  chatUnavailable.value = true;
  return;
}
```

---

### 🔵 LOW PRIORITY (Nice to Have)

#### 14. Add Migration Tests
**Create:** `chat-websocket-service/tests/migrations.test.js`

**Test each migration independently:**
```javascript
describe('Database Migrations', () => {
  test('000_initial_schema creates base tables', async () => {
    // Run migration
    // Check tables exist
    // Verify column types
  });

  test('001_message_updates adds tracking columns', async () => {
    // Verify columns added
    // Check indexes created
  });

  // ... etc
});
```

---

#### 15. Add Observability
**Recommendations:**
- Add Prometheus metrics endpoint
- Track message counts, connection counts
- Monitor migration execution time
- Alert on repeated migration failures

---

#### 16. Document Service Dependencies
**Create:** `chat-websocket-service/docs/DEPENDENCIES.md`

**Content:**
```markdown
# Service Dependencies

## Required Tables (Created by Other Services)

### Main Backend
- users (id, email, username, name)
- tasks (id, title, status, created_by, assigned_to)
- applications (id, task_id, user_id, status)

### Store Service
- store_items (id, seller_id, title, status)  ← OPTIONAL

## What Happens if Dependencies Missing?

- Missing users table: ✗ Service won't start (foreign keys fail)
- Missing tasks table: ✗ Service won't start
- Missing store_items: ✓ Service starts, store messages disabled
```

---

## Migration Fix Priority Order

### Step 1: Quick Wins (15 minutes)
1. ✅ Remove BEGIN/COMMIT from migrations 003, 004
2. ✅ Fix default database name in migrate.js
3. ✅ Add DATABASE_URL fallback

### Step 2: Core Fixes (1-2 hours)
4. ✅ Split migration 002 into simpler parts
5. ✅ Remove duplicate column creation in 003
6. ✅ Add IF NOT EXISTS to all migrations
7. ✅ Test migrations on fresh database

### Step 3: Resilience (30 minutes)
8. ✅ Add retry logic to docker-entrypoint.sh
9. ✅ Enhance error logging
10. ✅ Update frontend to handle missing chat gracefully

### Step 4: Testing (1 hour)
11. ✅ Test full migration sequence
12. ✅ Test with missing store_items table
13. ✅ Test migration rollback
14. ✅ Verify service starts after migration success

---

## Testing Checklist

Before considering chat service "fixed":

### Migration Tests
- [ ] Fresh database: All migrations run successfully
- [ ] Idempotent: Running migrations twice doesn't break
- [ ] Partial: Can resume from failed migration
- [ ] Rollback: Can downgrade migrations
- [ ] Missing tables: Gracefully handles optional dependencies

### Service Tests
- [ ] Health endpoint responds
- [ ] WebSocket connections succeed
- [ ] JWT authentication works
- [ ] Messages send/receive correctly
- [ ] Frontend connects without errors
- [ ] Graceful degradation when DB unavailable

### Integration Tests
- [ ] Works with main backend running
- [ ] Works with store service running
- [ ] Works when store service down (optional dependency)
- [ ] Multiple concurrent connections stable
- [ ] Message persistence across restarts

---

## Risk Assessment

### Current State Risks
- 🔴 **CRITICAL**: Chat completely unavailable
- 🟡 **HIGH**: Complex migrations can break again on next deploy
- 🟡 **HIGH**: Frontend console errors impact user experience
- 🟢 **LOW**: Main app functionality unaffected

### Post-Fix Risks
- 🟡 **MEDIUM**: Schema conflicts with other services
- 🟡 **MEDIUM**: Performance issues with unified messages table at scale
- 🟢 **LOW**: Migration timing issues in multi-instance deployment

---

## Long-Term Recommendations

1. **Extract to Separate Database** (3-5 days)
   - Eliminates schema conflicts
   - Enables independent scaling
   - Clearer ownership

2. **Add Integration Tests** (2-3 days)
   - Catch migration issues before production
   - Test service interactions
   - Automated regression testing

3. **Implement Circuit Breaker Pattern** (1 day)
   - Frontend doesn't spam failed connections
   - Automatic service discovery
   - Graceful degradation

4. **Add Monitoring & Alerts** (1-2 days)
   - Track chat availability
   - Alert on migration failures
   - Monitor message throughput

5. **Consider Alternative Migration Tool** (2-3 days)
   - node-pg-migrate struggles with complex SQL
   - Alternatives: Flyway, Liquibase, custom solution
   - More control over transaction handling

---

## Conclusion

The chat service is well-architected but suffering from migration complexity. The issues are fixable with systematic refactoring of the migration files and better error handling.

**Estimated Fix Time:** 4-6 hours for full resolution
**Risk Level:** Medium (main app works, isolated to chat feature)
**User Impact:** High for chat users, none for other features

**Recommended Approach:**
1. Start with quick wins (Step 1) - get service running
2. Add resilience (Step 3) - prevent future crashes
3. Refactor migrations properly (Step 2) - long-term stability
4. Add testing (Step 4) - prevent regressions

Once fixed, document the migration process and add integration tests to catch similar issues early.
