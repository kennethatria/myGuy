# FIXLOG: Booking Review UI Issues & Booking Flow Improvements

## Issue: "Submit Review" Button Requires Page Refresh
**Date:** 2026-01-06
**Status:** Fixed

### Symptoms
Users reported that the "Submit Review" button on the task review page would not function (or appear to function) unless the page was refreshed.

### Root Cause Analysis
The issue was likely caused by reactivity state management in the `ReviewForm` component or its parent `CreateReviewView`. 
1. **Component Reuse:** Vue might have been reusing the `ReviewForm` component without properly resetting its internal state when the task context was established.
2. **State Initialization:** The component's `loading` or internal validation state might have been stale if navigation occurred without a full mount cycle.

### Fix Implementation
1. **Force Re-mount:** Added `:key="taskId"` to the `ReviewForm` component in `CreateReviewView.vue`. This forces Vue to destroy and recreate the component whenever the task ID is present, ensuring a completely fresh state (form fields empty, loading false, listeners fresh).
2. **Logging:** Added console logging to the `handleSubmit` function in `ReviewForm.vue` to aid in future debugging of submission flows.

---

## Identified Potential UX Issues in Booking Flow

During the investigation, several other issues affecting the booking experience were identified. These should be addressed to improve the professional feel of the application.

### 1. Blocking Native Alerts (High Priority)
The application currently uses `window.alert()` and `window.confirm()` for critical user interactions in `TaskDetailView.vue`.
- **Impact:** Native alerts block the browser's main thread, offer a poor visual experience that doesn't match the app's UI, and can't be styled.
- **Locations:**
  - Application submission success/failure (`handleApplicationSubmit`)
  - Task completion confirmation (`handleComplete`)
  - Application acceptance/decline (`handleAcceptApplication`, `handleDeclineApplication`)
- **Recommendation:** Replace with a dedicated Toast/Notification system or custom Modals.

### 2. Lack of Optimistic UI Updates
When accepting or declining an application, the UI waits for the API response before updating.
- **Impact:** The interface feels sluggish.
- **Recommendation:** Implement optimistic updates (update the UI immediately, revert if API fails) for smoother interaction.

### 3. Review Form Feedback
The review form relies on button text change ("Submitting...") for feedback.
- **Impact:** If the network is fast, the user might miss the feedback. If it's slow, they might think it's stuck.
- **Recommendation:** Add a clear progress indicator or disable the form fields during submission to prevent double-submission.

### 4. Navigation Feedback
After submitting a review, the user is redirected to the task detail page via `router.push`.
- **Impact:** There is no "Review Submitted Successfully" message shown on the destination page.
- **Recommendation:** Pass a state or query parameter to the destination page to show a success toast upon arrival.
