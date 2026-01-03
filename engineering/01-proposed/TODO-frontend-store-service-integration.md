# TODO: Fix Frontend and Backend Integration for Store Service

**Status:** Proposed
**Priority:** 🔴 Critical

This document outlines critical issues found in the frontend implementation of the store/marketplace functionality. These issues prevent core features from working correctly and must be addressed to have a functional store service.

## 1. Fix Inefficient Client-Side Filtering

-   **File:** `frontend/src/views/store/StoreView.vue`
-   **Function:** `loadItems()` and `filteredItems()` computed property.

-   **Issue:** The current implementation fetches *all* items from the backend API (`GET /api/v1/items`) and then applies search, category, and condition filters on the client side. This is not scalable and will lead to severe performance degradation and high memory usage in the browser as the number of items increases.

-   **Required Action:**
    1.  Modify the `loadItems()` function to accept filter parameters (search query, category, condition).
    2.  When a filter or search query changes, call `loadItems()` again, passing the new parameters as query strings in the API request.
    3.  Example API call: `GET /api/v1/items?search=camera&category=electronics`.
    4.  Remove the `filteredItems` computed property, as the filtering will now be handled by the backend. The `items` ref should be directly populated with the filtered results from the API.

## 2. Implement Missing Pagination and Sorting UI

-   **File:** `frontend/src/views/store/StoreView.vue`

-   **Issue:** The backend `GET /api/v1/items` endpoint supports pagination (`page`, `per_page`) and sorting (`sort_by`, `sort_order`), but the frontend has no UI controls for these features and does not send these parameters.

-   **Required Action:**
    1.  Add UI elements (e.g., dropdowns for sorting, a pagination component) to the `StoreView.vue` template.
    2.  Add state variables (`ref`s) to hold the current page, items per page, and sort order.
    3.  Include these state variables as query parameters in the `loadItems()` API call.
    4.  The `loadItems()` function should update the total item count from the API response to correctly render the pagination controls.

## 3. Fix Critical Bug in "Create Item" Payload

-   **Status:** **RESOLVED** (Upon re-examination, the `createItem` function already correctly maps fields as per backend contract.)

-   **File:** `frontend/src/views/store/StoreView.vue`

-   **Function:** `createItem()`

-   **Issue:** The form for creating a new item constructs a JSON payload with field names that **do not match** the backend API's expectations. This will cause all item creation attempts to fail validation on the backend.

-   **Required Action:**
    -   Modify the `jsonPayload` object in the `createItem` function to use the correct field names as expected by the backend.

    | Incorrect Frontend Field | Correct Backend Field |
    | :--- | :--- |
    | `is_auction` (boolean) | `price_type` (string: "fixed" or "bidding") |
    | `price` | `fixed_price` |
    | `bid_increment` | `min_bid_increment` |

## 4. Remove All Hardcoded URLs

-   **Files:** `frontend/src/views/store/StoreView.vue`, `frontend/src/views/store/StoreItemView.vue`

-   **Issue:** API calls and image `src` attributes are constructed using hardcoded `http://localhost:xxxx` URLs. This prevents the application from functioning in any deployed (staging, production) or different local environment.

-   **Required Action:**
    1.  Import the `config` object from `frontend/src/config.ts` in every component that makes an API call.
    2.  Replace all hardcoded URLs with the appropriate variables from the `config` object.
    3.  **Example (API call):**
        ```javascript
        // Incorrect
        await fetch('http://localhost:8081/api/v1/items', ...);

        // Correct
        import config from '@/config';
        await fetch(`${config.STORE_API_URL}/items`, ...);
        ```
    4.  **Example (Image URL):**
        ```html
        <!-- Incorrect -->
        <img :src="'http://localhost:8081' + item.images[0].url" ... />

        <!-- Correct -->
        <img :src="config.STORE_API_BASE_URL + item.images[0].url" ... />
        <!-- Note: You will need to add STORE_API_BASE_URL=http://localhost:8081 to your config.ts -->
        ```

This list represents the critical fixes needed to make the store functionality operational.
