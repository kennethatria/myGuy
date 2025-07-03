# Store Booking Flow Documentation

## Overview
The MyGuy store booking flow allows users to request booking for fixed-price items, enabling item owners to approve or decline requests and coordinate item exchange.

## Complete User Journey

### 1. Browse Store Items
- Users navigate to `/store` to view all available items
- Items display with price, condition, seller info
- "View Details" button on each item card

### 2. View Item Details
- Click "View Details" navigates to `/store/:id`
- Displays full item information, images, seller details
- Shows different interface based on user relationship to item

### 3. Booking Request (Buyer Perspective)

#### Conditions for Booking Button
- ✅ User is NOT the item owner
- ✅ Item status is "active"
- ✅ Item is fixed-price (not auction)
- ✅ No existing booking request

#### Request Process
1. Click "Request Booking" button
2. System sends POST to `/api/v1/items/:id/booking-request`
3. Request includes auto-generated message: "I'm interested in booking this item: {title}"
4. Button shows "Sending Request..." during processing
5. Success shows "Booking Request Sent" status
6. Failure shows error message

#### Status Display for Buyers
- **Pending**: "Waiting for the owner to respond"
- **Approved**: "Booking Approved! You can now message the owner..."
- **Rejected**: "The owner has declined your booking request"

### 4. Booking Management (Owner Perspective)

#### Owner Interface
- Item owners see "This is your listing" message
- Pending booking requests show requester information
- Two action buttons: "Approve" and "Decline"

#### Approval Process
1. Owner clicks "Approve" button
2. System sends POST to `/api/v1/booking-requests/:id/approve`
3. Success updates status to "approved"
4. Shows confirmation: "Booking request approved! The requester can now message you with up to 10 messages."

#### Rejection Process
1. Owner clicks "Decline" button
2. System prompts: "Are you sure you want to decline this booking request?"
3. If confirmed, sends POST to `/api/v1/booking-requests/:id/reject`
4. Success updates status to "rejected"
5. Shows confirmation: "Booking request declined."

## Messaging Integration

### Message Limits
- **Before Approval**: 3 messages per user per item
- **After Approval**: 10 messages per user per item
- Limits automatically update when booking status changes

### Chat Integration
- "Message Seller" button available on all items
- Opens chat modal with item context
- Message history preserved across sessions
- Content filtering applied to all messages

## API Endpoints

### Store Service (Port 8081)
- `GET /api/v1/items` - List all items
- `GET /api/v1/items/:id` - Get item details
- `POST /api/v1/items/:id/booking-request` - Create booking request
- `GET /api/v1/items/:id/booking-request` - Get booking request
- `POST /api/v1/booking-requests/:id/approve` - Approve request
- `POST /api/v1/booking-requests/:id/reject` - Reject request
- `GET /api/v1/user/booking-requests` - Get user's requests

### Chat Service (Port 8082)
- `GET /api/v1/store-messages/:itemId` - Get item messages
- `POST /api/v1/store-messages` - Send item message
- `GET /api/v1/store-messages/:itemId/limits` - Get message limits

## Frontend Components

### StoreView.vue
- Item grid display
- Search and filtering
- Navigation to item details via `viewItem(item)` function
- "List Item" functionality for creating new items

### StoreItemView.vue
- Complete item details display
- Booking request interface (buyer)
- Booking management interface (owner)
- Integrated chat functionality
- Image gallery with error handling

## State Management

### Key State Variables
```javascript
// Booking-related
const bookingRequest = ref(null)
const hasBookingRequest = ref(false)
const loadingBookingRequest = ref(false)

// Computed properties
const bookingStatus = computed(() => bookingRequest.value?.status || null)
const currentMessageLimit = computed(() => 
  bookingRequest.value?.status === 'approved' ? 10 : 3
)
```

### State Updates
- Booking creation updates `bookingRequest` and `hasBookingRequest`
- Status changes (approve/reject) update `bookingRequest.status`
- Message limits react to booking status changes
- Loading states prevent duplicate requests

## Error Handling

### Common Error Scenarios
- **400 Bad Request**: Invalid JSON or missing fields
- **403 Forbidden**: Cannot book own item
- **404 Not Found**: Item or booking request not found
- **409 Conflict**: Duplicate booking request

### Frontend Error Display
- API errors shown via `alert()` dialogs
- Loading states prevent UI interaction during requests
- Validation prevents invalid actions (e.g., booking own items)
- Network errors handled gracefully with user feedback

## Testing Coverage

### Backend Tests
- ✅ Unit tests for all service methods
- ✅ Integration tests for complete workflows
- ✅ API endpoint tests with proper validation
- ✅ Database transaction testing

### Frontend Tests
- ✅ Component rendering tests
- ✅ User interaction simulation
- ✅ API call verification
- ✅ State management testing
- ✅ Error handling validation

## Security Considerations

### Authentication
- All booking endpoints require JWT authentication
- User authorization verified for ownership actions
- Token validation on both frontend and backend

### Authorization
- Owners can only approve/reject their own item requests
- Users cannot book their own items
- Access control enforced at API level

### Data Validation
- Input sanitization on all user data
- Booking request uniqueness enforced
- Item status validation before booking
- Message content filtering applied

## Performance Features

### Caching
- Item details cached during session
- Message history preserved locally
- Booking status updates in real-time

### Optimizations
- Lazy loading of non-critical data
- Debounced search functionality
- Efficient state updates
- Minimal re-renders

## Future Enhancements

### Potential Improvements
- Email notifications for booking status changes
- Booking expiration after set time period
- Multiple booking requests per item
- Advanced booking scheduling
- Payment integration
- Review system post-booking