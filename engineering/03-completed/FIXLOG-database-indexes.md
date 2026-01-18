# Fix Log: Database Indexes for Performance

**Date:** January 18, 2026
**Priority:** P2 (Recommended Before Launch)
**Status:** ✅ COMPLETED

## Problem

The database was missing key indexes on foreign key columns and common query filters. This could lead to:
- Full table scans on JOIN operations
- Slow filtering queries as data grows
- Performance degradation under moderate load

This was identified as part of the "Address Backend Scalability & Performance" item in the MVP roadmap.

## Solution

Added comprehensive indexes to all three service databases using GORM struct tags (Go services) and SQL migrations (Node.js service).

---

## Analysis Summary

### Chat Service (my_guy_chat)
**Status:** Already well-indexed ✓

The chat service already had comprehensive indexes defined in `migrations/001_chat_schema.sql`:
- Foreign key indexes: `sender_id`, `recipient_id`, `task_id`, `application_id`, `store_item_id`
- Query pattern indexes: `created_at`, `message_type`
- Composite indexes for common queries
- Partial indexes for sparse columns

**No changes needed.**

### Backend (my_guy) - Indexes Added

#### Tasks Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| status | `idx_tasks_status` | Filter by task status (open/in_progress/completed/cancelled) |
| created_by | `idx_tasks_created_by` | Foreign key to users, list user's created tasks |
| assigned_to | `idx_tasks_assigned_to` | Foreign key to users, list assigned tasks |
| fee | `idx_tasks_fee` | Range queries on task price |
| deadline | `idx_tasks_deadline` | Filter/sort by deadline |
| created_at | `idx_tasks_created_at` | Default sort order (newest first) |

#### Applications Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| task_id | `idx_applications_task_id` | Foreign key, list applications for a task |
| applicant_id | `idx_applications_applicant_id` | Foreign key, list user's applications |
| status | `idx_applications_status` | Filter pending/accepted/declined |

#### Reviews Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| task_id | `idx_reviews_task_id` | Foreign key, get reviews for a task |
| reviewer_id | `idx_reviews_reviewer_id` | Foreign key, get reviews by a user |
| reviewed_user_id | `idx_reviews_reviewed_user_id` | Foreign key, get reviews for a user |
| (task_id, reviewer_id) | `idx_reviews_task_reviewer` | **Unique composite** - prevent duplicate reviews |

### Store Service (my_guy_store) - Indexes Added

#### Store Items Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| seller_id | `idx_store_items_seller_id` | Foreign key, seller's listings |
| buyer_id | `idx_store_items_buyer_id` | Foreign key, buyer's purchases |
| status | `idx_store_items_status` | Filter active/sold/expired/cancelled |
| category | `idx_store_items_category` | Filter by category |
| price_type | `idx_store_items_price_type` | Filter fixed/bidding |
| condition | `idx_store_items_condition` | Filter by item condition |
| fixed_price | `idx_store_items_fixed_price` | Price range queries |
| bid_deadline | `idx_store_items_bid_deadline` | Auction expiration queries |
| created_at | `idx_store_items_created_at` | Default sort order |

#### Item Images Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| item_id | `idx_item_images_item_id` | Foreign key, get images for an item |

#### Bids Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| item_id | `idx_bids_item_id` | Foreign key, get bids for an item |
| bidder_id | `idx_bids_bidder_id` | Foreign key, get user's bids |
| status | `idx_bids_status` | Filter active/outbid/won bids |
| created_at | `idx_bids_created_at` | Sort bids chronologically |

#### Booking Requests Table

| Column | Index Name | Purpose |
|--------|------------|---------|
| item_id | `idx_booking_requests_item_id` | Foreign key, get bookings for an item |
| requester_id | `idx_booking_requests_requester_id` | Foreign key, get user's booking requests |
| status | `idx_booking_requests_status` | Filter by booking status |
| chat_notified | `idx_booking_requests_chat_notified` | Find failed notification retries |
| created_at | `idx_booking_requests_created_at` | Sort by date |

---

## Implementation Details

### GORM Index Tags

Indexes are defined using GORM struct tags in the model files. Example:

```go
type Task struct {
    ID        uint   `gorm:"primaryKey"`
    Status    string `gorm:"default:'open';index:idx_tasks_status"`
    CreatedBy uint   `gorm:"not null;index:idx_tasks_created_by"`
    // ...
}
```

### Automatic Migration

GORM's `AutoMigrate()` automatically creates these indexes on service startup:

```go
db.AutoMigrate(&models.Task{}, &models.Application{}, &models.Review{})
```

No manual SQL migration scripts are required.

---

## Files Modified

| File | Changes |
|------|---------|
| `backend/internal/models/task.go` | Added index tags to Task, Application, Review structs |
| `store-service/internal/models/store_item.go` | Added index tags to StoreItem, ItemImage, Bid, BookingRequest structs |

---

## Query Patterns Optimized

### Backend Queries

| Query Pattern | Index Used |
|---------------|------------|
| `WHERE status = 'open'` | `idx_tasks_status` |
| `WHERE created_by = ?` | `idx_tasks_created_by` |
| `WHERE assigned_to = ?` | `idx_tasks_assigned_to` |
| `WHERE fee >= ? AND fee <= ?` | `idx_tasks_fee` |
| `ORDER BY created_at DESC` | `idx_tasks_created_at` |
| `WHERE task_id = ?` (applications) | `idx_applications_task_id` |
| `WHERE task_id = ? AND reviewer_id = ?` | `idx_reviews_task_reviewer` (unique) |

### Store Service Queries

| Query Pattern | Index Used |
|---------------|------------|
| `WHERE status = 'active'` | `idx_store_items_status` |
| `WHERE seller_id = ?` | `idx_store_items_seller_id` |
| `WHERE category = ?` | `idx_store_items_category` |
| `WHERE price_type = 'bidding' AND bid_deadline < ?` | `idx_store_items_bid_deadline` |
| `WHERE item_id = ? AND status = 'active'` (bids) | `idx_bids_item_id`, `idx_bids_status` |
| `WHERE chat_notified = false` | `idx_booking_requests_chat_notified` |

---

## Deployment Notes

1. **No downtime required** - GORM creates indexes in the background
2. **First startup after deployment** will create the indexes automatically
3. **Large tables** - Index creation may take a few seconds on tables with existing data
4. **Verify indexes** - After deployment, verify indexes exist:

```sql
-- PostgreSQL
SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'tasks';
SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'store_items';
```

---

## Performance Impact

**Expected Improvements:**
- JOIN operations (task → creator, item → seller) now use indexed foreign keys
- Filter queries (by status, category, etc.) avoid full table scans
- Pagination queries benefit from sorted indexes
- Unique constraint on reviews prevents duplicate reviews at database level

**Trade-offs:**
- Slightly slower INSERT/UPDATE operations (index maintenance)
- Increased storage for index data structures
- These trade-offs are acceptable for read-heavy workloads typical of marketplaces

---

## Related Documentation

- `engineering/01-proposed/ROADMAP-mvp-prioritization.md` - P2 item "Address Backend Scalability & Performance"
- `engineering/01-proposed/ROADMAP-store-service-improvements.md` - Database index recommendations
- `chat-websocket-service/migrations/001_chat_schema.sql` - Chat service index definitions

---

**Document Version:** 1.0
**Created:** January 18, 2026
