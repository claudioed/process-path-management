package report

import (
	"context"
	"time"
)

// ReportStore is the read side of the catalogue-growth data product: the
// reader process queries it to serve reports. It is read-only by contract —
// the Postgres implementation runs over a pool pinned to a read-only role.
type ReportStore interface {
	// Query returns the catalogue-growth rows matching q.
	Query(ctx context.Context, q ReportQuery) (CatalogueReport, error)
	// FreshnessLag reports how far the read model lags real time: the age
	// of the most recently applied event. A larger lag means the
	// projection is further behind the event stream.
	FreshnessLag(ctx context.Context) (time.Duration, error)
}

// ProjectionStore is the write side of the catalogue-growth data product:
// the projector process applies each consumed event to it. Every Apply*
// method is idempotent on eventId — applying the same eventId twice records
// the effect once, so the at-least-once Kafka stream can be projected
// exactly once.
type ProjectionStore interface {
	// ApplyProcessPathCreated records a path definition in at's day
	// bucket.
	ApplyProcessPathCreated(ctx context.Context, eventId string, at time.Time) error
	// ApplyProcessPathUpdated records a path revision.
	ApplyProcessPathUpdated(ctx context.Context, eventId string, at time.Time) error
	// ApplyProcessPathDeactivated records a path deactivation.
	ApplyProcessPathDeactivated(ctx context.Context, eventId string, at time.Time) error
}
