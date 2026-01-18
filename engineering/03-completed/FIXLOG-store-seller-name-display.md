# Fix Log: P1 - Store Item Seller Name Display

**Date:** January 18, 2026
**Priority:** P1
**Status:** ✅ Fixed

## Problem
Users reported that the seller's name was not appearing on Store Item pages. The "View Profile" link was visible, but the name field above it was empty.

**Root Cause:**
Data contract mismatch between the Backend Service and Store Service:
- **Main Backend Service:** Returns user object with `full_name`.
- **Store Service:** Returns user object with `name`.
- **Frontend (`StoreItemView.vue`):** Was expecting `full_name`, resulting in `undefined`.

## Solution
Updated `frontend/src/views/store/StoreItemView.vue` to prioritize `name` while maintaining backward compatibility fallbacks.

**Changes:**
Updated 4 occurrences of name access logic:
1. Seller name display in item details.
2. Bidder name display in bid history.
3. Chat recipient name initialization (Buyer -> Seller).
4. Chat recipient name initialization (Seller -> General Messages).

**Code Pattern Used:**
```javascript
// Before
item.seller.full_name

// After
item.seller.name || item.seller.full_name || item.seller.username
```

## Verification
- Checked `store-service` data models (`internal/models/user.go`) to confirm `Name` field exists and maps to JSON `name`.
- Checked `StoreItemView.vue` to ensure all usages were updated.

## Impact
- Seller names will now correctly display on all store item pages.
- Bidder names will now correctly display in auction histories.
- Chat windows will correctly show the other party's name instead of generic titles or fallbacks.
