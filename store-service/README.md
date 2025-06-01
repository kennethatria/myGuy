# Store Service - MyGuy Marketplace

A microservice for the MyGuy platform that provides a comprehensive marketplace where users can list items for sale with either fixed prices or auction-style bidding.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [API Documentation](#api-documentation)
- [Data Models](#data-models)
- [Business Logic](#business-logic)
- [Configuration](#configuration)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)

## Features

### Item Management
- **Create Listings**: Post items with detailed information, images, and pricing options
- **Update Listings**: Edit active items (title, description, images, etc.)
- **Delete Listings**: Remove items from the marketplace
- **Image Support**: Multiple images per item stored as array
- **Categories**: Organize items by type (electronics, collectibles, etc.)
- **Condition Tracking**: New, like-new, good, fair, poor

### Pricing Models
- **Fixed Price**: Immediate purchase at set price
- **Bidding/Auction**: 
  - Starting bid configuration
  - Minimum bid increments
  - Bid deadlines with automatic expiration
  - Current bid tracking

### User Features
- **Seller Dashboard**: View all listings, sales history
- **Buyer Dashboard**: Track purchases and active bids
- **Bid Management**: Place, track, and monitor bid status
- **Purchase History**: Complete transaction records

### Advanced Features
- **Search & Filter**: Full-text search with multiple filter options
- **Sorting**: By price, date, title
- **Pagination**: Efficient data loading
- **Status Management**: Active, sold, expired, cancelled
- **Automatic Expiration**: Bid items expire at deadline

## Architecture

### Technology Stack
- **Language**: Go 1.21+
- **Framework**: Gin HTTP framework
- **Database**: PostgreSQL with GORM ORM
- **Authentication**: JWT middleware (shared with main service)
- **Architecture Pattern**: Clean Architecture

### Project Structure
```
store-service/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── models/
│   │   ├── store_item.go        # Item and bid models
│   │   └── user.go              # User model (simplified)
│   ├── repositories/
│   │   ├── interfaces.go        # Repository contracts
│   │   ├── store_item_repository.go
│   │   └── bid_repository.go
│   ├── services/
│   │   └── store_service.go     # Business logic
│   ├── api/
│   │   └── handlers/
│   │       └── store_handlers.go # HTTP handlers
│   └── middleware/
│       └── jwt.go               # Auth middleware
├── migrations/
├── Dockerfile
├── go.mod
└── README.md
```

## API Documentation

### Base URL
```
http://localhost:8081/api/v1
```

### Authentication
All protected endpoints require JWT token in Authorization header:
```
Authorization: Bearer <token>
```

### Public Endpoints

#### GET /items
Browse all items with filtering and pagination.

**Query Parameters:**
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| search | string | Search in title/description | `vintage camera` |
| category | string | Filter by category | `electronics` |
| price_type | string | `fixed` or `bidding` | `fixed` |
| condition | string | Item condition | `like-new` |
| min_price | float | Minimum price | `50.00` |
| max_price | float | Maximum price | `500.00` |
| seller_id | uint | Filter by seller | `123` |
| status | string | Item status | `active` |
| sort_by | string | Sort field | `price` |
| sort_order | string | `asc` or `desc` | `asc` |
| page | int | Page number | `1` |
| per_page | int | Items per page | `20` |

**Response:**
```json
{
  "items": [
    {
      "id": 1,
      "title": "Vintage Camera",
      "description": "Professional camera",
      "seller_id": 123,
      "price_type": "fixed",
      "fixed_price": 299.99,
      "status": "active",
      "category": "electronics",
      "images": ["url1", "url2"],
      "condition": "good",
      "location": "New York, NY",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "per_page": 20
}
```

#### GET /items/:id
Get detailed information about a specific item.

**Response:**
```json
{
  "id": 1,
  "title": "Vintage Camera",
  "description": "Professional camera in excellent condition",
  "seller_id": 123,
  "price_type": "fixed",
  "fixed_price": 299.99,
  "status": "active",
  "category": "electronics",
  "images": ["url1", "url2"],
  "condition": "good",
  "location": "New York, NY",
  "shipping_info": "Free shipping within US",
  "created_at": "2024-01-15T10:00:00Z",
  "bids": []
}
```

#### GET /items/:id/bids
View all bids for an auction item.

**Response:**
```json
[
  {
    "id": 1,
    "item_id": 1,
    "bidder_id": 456,
    "amount": 55.00,
    "message": "Very interested!",
    "status": "active",
    "created_at": "2024-01-15T11:00:00Z"
  }
]
```

### Protected Endpoints

#### POST /items
Create a new item listing.

**Request Body (Fixed Price):**
```json
{
  "title": "Vintage Camera",
  "description": "Professional camera in excellent condition",
  "price_type": "fixed",
  "fixed_price": 299.99,
  "category": "electronics",
  "images": ["url1", "url2"],
  "condition": "good",
  "location": "New York, NY",
  "shipping_info": "Free shipping within US"
}
```

**Request Body (Bidding):**
```json
{
  "title": "Rare Collectible Card",
  "description": "Limited edition trading card",
  "price_type": "bidding",
  "starting_bid": 50.00,
  "min_bid_increment": 5.00,
  "bid_deadline": "2024-01-20T23:59:59Z",
  "category": "collectibles",
  "images": ["url1"],
  "condition": "like-new",
  "location": "Los Angeles, CA",
  "shipping_info": "Insured shipping"
}
```

#### PUT /items/:id
Update an existing item (owner only).

**Request Body:**
```json
{
  "title": "Updated Title",
  "description": "Updated description",
  "category": "new-category",
  "images": ["new-url1", "new-url2"],
  "condition": "good",
  "location": "Updated location",
  "shipping_info": "Updated shipping info"
}
```

**Note:** Cannot update price type, prices, or bid settings after creation.

#### DELETE /items/:id
Delete an item (owner only, active items only).

#### POST /items/:id/purchase
Purchase a fixed-price item.

**Response:**
```json
{
  "message": "item purchased successfully"
}
```

#### POST /items/:id/bids
Place a bid on an auction item.

**Request Body:**
```json
{
  "amount": 75.00,
  "message": "Great item, very interested!"
}
```

**Validation:**
- Amount must be >= starting bid (if no bids)
- Amount must be >= current bid + min increment
- Item must be active and not expired
- Cannot bid on own items

#### POST /items/:id/bids/:bidId/accept
Accept a bid and complete the sale (seller only).

#### GET /user/listings
Get all items listed by the authenticated user.

#### GET /user/purchases
Get all items purchased by the authenticated user.

#### GET /user/bids
Get all bids placed by the authenticated user.

## Data Models

### StoreItem
```go
type StoreItem struct {
    ID              uint           `gorm:"primaryKey"`
    Title           string         `gorm:"not null"`
    Description     string         
    SellerID        uint           `gorm:"not null"`
    PriceType       string         // "fixed" or "bidding"
    FixedPrice      float64        
    StartingBid     float64        
    CurrentBid      float64        
    MinBidIncrement float64        
    BidDeadline     *time.Time     
    Status          string         // active, sold, expired, cancelled
    Category        string         
    Images          []string       `gorm:"type:text[]"`
    Condition       string         // new, like-new, good, fair, poor
    Location        string         
    ShippingInfo    string         
    BuyerID         *uint          
    SoldAt          *time.Time     
    Bids            []Bid          `gorm:"foreignKey:ItemID"`
    CreatedAt       time.Time      
    UpdatedAt       time.Time      
    DeletedAt       gorm.DeletedAt `gorm:"index"`
}
```

### Bid
```go
type Bid struct {
    ID        uint           `gorm:"primaryKey"`
    ItemID    uint           `gorm:"not null"`
    BidderID  uint           `gorm:"not null"`
    Amount    float64        `gorm:"not null"`
    Message   string         
    Status    string         // active, outbid, won, cancelled
    CreatedAt time.Time      
    UpdatedAt time.Time      
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

## Business Logic

### Item Creation Rules
1. Fixed price items require `fixed_price > 0`
2. Bidding items require:
   - `starting_bid > 0`
   - `min_bid_increment > 0` (defaults to 1.0)
   - `bid_deadline` must be future date

### Bidding Rules
1. First bid must be >= starting bid
2. Subsequent bids must be >= current bid + min increment
3. Cannot bid on own items
4. Cannot bid on expired/sold items
5. Accepting a bid:
   - Marks item as sold
   - Sets buyer_id
   - Updates bid status to "won"
   - Marks other bids as "outbid"

### Status Transitions
- `active` → `sold` (via purchase or bid acceptance)
- `active` → `expired` (automatic at bid deadline)
- `active` → `cancelled` (via delete)

### Automatic Processes
- Bid expiration checked on each query
- Items past bid_deadline automatically marked as expired

## Configuration

### Environment Variables
```env
# Server Configuration
PORT=8081

# Database
DB_CONNECTION=host=postgres user=postgres password=password dbname=myguy port=5432 sslmode=disable

# Authentication (must match main service)
JWT_SECRET=your-secret-key-here

# Logging
LOG_LEVEL=info
```

### Database Configuration
- Connection pool: Max 20 connections
- Idle timeout: 30 seconds
- Auto-migration on startup

## Development

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 12+
- Docker (optional)

### Local Setup
```bash
# Clone repository
cd store-service

# Install dependencies
go mod download

# Create .env file
cp .env.example .env
# Edit .env with your configuration

# Run service
go run cmd/api/main.go
```

### Docker Development
```bash
# Build image
docker build -t store-service .

# Run with docker-compose
docker-compose up store-service
```

### Database Migrations
The service automatically runs migrations on startup using GORM AutoMigrate.

## Testing

### Unit Tests
```go
// Example test structure
func TestStoreService_CreateItem(t *testing.T) {
    // Setup
    mockRepo := &MockStoreItemRepository{}
    service := NewStoreService(mockRepo, nil)
    
    // Test fixed price item
    req := CreateStoreItemRequest{
        Title:      "Test Item",
        PriceType:  "fixed",
        FixedPrice: 100.00,
    }
    
    item, err := service.CreateItem(1, req)
    assert.NoError(t, err)
    assert.Equal(t, "Test Item", item.Title)
}
```

### Integration Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

### API Testing
```bash
# Create item
curl -X POST http://localhost:8081/api/v1/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Item",
    "price_type": "fixed",
    "fixed_price": 99.99
  }'

# Place bid
curl -X POST http://localhost:8081/api/v1/items/1/bids \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 75.00
  }'
```

## Deployment

### Production Considerations

1. **Database Indexes**
   ```sql
   CREATE INDEX idx_items_seller ON store_items(seller_id);
   CREATE INDEX idx_items_status ON store_items(status);
   CREATE INDEX idx_items_category ON store_items(category);
   CREATE INDEX idx_bids_item ON bids(item_id);
   CREATE INDEX idx_bids_bidder ON bids(bidder_id);
   ```

2. **Health Checks**
   ```yaml
   healthcheck:
     test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
     interval: 30s
     timeout: 10s
     retries: 3
   ```

3. **Resource Limits**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '1'
         memory: 512M
       reservations:
         cpus: '0.5'
         memory: 256M
   ```

4. **Monitoring**
   - Response times
   - Database connection pool usage
   - Failed bid attempts
   - Expired items cleanup

### Security Considerations
- JWT validation on all protected endpoints
- User authorization checks (can't edit others' items)
- Input validation and sanitization
- SQL injection prevention via GORM
- Rate limiting (recommended)

## Error Handling

### Common Error Responses
```json
{
  "error": "item not found"
}
```

### Error Codes
- `400`: Bad Request (validation errors)
- `401`: Unauthorized (missing/invalid token)
- `403`: Forbidden (not owner)
- `404`: Not Found
- `500`: Internal Server Error

## Future Enhancements
1. Image upload handling
2. Payment integration
3. Shipping label generation
4. Watch lists / favorites
5. Item recommendations
6. Price history tracking
7. Bulk operations
8. Export functionality

## Contributing
1. Fork the repository
2. Create feature branch
3. Commit changes
4. Push to branch
5. Create Pull Request

## License
Part of the MyGuy platform.