# Booking Request API Test

## Testing the GET /api/v1/items/:id/booking-request endpoint

### Fixed Issues:
1. **404 Error Resolved**: Changed response from 404 to 200 with null data when no booking request exists
2. **URL Consistency**: Fixed URL endpoint from `/booking-requests` (plural) to `/booking-request` (singular) to match actual routing
3. **Response Format**: Standardized response format to `{"booking_request": data}` structure

### Test Cases:

#### 1. When no booking request exists:
**Request:** `GET /api/v1/items/2/booking-request`
**Expected Response:** 
```json
{
  "booking_request": null
}
```
**Status Code:** 200 OK

#### 2. When booking request exists:
**Request:** `GET /api/v1/items/2/booking-request`
**Expected Response:**
```json
{
  "booking_request": {
    "id": 1,
    "item_id": 2,
    "requester_id": 456,
    "requester": {
      "id": 456,
      "username": "buyer123",
      "email": "buyer@example.com",
      "name": "John Buyer"
    },
    "status": "pending",
    "message": "I'm interested in this item",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```
**Status Code:** 200 OK

#### 3. When item doesn't exist:
**Request:** `GET /api/v1/items/999/booking-request`
**Expected Response:**
```json
{
  "error": "item not found"
}
```
**Status Code:** 404 Not Found

### Changes Made:

1. **Handler Update** (`store_handlers.go:466-479`):
   - Added proper GORM error detection using `errors.Is(err, gorm.ErrRecordNotFound)`
   - Return 200 with `{"booking_request": null}` when no booking exists
   - Return 404 only for actual errors (like item not found)

2. **Test Updates** (`store_handlers_test.go`):
   - Updated `TestGetBookingRequest` to expect 200 status instead of 404 for missing booking requests
   - Fixed all test URLs from `/booking-requests` to `/booking-request`
   - Added response body validation for null booking request case

3. **Integration Test Updates** (`integration_test.go`):
   - Fixed all integration test URLs to use correct endpoint

### Benefits:
- **Frontend Compatibility**: Frontend no longer receives 404 errors when checking for booking requests
- **Better UX**: Users can see when no booking request exists vs when there's an actual error
- **Consistent API**: Standardized response format across all booking request endpoints
- **URL Consistency**: All endpoints now match the actual routing configuration