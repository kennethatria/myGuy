package services

import "errors"

// UserError is a message written for the person using the app (a rule
// they broke, something they can't do). Only these reach clients verbatim;
// any other error, such as one from the database, is logged and answered
// with a generic message so its details don't leak.
type UserError struct {
	msg string
}

func (e *UserError) Error() string { return e.msg }

// NewUserError returns an error whose message is safe to show users.
func NewUserError(msg string) error { return &UserError{msg: msg} }

// IsUserError reports whether err (or an error it wraps) is a UserError.
func IsUserError(err error) bool {
	var u *UserError
	return errors.As(err, &u)
}
