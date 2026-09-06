// Package memory provides in-memory implementations of the outbound
// ports, used by the unit/httptest suites and by `go run ./cmd/pathmgmt`
// with no DATABASE_URL set, so the service is fully functional without
// Postgres.
package memory

import "time"

// SystemClock implements ports.Clock using wall-clock time.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
