package services

import (
	"errors"

	"myguy/internal/proximity"
)

// userFacing are the errors written for the person using the app. Only
// these reach clients verbatim; anything else (a database error, say) is
// logged and answered generically so its details don't leak.
var userFacing = []error{
	ErrInvalidRating, ErrTaskNotCompleted, ErrAlreadyReviewed, ErrNotTaskParticipant,
	ErrInvalidCode, ErrTooManyRequests, ErrFullNameRequired,
	ErrUserNotFound, ErrEmailExists, ErrUsernameExists,
	ErrTaskNotFound, ErrUnauthorized, ErrTaskNotOpen, ErrInvalidStatus,
	ErrApplicationNotFound, ErrApplicationNotPending, ErrOwnTask, ErrAlreadyApplied, ErrTaskWasAssigned,
	ErrHeadlineRequired, ErrBodyRequired, ErrHeadlineTooLong, ErrBodyTooLong, ErrMessageTooLong, ErrContactDetails,
	proximity.ErrInvalidLocation,
}

// IsUserFacing reports whether err is one of the messages written for users.
func IsUserFacing(err error) bool {
	for _, e := range userFacing {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
