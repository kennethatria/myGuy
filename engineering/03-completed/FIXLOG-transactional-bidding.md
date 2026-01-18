# Fix Log: P1 - Transactional Bidding

**Date:** January 18, 2026
**Priority:** P1
**Status:** ✅ Fixed

## Problem
The bidding logic was vulnerable to race conditions. The read-validate-write cycle for placing a bid was not atomic, allowing two users to potentially place conflicting bids or bypass validation checks if requests were processed simultaneously.

## Solution
Implemented database-level locking and transaction management for the bid placement process.

### Changes
1.  **Repository Update:** Added `GetByIDForUpdate` method to `StoreItemRepository`. This uses `SELECT ... FOR UPDATE` (via GORM `clause.Locking{Strength: "UPDATE"}`) to lock the store item row, preventing other concurrent updates.
2.  **Service Refactor:** Updated `StoreService.PlaceBid` to execute within a database transaction (`s.db.Transaction`).
3.  **Dependency Injection:** Injected `*gorm.DB` into `StoreService` to enable transaction management.
4.  **Test Updates:** Updated unit tests to mock the new `GetByIDForUpdate` method and verify logic correctness.

### Technical Details
-   **Locking:** The `StoreItem` row is locked at the beginning of the transaction. Any other transaction trying to read this item for update will wait until the first transaction commits or rolls back.
-   **Atomicity:** The validation (checking current bid), bid creation, and item update now happen atomically.
-   **Testability:** The `PlaceBid` implementation falls back to non-transactional logic if the DB connection is nil, preserving the ability to unit test business logic with mocks.

## Verification
-   Unit tests in `store-service/internal/services` passed (`go test ./internal/services/...`).
-   Code review confirms usage of `clause.Locking{Strength: "UPDATE"}` and `db.Transaction`.
