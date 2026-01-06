# Fix Log: User Profiles Showing Fake/Incorrect Data

**Date:** January 5, 2026
**Priority:** P0 (Critical Data Integrity Issue)
**Status:** ✅ RESOLVED
**Component:** frontend/src/stores/users.ts

---

## Problem Statement

When users clicked "View Profile" on store items, the profile information displayed was **fake/incorrect data** with random names like "Jordan Smith" or "Morgan Williams" instead of the actual seller's information from the database.

### Impact
- **Critical data integrity issue** - Users seeing incorrect information
- **Trust erosion** - Fake profiles undermined platform credibility
- **Business logic failure** - Unable to verify seller reputation before purchases
- **Security concern** - Users couldn't identify who they were actually dealing with
- **Hidden backend failures** - API errors silently replaced with fake data

### User Experience
```
1. User views a store item
2. Sees seller name "John Doe"
3. Clicks "View Profile"
4. Profile shows "Morgan Williams" with random data
5. User confused: "Who is this person? Where is John Doe?"
```

---

## Root Cause Analysis

### The Critical Flaw: Silent Fake Data Generation

**File:** `frontend/src/stores/users.ts:36-109` (old version)

The users store had a **dangerous fallback mechanism** that silently generated fake user data when the API call failed:

```javascript
// ❌ CRITICAL ISSUE: Old Code
const getUserById = async (userId: number): Promise<User | null> => {
  // Check cache first
  if (userCache.value.has(userId)) {
    return userCache.value.get(userId) || null;
  }

  try {
    // Try to fetch from API
    const response = await fetch(`${config.API_URL}/users/${userId}`, { ... });

    if (response.ok) {
      return await response.json(); // ✅ Return real data if successful
    }
  } catch (apiError) {
    console.warn(`API endpoint for users likely doesn't exist:`, apiError);
    // ❌ PROBLEM: Silently ignores API errors
  }

  // ❌ CRITICAL PROBLEM: Generate fake data as fallback
  const randomNames = ["Alex", "Morgan", "Jordan", "Taylor", "Riley", "Casey", "Jamie", "Avery"];
  const randomLastNames = ["Smith", "Johnson", "Williams", "Jones", "Brown", "Davis"];
  const randomName = randomNames[Math.floor(Math.random() * randomNames.length)];
  const randomLastName = randomLastNames[Math.floor(Math.random() * randomLastNames.length)];

  const mockUser: User = {
    id: userId,
    username: `${randomName.toLowerCase()}${userId}`,
    fullName: `${randomName} ${randomLastName}`,  // ❌ FAKE NAME
    email: `${randomName.toLowerCase()}${userId}@example.com`,  // ❌ FAKE EMAIL
    averageRating: (3 + Math.random() * 2).toFixed(1),  // ❌ FAKE RATING
    created_at: joinDate.toISOString(),
  };

  // Cache the fake data (!)
  userCache.value.set(userId, mockUser);
  return mockUser;  // ❌ Return fake data to user
}
```

### Why This Was Catastrophically Wrong

1. **Data Integrity Violation**
   - Users saw **completely fabricated information**
   - No indication the data was fake
   - Cached fake data prevented recovery on reload

2. **Silent Failure**
   - API errors were caught and ignored: `console.warn(...)`
   - No error shown to user
   - No way to know API was failing

3. **Security/Trust Issue**
   - Users couldn't verify who they were transacting with
   - Ratings were random numbers, not real reviews
   - Impossible to assess seller trustworthiness

4. **Development Anti-Pattern**
   - Fallback behavior masked real bugs
   - Made testing unreliable (sometimes real data, sometimes fake)
   - Impossible to debug why profiles were wrong

### Contributing Issues

#### Issue 1: Mock User Pre-Population
```javascript
// ❌ Lines 19-32: Pre-populated cache with fake users
const mockUsers: User[] = [
  { id: 1, username: "test_user", fullName: "Test User", ... },
  { id: 2, username: "alice_dev", fullName: "Alice Developer", ... },
  // ...
];

