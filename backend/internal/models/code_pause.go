package models

import "time"

// CodePause stops sign-in codes going to an address for a while: for an
// address someone is flooding with codes (email bombing), to protect its
// owner and the mail sender's reputation.  Nothing else changes: the
// account, if any, stays signed in, and a code already sent still works.
type CodePause struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Until     time.Time `gorm:"column:paused_until;not null"`
	CreatedBy string
	CreatedAt time.Time
}
