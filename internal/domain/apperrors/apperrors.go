// Package apperrors models business rules whose reason is safe to reach the
// end user. It lives in the domain layer so both the application and the
// interface layers can depend on it without inverting the dependency graph.
package apperrors

import "errors"

// UserFacing represents a business rule whose reason MUST reach the end user.
// Any error that is not of this type is logged on the server and never exposed
// to the client.
type UserFacing struct {
	Message string
}

func (e *UserFacing) Error() string { return e.Message }

// New creates an error that is safe to show to the user.
func New(message string) *UserFacing { return &UserFacing{Message: message} }

// Is reports whether err is, or wraps, a user-facing error and returns it so
// the caller can use its curated message.
func Is(err error) (*UserFacing, bool) {
	var target *UserFacing
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}