mockUsers.forEach(user => {
  userCache.value.set(user.id, user);  // ❌ Cache filled with fake data at startup
});
```

If a user with ID 1-5 existed in the database, the store would return the pre-cached fake user instead of fetching real data.

#### Issue 2: No Field Name Normalization
The backend returns `snake_case` fields (`full_name`, `average_rating`), but the frontend expected `camelCase` (`fullName`, `averageRating`). No normalization was done, causing data to be lost or mapped incorrectly.

#### Issue 3: No Error Propagation
Errors were caught and logged but never re-thrown, so calling code (like UserProfileView) had no way to know the fetch failed and display an appropriate error message to the user.

---

## Solution Implemented

### Complete Rewrite of getUserById

**File:** `frontend/src/stores/users.ts:21-107`

Replaced the dangerous fallback logic with a **proper, production-ready implementation**:

```javascript
// ✅ NEW CODE: Always fetch real data, fail properly on errors
const getUserById = async (userId: number): Promise<User | null> => {
  // Return cached data if available
  if (userCache.value.has(userId)) {
    console.log(`✅ Using cached user data for ID ${userId}`);
    return userCache.value.get(userId) || null;
  }

  console.log(`🔄 Fetching user data from API for ID ${userId}`);

  try {
    const authStore = useAuthStore();
    const token = authStore.token;

    // ✅ Check for auth token
    if (!token) {
      console.error('❌ No authentication token available');
      throw new Error('Authentication required');
    }

    const apiUrl = `${config.API_URL}/users/${userId}`;
    console.log(`📡 API Request: GET ${apiUrl}`);

    const response = await fetch(apiUrl, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      }
    });

    console.log(`📥 API Response: ${response.status} ${response.statusText}`);

    if (response.ok) {
      const userData = await response.json();
      console.log(`✅ User data loaded successfully:`, {
        id: userData.id,
        username: userData.username,
        fullName: userData.full_name || userData.fullName
      });

      // ✅ Normalize field names (backend snake_case → frontend camelCase)
      const normalizedUser: User = {
        id: userData.id,
        username: userData.username,
        email: userData.email,
        fullName: userData.full_name || userData.fullName,
        bio: userData.bio,
        averageRating: userData.average_rating || userData.averageRating,
        created_at: userData.created_at || userData.createdAt,
        createdAt: userData.created_at || userData.createdAt
      };

      // Cache the real data
      userCache.value.set(userId, normalizedUser);
      return normalizedUser;
    } else {
      // ✅ Proper error handling with specific messages
      const errorText = await response.text();
      console.error(`❌ API Error: ${response.status} - ${errorText}`);

      if (response.status === 404) {
        throw new Error(`User with ID ${userId} not found`);
      } else if (response.status === 401) {
        throw new Error('Authentication failed - please log in again');
      } else {
        throw new Error(`Failed to load user: ${response.status} ${response.statusText}`);
      }
    }
  } catch (error) {
    console.error(`❌ Error fetching user ${userId}:`, error);
    // ✅ Re-throw error so caller can handle it
    throw error;
  }
}
```

### Key Improvements

1. **✅ No Fake Data Generation**
   - Removed all mock user generation code
   - Removed random name arrays
   - Removed fallback user creation

2. **✅ Proper Error Handling**
   - Specific error messages for 404, 401, etc.
   - Errors are re-thrown to calling code
   - Detailed logging at each step

3. **✅ Field Name Normalization**
   - Converts backend `snake_case` to frontend `camelCase`
   - Handles both formats for compatibility
   - Ensures all fields are properly mapped

4. **✅ Better Debugging**
   - Emoji indicators in logs (🔄 📡 ✅ ❌)
   - Logs API URL, status, and response data
   - Clear indication of success/failure

5. **✅ Auth Token Validation**
   - Checks for token before making request
   - Fails fast if not authenticated
   - Clear error message to user

### Removed Mock Data Initialization

**Before:**
```javascript
// ❌ Lines 19-33: Pre-filled cache with fake users
const mockUsers: User[] = [ ... ];
mockUsers.forEach(user => {
  userCache.value.set(user.id, user);
});
```

**After:**
```javascript
// ✅ Line 18-19: Empty cache, filled only with real API data
const userCache = ref<Map<number, User>>(new Map())
```

---

## How It Works Now

### Normal Flow (API Success)

```
1. User clicks "View Profile" for User #42
   ├─→ UserProfileView calls usersStore.getUserById(42)

2. Store checks cache
   ├─→ User #42 not in cache
   ├─→ Logs: "🔄 Fetching user data from API for ID 42"

3. Store makes API request
   ├─→ GET http://localhost:8080/api/v1/users/42
   ├─→ Headers: Authorization: Bearer <token>
   ├─→ Logs: "📡 API Request: GET ..."

4. Backend responds with real user data
   ├─→ 200 OK
   ├─→ Body: { id: 42, username: "john_doe", full_name: "John Doe", ... }
   ├─→ Logs: "📥 API Response: 200 OK"

5. Store normalizes and caches data
   ├─→ Converts snake_case to camelCase
   ├─→ Stores in cache for future requests
   ├─→ Logs: "✅ User data loaded successfully: {id: 42, username: 'john_doe', ...}"

6. Returns real user data to UserProfileView
   ├─→ Profile displays "John Doe" (correct!)
   ├─→ Shows real rating, bio, join date
