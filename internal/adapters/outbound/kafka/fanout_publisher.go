package kafka

import (
	"context"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// FanOutPublisher forwards each domain event to several EventPublishers in
// turn. It is how the OLTP composition root publishes one event stream to
// both the integration topic and the analytics topic when running with
// Kafka but no Postgres (no outbox/transaction to bind them to) — the
// transactional-outbox path (postgres.OutboxPublisher) is used instead
// whenever a database is configured.
//
// Fan-out is fail-fast in publisher order: the first error stops the
// fan-out and is returned, so an analytics-publish failure is never
// silently swallowed behind an earlier publisher's success.
type FanOutPublisher []Sender

// Sender is the minimal shape a fan-out target needs: publish one domain
// event. Both *Publisher (integration, direct) and *AnalyticsPublisher
// satisfy it.
type Sender interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}

// Publish forwards event to every configured publisher, returning the
// first error encountered.
func (f FanOutPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	for _, p := range f {
		if err := p.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
