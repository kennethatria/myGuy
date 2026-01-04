# RFC: Unified Booking & Messaging Flow

## 1. Context & Current State
Currently, the "Book Now" functionality for store items operates independently of the messaging system.

### The Current Flow
1.  **Buyer:** Clicks "Book Now" on an item page.
    *   This calls `POST store-service/items/:id/booking-request`.
    *   The request is stored in the `store_db` with a status of `pending`.
2.  **Seller:**
    *   Must manually visit the specific item page (`/store/item/:id`).
    *   Sees a "Booking Requests" section if they are the owner.
    *   Can "Approve" or "Decline" the request via buttons.
3.  **Post-Action:**
    *   If approved, a "Message {User}" button appears, allowing the seller to manually start a chat.

### Identified Issues
*   **Poor Visibility:** Sellers are not notified of new booking requests. They must proactively check every item listing.
*   **Fragmented Experience:** The negotiation (booking) and communication (chat) happen in two different places, even though they are logically the same interaction.
*   **Scalability:** As a seller lists more items, managing bookings by visiting individual item pages becomes impossible.
*   **Mobile Unfriendly:** The current complex item page UI with management controls is dense and difficult to manage on mobile devices.

## 2. Proposed Solution
We propose making the **Chat Service (`/messages`)** the single source of truth for all user-to-user interactions, including booking requests.

### The New Flow
1.  **Buyer:** Clicks "Book Now" on an item page.
    *   **Backend Action:** `store-service` creates the booking record AND triggers a notification to `chat-service`.
    *   **User Experience:** The user is immediately redirected to a chat conversation with the seller. A system message appears: *"I have sent a booking request for {Item Name}."*
2.  **Seller:**
    *   Receives a new message notification (badge/push).
    *   Opens the chat and sees a **Structured System Message**.
    *   **UI:** The message contains "Approve" and "Decline" buttons directly within the chat bubble.
3.  **Action:**
    *   Seller clicks "Approve".
    *   **Backend Action:** `chat-service` calls `store-service` to update status.
    *   **Feedback:** The chat updates to show "Booking Approved ✅". A new system message is added: *"Booking approved. You can now discuss pickup details."*

## 3. Technical Implementation Plan

### Phase 1: Service Integration (Backend)
*   **Inter-Service Communication:** Implement a mechanism for `store-service` to notify `chat-service`.
    *   *Initial MVP:* HTTP call from `store-service` to a secured internal endpoint on `chat-service`.
    *   *Robust:* Message queue (RabbitMQ/Redis) for async events (`BookingCreated`).
*   **System Messages:** Update `chat-service` schema to support `message_type` (e.g., `text`, `booking_request`, `system_alert`).

### Phase 2: Frontend Updates
*   **Chat Component:** Update `ChatWindow.vue` to render "Booking Request" messages differently.
    *   Needs to display Item info (Title, Image).
    *   Needs to display Action Buttons (Approve/Decline).
*   **Store Item Page:** 
    *   Change "Book Now" behavior to redirect to chat after success.
    *   Remove the "Booking Management" section from the item page eventually.

### Phase 3: Migration & Cleanup
*   **Dual Run:** Initially, keep the item page management as a fallback.
*   **Deprecation:** Once the chat flow is stable, remove the management UI from the item page.

## 4. Risks & Mitigations
*   **Service Coupling:** `store-service` becoming dependent on `chat-service` uptime.
    *   *Mitigation:* Use async messaging or ensure the booking can succeed even if the chat notification fails (and retry later).
*   **Complexity:** Managing state sync between the chat bubble UI and the actual booking status in `store_db`.
    *   *Mitigation:* The chat message should purely be a view; clicking the button should query `store-service` for the current source-of-truth status.

## 5. Roadmap Recommendation
This initiative should be added to the **MVP Roadmap**. The current "silent booking" failure mode (users booking and never getting a response because the seller didn't check the specific item page) is a critical UX flaw that will prevent platform adoption.