```

### Error Flow (User Not Found)

```
1. User clicks "View Profile" for User #999 (doesn't exist)
   ├─→ UserProfileView calls usersStore.getUserById(999)

2. Store makes API request
   ├─→ GET http://localhost:8080/api/v1/users/999

3. Backend responds with 404
   ├─→ Response: 404 Not Found
   ├─→ Body: {"error": "User not found"}
   ├─→ Logs: "❌ API Error: 404 - {"error":"User not found"}"

4. Store throws specific error
   ├─→ throw new Error("User with ID 999 not found")

5. UserProfileView catches error
   ├─→ Shows error message to user: "User not found"
   ├─→ No fake data generated
   ├─→ User knows there's a problem
```

### Error Flow (Network/Auth Issues)

```
1. Network failure or auth token expired

2. Store throws appropriate error
   ├─→ No token: "Authentication required"
   ├─→ 401: "Authentication failed - please log in again"
   ├─→ Network: "Failed to load user: [details]"

3. UserProfileView displays error
   ├─→ User sees clear error message
   ├─→ Can take action (re-login, check connection)
```

---

## Testing & Verification

### Test Scenarios

✅ **Scenario 1: View Real User Profile**
1. Navigate to store item with real seller
2. Click "View Profile"
3. Verify: Displays actual seller name from database
4. Verify: Shows real rating, bio, join date
5. Console shows: "✅ User data loaded successfully"

✅ **Scenario 2: View Non-Existent User**
1. Manually navigate to `/profile/99999`
2. Verify: Error message displayed
3. Verify: No fake data shown
4. Console shows: "❌ API Error: 404"

✅ **Scenario 3: Unauthenticated Request**
1. Clear auth token / logout
2. Try to view a profile
3. Verify: "Authentication required" error
4. Verify: Prompted to log in

✅ **Scenario 4: Backend Down**
1. Stop backend service
2. Try to view profile
3. Verify: Clear error message (not fake data)
4. Console shows network error

✅ **Scenario 5: Cached Data**
1. View a profile (loads from API)
2. View same profile again
3. Verify: Instant load (from cache)
4. Console shows: "✅ Using cached user data"

### Backend Verification

Confirmed backend endpoint working correctly:
- **Endpoint:** `GET /api/v1/users/:id`
- **Handler:** `backend/internal/api/handlers.go:506-524`
- **Service:** `backend/internal/services/user_service.go:141-157`
- **Returns:** Full UserResponse with all fields
- **Fields:** id, username, email, full_name, bio, average_rating, created_at, updated_at

---

## Files Modified

| File | Lines Changed | Change Summary |
|------|--------------|----------------|
| `frontend/src/stores/users.ts` | 19-109 | Complete rewrite of getUserById, removed mock data generation and initialization |

**Detailed Changes:**
- **Removed:** ~75 lines of mock data generation logic
- **Removed:** Pre-populated mock users array
- **Added:** Proper error handling with specific error types
- **Added:** Field name normalization (snake_case → camelCase)
- **Added:** Comprehensive logging with emoji indicators
- **Added:** Auth token validation
- **Added:** Error re-throwing for proper error propagation

---

## Impact Assessment

### Before Fix

| Aspect | Status |
|--------|--------|
| **Data Accuracy** | ❌ Fake data shown 100% of time when API failed |
| **User Trust** | ❌ Users saw random incorrect names |
| **Error Visibility** | ❌ Errors silently replaced with fake data |
| **Debugging** | ❌ Impossible to diagnose - logs just said "creating mock user" |
| **Security** | ❌ Couldn't verify transaction partner identity |

### After Fix

| Aspect | Status |
|--------|--------|
| **Data Accuracy** | ✅ Real data from database always (or explicit error) |
| **User Trust** | ✅ See actual seller information |
| **Error Visibility** | ✅ Clear error messages with actionable information |
| **Debugging** | ✅ Detailed logs show exactly what happened |
| **Security** | ✅ Can verify who they're dealing with |

---

## Why This Happened

### Development Practice Issues

1. **Over-Aggressive Fallbacks**
   - Developer wanted app to "work" even if backend wasn't ready
   - Created fallback that was too permissive
   - Should have failed loudly, not silently masked errors

2. **Mock Data for Development**
   - Mock data is useful for development
   - Should be explicitly enabled (env variable/flag)
   - Should NEVER silently replace real data in production

3. **Insufficient Error Handling**
   - Errors were caught but not propagated
   - No distinction between "development mode" and "production mode"
   - No visibility into when/why fallbacks were triggered

4. **Lack of Testing**
   - No integration tests to verify API connection
   - No tests to ensure real data was being fetched
   - Mock data masked the fact that API integration was broken

---

## Preventive Measures

### Immediate (Completed)

✅ **Removed all fake data generation**
- No more mock users created at runtime
- No pre-populated fake cache
- API is the single source of truth

✅ **Proper error handling**
- Errors are surfaced to users
- Clear actionable error messages
- Detailed logging for debugging

✅ **Field normalization**
- Backend snake_case properly converted
- All fields correctly mapped
- No data loss in translation

### Short-term (Recommended)

1. **Add API Health Checks**
   - Monitor users endpoint availability
   - Alert if endpoint returns errors
   - Dashboard showing API success rates

2. **Integration Tests**
   - Test that profiles load real data from API
   - Test error handling (404, 401, 500)
   - Test field name normalization

3. **Development Mode Flag**
   - Explicit `VITE_USE_MOCK_DATA=true` env variable
   - Only use mocks when explicitly enabled
   - Log warning if mocks are enabled

### Long-term (Future)

1. **Centralized Error Handling**
   - Create ErrorService to handle all API errors
   - Consistent error messages across app
   - User-friendly error recovery flows

2. **Data Validation**
   - Validate API responses match expected schema
   - TypeScript interfaces for all API responses
   - Runtime validation with Zod/Yup

3. **Observability**
   - Track profile load success/failure rates
   - Monitor cache hit rates
   - Alert on unusual error patterns

---

## Related Backend API

### Endpoint Details

**URL:** `GET /api/v1/users/:id`

**Request:**
```http
GET /api/v1/users/42 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Successful Response (200 OK):**
```json
{
  "id": 42,
  "username": "john_doe",
  "email": "john@example.com",
  "full_name": "John Doe",
  "phone_number": "+1234567890",
  "bio": "Seller since 2023",
  "average_rating": 4.7,
  "created_at": "2023-06-15T10:30:00Z",
  "updated_at": "2024-01-05T14:20:00Z"
}
```

