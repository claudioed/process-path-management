package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	outboundkafka "github.com/claudioed/process-path-management/internal/adapters/outbound/kafka"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// OutboxPublisher implements ports.EventPublisher by writing the event's
// Kafka wire form into outbox_events instead of the broker (ADR 0003).
// When called inside UnitOfWork.Execute the insert joins the use case's
// transaction, so the aggregate change and its event commit together or
// not at all. OutboxRelay later drains the table onto Kafka.
type OutboxPublisher struct {
	pool  *pgxpool.Pool
	newId func() string
}

// NewOutboxPublisher constructs an OutboxPublisher over pool. newId mints
// the envelope event_id (uuid) persisted with the row and published with
// the message, so a redelivered event carries the same id both times.
func NewOutboxPublisher(pool *pgxpool.Pool, newId func() string) *OutboxPublisher {
	return &OutboxPublisher{pool: pool, newId: newId}
}

// Publish stores event in the outbox. It never touches Kafka.
func (p *OutboxPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	enc, err := outboundkafka.Encode(event, p.newId())
	if err != nil {
		return err
	}
	_, err = querierFrom(ctx, p.pool).Exec(ctx, `
		INSERT INTO outbox_events (event_id, event_type, aggregate_id, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
	`, enc.EventId, enc.EventType, enc.Key, enc.Value, event.OccurredAt())
	if err != nil {
		return fmt.Errorf("postgres: enqueue outbox event %s: %w", enc.EventType, err)
	}
	return nil
}
