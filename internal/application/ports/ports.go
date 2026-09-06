// Package ports declares the driven (outbound) interfaces the application
// layer needs, implemented by adapters. No adapter/framework type ever
// appears here — the domain and application layers depend on nothing but
// these interfaces and the domain package itself.
package ports

import (
	"context"
	"time"

	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// ProcessPathRepo persists and retrieves ProcessPath aggregates, keyed by
// their PathId (the natural key — there is no separate surrogate id,
// matching the retired YAML file's own schema where `id` was already the
// identity).
type ProcessPathRepo interface {
	// Save upserts a ProcessPath. Used for both first Define and every
	// subsequent Revise/Deactivate — the aggregate's own Status field is
	// what distinguishes a live path from a deactivated one, not deletion.
	Save(ctx context.Context, p *processpath.ProcessPath) error
	FindByID(ctx context.Context, id shared.PathId) (*processpath.ProcessPath, error)
	// ListActive returns every path currently Active — the read model the
	// operator SPA's default view and each consumer's own boot-time/cache
	// hydration both use.
	ListActive(ctx context.Context) ([]*processpath.ProcessPath, error)
	// ListAll returns every path regardless of status, for the SPA's
	// "show deactivated too" view and audit purposes.
	ListAll(ctx context.Context) ([]*processpath.ProcessPath, error)
}

// EventPublisher publishes a domain event raised by a use case. Mirrors
// the exact interface shape already used across the fleet (see
// labor-performance's ports.EventPublisher) so the outbound Kafka/log
// adapters can be lifted with minimal changes.
type EventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}

// Clock abstracts wall-clock time so use cases and tests never call
// time.Now() directly — same convention used fleet-wide.
type Clock interface {
	Now() time.Time
}
