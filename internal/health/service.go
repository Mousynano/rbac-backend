// Package health contains application health checks independent of HTTP.
package health

import "context"

// Checker is the dependency capability needed by readiness.
type Checker interface {
	Check(context.Context) error
}

// Service checks whether dependencies needed to serve requests are available.
type Service struct {
	db Checker
}

// New creates a health service with its required database dependency.
func New(db Checker) *Service { return &Service{db: db} }

// Ready reports whether the database can be reached.
func (s *Service) Ready(ctx context.Context) error { return s.db.Check(ctx) }
