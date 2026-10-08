// Key actions reported to Umami as custom events (its Events page).  Only an
// event name and a few fixed options: never ids, text or contact details.
// Without Umami (no VITE_UMAMI_WEBSITE_ID, blocked script) nothing happens.

export type AnalyticsEvent =
  | 'sign-in'
  | 'sign-up'
  | 'gig-posted'
  | 'gig-applied'
  | 'gig-status'
  | 'application-answered'
  | 'listing-posted'
  | 'request-posted'
  | 'item-booked'
  | 'booking-step'
  | 'review-left'
  | 'post-reposted'

type EventData = Record<string, string | boolean>

type UmamiWindow = Window & {
  umami?: { track: (event: string, data?: EventData) => void }
}

export function trackEvent(event: AnalyticsEvent, data?: EventData): void {
  try {
    ;(window as UmamiWindow).umami?.track(event, data)
  } catch {
    // Analytics must never break an action that already succeeded
  }
}
