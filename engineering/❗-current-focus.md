# Engineering Priorities: Q1 2026

**This document is a high-level summary. For the detailed, canonical list of MVP priorities, see [ROADMAP-mvp-prioritization.md](./01-proposed/ROADMAP-mvp-prioritization.md).**

---

## 🎉 P0 Complete: All MVP Blockers Resolved!

- **Status:** ✅ **COMPLETED** - January 4, 2026
- **Problem:** The chat system was unusable because users couldn't see who was sending messages ("Unknown User") or what conversations were about (generic "Conversation" titles).
- **Solution:** Implemented frontend enrichment with user and context stores. Messages now show real usernames and conversation titles.
- **Details:** See `../03-completed/FIXLOG-p0-messaging-ux.md`

**All P0 blockers are now resolved! The application is ready for P1 critical items.**

---

## ⏳ Other Critical Priorities (P1)

The following `P1` items are also critical for a successful MVP launch.

- **Backend Filtering for Store Items:** To prevent performance collapse.
- **Backend Testing Foundation:** To reduce regression risk.
- **Transactional Bidding:** To ensure data integrity in auctions.
- **Seller Name Display Bug:** Users cannot see seller names on store items (field mismatch: `full_name` vs `name`).

For a full breakdown of all `P1`, `P2`, and `P3` items, please refer to the [MVP Roadmap](./01-proposed/ROADMAP-mvp-prioritization.md).

---

## 🎉 P2 Complete: Unified Booking & Messaging Flow

- **Status:** ✅ **COMPLETED** - January 4, 2026
- **Problem:** Sellers weren't notified of booking requests and had to manually check each item page. Booking and messaging were fragmented.
- **Solution:** Integrated booking into chat system. Buyers redirected to messages after booking, sellers see requests with approve/decline buttons.
- **Details:** See `03-completed/IMPLEMENTATION-unified-booking-backend.md` and `IMPLEMENTATION-unified-booking-frontend.md`
- **Deployment:** Ready for deployment - see `01-proposed/DEPLOYMENT-CHECKLIST-booking.md`

---

## 📋 Technical Debt: TypeScript Type Errors (P2)

- **Status:** 📋 **TRACKED** - January 4, 2026
- **Problem:** 62 pre-existing TypeScript type errors across 10 frontend files
- **Impact:** Reduced type safety, potential runtime errors, harder maintenance
- **Critical Phase 1 (Before MVP):** Fix ~40 critical errors in App.vue, StoreItemView, TaskDetailView (4-6 hours)
- **Phase 2 (Before Production):** Standardize naming, consolidate types (~22 errors, 10-15 hours)
- **Details:** See `01-proposed/TODO-typescript-errors.md`
- **Root Causes:** Inconsistent snake_case/camelCase, missing null checks, duplicate type definitions

---

## 🎉 P2 Complete: "Book Now" Redirect Context Fix

- **Status:** ✅ **COMPLETED** - January 5, 2026
- **Problem:** After booking a store item, users redirected to `/messages` with no conversation auto-opened
- **Solution:** Pass itemId in redirect URL, Message Center now handles query params to auto-open conversations
- **Impact:** Seamless booking UX - users immediately see their booking conversation with the seller
- **Details:** See `03-completed/FIXLOG-p2-booking-redirect-context.md`
- **Related:** Completes the unified booking & messaging flow

---

## 🎉 P2 Complete: Enhanced Booking Request Flow

- **Status:** ✅ **COMPLETED** - January 5, 2026
- **Problem:** Buyers had no feedback after booking; sellers missed booking requests on individual item pages
- **Solution:**
  - Confirmation modal with embedded chat for buyers
  - Golden badges and priority sorting for sellers in /messages
  - Retry logic for async booking notifications
  - Fixed WebSocket room name bug
- **Impact:**
  - Buyers: Immediate confirmation, ability to message with booking
  - Sellers: Centralized booking view, no missed requests
- **Details:** See `03-completed/FIXLOG-enhanced-booking-flow.md`
- **Related:** Builds on P2 "Unified Booking & Messaging Flow"
