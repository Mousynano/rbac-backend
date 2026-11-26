// Package auth defines errors shared by authentication use cases and adapters.
package auth

import "errors"

var ErrUnavailable = errors.New("authentication dependency unavailable")

// UserProfile is the public identity returned to the authenticated caller.
type UserProfile struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}
