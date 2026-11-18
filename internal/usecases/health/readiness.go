// Package health defines the application readiness boundary.
package health

import "context"

// ReadinessChecker verifies dependencies required to serve the core API.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}