**Error Response (404 Not Found):**
```json
{
  "error": "User not found"
}
```

**Error Response (401 Unauthorized):**
```json
{
  "error": "Unauthorized"
}
```

---

## Lessons Learned

### 1. **Never Silently Replace Real Data with Fake Data**
**Problem:** Fake data fallback masked API failures
**Solution:** Always fail explicitly when real data unavailable

### 2. **Fail Loudly, Not Silently**
**Problem:** Errors were caught and ignored
**Solution:** Propagate errors to UI, let users know something's wrong

### 3. **Development Mocks Should Be Explicit**
**Problem:** Mock data was default behavior
**Solution:** Require explicit flag to use mocks, warn when enabled

### 4. **Field Name Consistency Matters**
**Problem:** Backend and frontend used different naming conventions
**Solution:** Normalize field names at API boundary

### 5. **Logging Should Tell a Story**
**Problem:** Logs were minimal and unclear
**Solution:** Comprehensive logging with clear indicators

---

## Monitoring Recommendations

### Metrics to Track

1. **getUserById Success Rate**
   - Target: >99%
   - Alert if drops below 95%

2. **Average getUserById Latency**
   - Target: <200ms
   - Alert if >1000ms

3. **Cache Hit Rate**
   - Track how often cache is used vs API calls
   - Optimize caching strategy based on patterns

4. **Error Types Distribution**
   - 404 errors: Track if many users not found
   - 401 errors: Indicates auth issues
   - 500 errors: Backend problems

### Alerts to Configure

1. **High Error Rate**
   - If >5% of requests fail, page on-call
   - Indicates backend or network issues

2. **Specific User Not Found**
   - If same user ID gets 404 repeatedly
   - May indicate data integrity issue

3. **Auth Failures Spike**
   - Sudden increase in 401 errors
   - May indicate JWT secret mismatch or expiry issues

---

## Status

**✅ RESOLVED** - User profiles now display real data from the database. Fake data generation completely removed.

**User Impact:** CRITICAL FIX - Users can now see accurate seller information

**Security Impact:** IMPROVED - Users can verify transaction partner identity

**Developer Impact:** IMPROVED - Errors are now visible and debuggable

**Next Steps:**
1. Monitor API success rates
2. Add integration tests for user profile loading
3. Consider adding development mode flag for optional mock data
4. Document API response schemas for all endpoints

---

## Related Documentation

- **Backend Endpoint:** `backend/internal/api/handlers.go:506-524` (GetUserByID)
- **Backend Service:** `backend/internal/services/user_service.go:141-157` (GetUser)
- **User Model:** `backend/internal/models/user.go:20-30` (UserResponse)
- **Router Config:** `backend/cmd/api/main.go:107` (Route registration)
- **Recent Fix:** See [FIXLOG-seller-profile-redirect-issue.md](./FIXLOG-seller-profile-redirect-issue.md) for related profile navigation fix
