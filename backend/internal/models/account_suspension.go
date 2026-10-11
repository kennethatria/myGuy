package models

import "time"

// AccountSuspension makes an account read-only for a while: it can sign in
// and look around, but not post, apply, book, review or message, and its
// posts are hidden.  Keyed by account (unlike a block, which is by email).
// Rows are kept after they end: they set the next suspension's length and
// the profile mark.
type AccountSuspension struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index;not null"`
	Reason    string
	StartsAt  time.Time  `gorm:"not null"`
	EndsAt    time.Time  `gorm:"index;not null"`
	LiftedAt  *time.Time // lifted early by hand
	CreatedBy string
}

// End is when the suspension stopped (or will): lifted early, or its end.
func (s AccountSuspension) End() time.Time {
	if s.LiftedAt != nil && s.LiftedAt.Before(s.EndsAt) {
		return *s.LiftedAt
	}
	return s.EndsAt
}

// ActiveAt reports whether the suspension applies at now.
func (s AccountSuspension) ActiveAt(now time.Time) bool {
	return !s.StartsAt.After(now) && s.End().After(now)
}
