# Investigation: "View Seller" Link Issues (Issue 2.2)

## Overview
This document details the investigation into Issue 2.2, where the "View Seller" link in the Store displays an incorrect profile or label. The issue report states that the profile shown does not belong to the item owner and the label is generic.

## Findings

### 1. Incorrect Label / Missing Name
*   **Observation:** The seller's name is not displayed in the "Seller" section of `StoreItemView.vue`.
*   **Root Cause:** Field mismatch between the API response and the Frontend code.
    *   **Store Service API:** The `User` model in `store-service` serializes the name field as `name`.
        ```go
        // store-service/internal/models/user.go
        Name string `json:"name"`
        ```
    *   **Frontend:** The `StoreItemView.vue` component attempts to access `full_name`.
        ```html
        <!-- frontend/src/views/store/StoreItemView.vue -->
        <span class="seller-name">{{ item.seller.full_name }}</span>
        ```
    *   **Result:** `item.seller.full_name` is `undefined`, so the name span is empty. The user only sees the "View Profile" link below an empty name.

### 2. "View Seller" Link Label
*   **Observation:** The link text is hardcoded as "View Profile".
    ```html
    <router-link ...>View Profile</router-link>
    ```
*   **Context:** The issue report calls it "generic".
*   **Recommendation:** If the design intent is to show the name *as* the link, the code should be updated to wrap the name in the link, or change the link text to something like "View [Name]'s Profile".

### 3. "Incorrect Profile" (Wrong User)
*   **Observation:** The issue states the profile shown does not belong to the actual item owner.
*   **Investigation:**
    *   The `StoreItem` model has a `SellerID` field which is a Foreign Key to the `users` table in the `store-service` database.
    *   The `store-service` synchronizes users from the main backend via JWT claims in the `UpsertFromJWT` middleware function.
    *   It explicitly sets the `ID` of the user record to match the `UserID` from the JWT (which comes from the main backend).
*   **Hypothesis:** The mismatch is likely due to **data inconsistency in the test/development environment**, specifically regarding seed data.
    *   If seed data for the Store Service was created with hardcoded `SellerID`s (e.g., 1, 2, 3) that do not match the current state of the Backend's `users` table (e.g., if Backend users were reset and re-created with new IDs), then the `SellerID` in the Store Item will point to a different (or non-existent) user profile.
    *   Example: Item has `SellerID: 1`. Backend User with ID 1 is "Admin". Actual Item Creator was "Alice" (who now has ID 5). User sees "Admin" profile.
*   **Mitigation:** Ensure seed data scripts for Store and Backend are synchronized and use consistent user IDs, or dynamically fetch IDs during seeding.

## Recommendations

1.  **Fix Name Display:** Update `StoreItemView.vue` to use `item.seller.name` instead of `item.seller.full_name`. Add a fallback to `username` if name is missing.
    ```javascript
    {{ item.seller.name || item.seller.username }}
    ```

2.  **Verify Data Integrity:** Re-seed the database ensuring that Store Items are created by users that actually exist in the Backend with matching IDs.

3.  **Enhance Link Label:** Consider changing "View Profile" to "View [Name]'s Profile" or making the name itself the link to improve clarity.

## Conclusion
The "generic label" issue is a definite code bug (`full_name` vs `name`). The "wrong profile" issue is a data integrity artifact resulting from the decoupled nature of the services and likely inconsistent seeding.
