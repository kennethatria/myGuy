package models

import "time"

// BlockedEmail keeps an address out of the app: it can't sign in or sign up,
// its sessions are refused, and its posts are hidden.  Keyed by the
// normalized email, so a blocked person can't come back with a new account
// on the same address.  Until nil means for good.
type BlockedEmail struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	Reason    string
	Until     *time.Time `gorm:"column:blocked_until"`
	CreatedBy string
	CreatedAt time.Time
}

// ActiveAt reports whether the block still applies at now.
func (b BlockedEmail) ActiveAt(now time.Time) bool {
	return b.Until == nil || b.Until.After(now)
}
