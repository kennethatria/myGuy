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
