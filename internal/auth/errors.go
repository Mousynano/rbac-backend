// Package auth defines errors shared by authentication use cases and adapters.
package auth

import "errors"

var ErrUnavailable = errors.New("authentication dependency unavailable")
