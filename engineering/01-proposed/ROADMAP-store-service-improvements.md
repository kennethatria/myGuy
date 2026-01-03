# Roadmap: Store Service Improvements

This document outlines potential future improvements for the `store-service` to enhance its architecture, reliability, performance, and feature set.

## 1. Architectural & Design Improvements

-   **Decouple User Data with an Event-Driven Architecture**
    -   **Current State:** The service duplicates user data from JWTs into a local table to solve foreign key constraints.
    -   **Potential Issue:** The local user cache can become stale if user details (e.g., username) are updated in the main `backend` service.
    -   **Suggestion:** Implement an event-driven approach. When a user's profile is updated, the main service should publish a `user.updated` event. The `store-service` can subscribe to these events to keep its local user table synchronized, ensuring long-term data consistency.

## 2. Logic & Reliability Enhancements

-   **Implement Transactional Bidding**
    -   **Current State:** The bidding logic (checking the current bid and inserting a new one) may not be atomic.
    -   **Potential Issue:** A race condition could occur if two users place a valid bid simultaneously, potentially leading to incorrect bid acceptance.
    -   **Suggestion:** Ensure the entire bid placement process (read, validate, write) is wrapped in a single database transaction with a `SERIALIZABLE` or `REPEATABLE READ` isolation level to guarantee atomicity.

-   **Implement a Background Job for Auction Expiration**
    -   **Current State:** The mechanism for expiring auctions is not explicitly defined. If it relies on being triggered by user queries, it can lead to inconsistent state.
    -   **Suggestion:** Create a scheduled background job (cron job) that runs periodically (e.g., every minute) to find and update the status of items where the `bid_deadline` has passed.

-   **Implement Soft Deletes for Items**
    -   **Current State:** Deleting an item appears to perform a hard delete from the database.
    -   **Suggestion:** Switch to a soft-delete pattern by using a `deleted_at` timestamp. This preserves data history, prevents accidental data loss, allows for "undo" functionality, and is better for data auditing. GORM has built-in support for this.

## 3. Performance Optimizations

-   **Comprehensive Database Indexing**
    -   **Current State:** The `GET /items` endpoint supports many filter parameters, but not all are necessarily indexed.
    -   **Suggestion:** Add composite database indexes for common filter combinations to prevent performance degradation as the data grows. For example:
        -   `CREATE INDEX idx_items_filters ON store_items (category, status, price_type);`
        -   For text search, implement PostgreSQL's full-text search (`tsvector` and `tsquery`) for much faster and more relevant results than `ILIKE`.

-   **Offload Image Serving to a CDN**
    -   **Current State:** The Go service serves static image files directly from its filesystem.
    -   **Suggestion:** For any production environment, offload image delivery to a dedicated Content Delivery Network (CDN) backed by an object storage service (like AWS S3). This drastically improves performance, reduces load on the application server, and provides better scalability and caching.

## 4. Feature Enhancements

-   **Outbid Notifications**
    -   **Current State:** Users are not notified when they are outbid on an auction.
    -   **Suggestion:** When a new valid bid is placed, the service should publish a `user.outbid` event containing the `previous_highest_bidder_id`. A separate notification service can then consume this event and send a real-time alert (via WebSocket, push notification, or email) to the outbid user.

-   **Advanced Auction Features**
    -   **Current State:** The auction model is basic.
    -   **Suggestion:** To create a more engaging auction experience, implement:
        -   **Proxy Bidding:** Allow users to set a maximum bid, and have the system automatically place the minimum required bid on their behalf until their maximum is reached.
        -   **Anti-Sniping:** Automatically extend the auction deadline by 1-2 minutes if a bid is placed in the final moments.

-   **Inventory Management**
    -   **Current State:** The `StoreItem` model does not appear to have a `quantity` field.
    -   **Suggestion:** Add a `quantity` field to `StoreItem`. For fixed-price items, a purchase should decrement the quantity. The item should only be marked as `sold` when the quantity reaches zero. This allows sellers to list multiple units of the same product.
