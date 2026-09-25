// Package kafka contains process-path-management's inbound Kafka adapters.
// Today that is the analytics consumer only: this service otherwise has no
// inbound Kafka consumer (it is the SOURCE of the process-path published
// language, never a consumer of anyone else's topic) — the analytics
// consumer here is different: it consumes THIS SERVICE'S OWN analytics
// topic, replaying its own past-tense events into the analytical read
// model (ADR 0007), mirroring facility-layout's ADR-0010 pattern.
//
// Consistent with the rest of the analytics pipeline, this consumer is
// trace-free: it opens no spans and reads no trace headers.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/claudioed/process-path-management/internal/analytics/report"
)

// AnalyticsConsumerGroup is the Kafka consumer group the analytics
// projector reads under.
const AnalyticsConsumerGroup = "process-path-management-analytics"

// ProcessedEvents is the consumer's idempotency gate: MarkProcessed
// records an event id if it has not been seen and reports whether this
// call was the first to record it. It is declared here (rather than in
// application/ports) because it is an analytics-only concern the OLTP
// layers never touch; the analyticsstore ConsumedEventsRepo implements
// it.
type ProcessedEvents interface {
	MarkProcessed(ctx context.Context, eventId string) (bool, error)
}

// analyticsEnvelope is the inbound decode shape of the Envelope v1 wrapper
// on the analytics topic. Declared here (rather than imported from the
// outbound publisher) so this inbound adapter does not depend on an
// outbound adapter (arch-go enforced).
type analyticsEnvelope struct {
	EventId       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Source        string          `json:"source"`
	SchemaVersion int             `json:"schema_version"`
	Data          json.RawMessage `json:"data"`
}

// AnalyticsConsumer reads analytics events off the analytics topic and
// applies each to the catalogue-growth ProjectionStore, exactly once per
// event_id despite Kafka's at-least-once delivery.
type AnalyticsConsumer struct {
	Reader     *kafkago.Reader
	Projection report.ProjectionStore
	Processed  ProcessedEvents
	Logger     *slog.Logger
}

// NewAnalyticsConsumer constructs an AnalyticsConsumer reading topic from
// brokers under AnalyticsConsumerGroup.
func NewAnalyticsConsumer(brokers []string, topic string, projection report.ProjectionStore, processed ProcessedEvents, logger *slog.Logger) *AnalyticsConsumer {
	if logger == nil {
		logger = slog.Default()
	}
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: AnalyticsConsumerGroup,
		// Start a brand-new consumer group at the EARLIEST offset. The
		// analytics projection must see the full history of the topic
		// (it is a replayable read model, not a live integration
		// reaction), so a fresh projector reads from the beginning
		// rather than kafka-go's default of the latest offset. Once the
		// group has committed offsets, those take precedence and this
		// only affects the first join.
		StartOffset: kafkago.FirstOffset,
	})
	return &AnalyticsConsumer{Reader: reader, Projection: projection, Processed: processed, Logger: logger}
}

// Run reads and handles messages until ctx is cancelled or the reader
// returns a fatal error. A handling error is logged and the loop
// continues so one bad message cannot wedge the projector.
func (c *AnalyticsConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if err := c.HandleMessage(ctx, msg.Value); err != nil {
			c.Logger.ErrorContext(ctx, "analytics message handling failed", "error", err)
		}
	}
}

// Close releases the underlying Kafka reader.
func (c *AnalyticsConsumer) Close() error {
	return c.Reader.Close()
}

// HandleMessage decodes raw as an analyticsEnvelope and applies the
// matching projection method for its event_type. Event types outside the
// projection contract are ignored (and not marked processed). For a
// projecting event it dedupes on event_id via ProcessedEvents before
// applying, so a redelivery is a no-op. It is exported separately from Run
// so tests can feed raw envelopes without a live broker.
func (c *AnalyticsConsumer) HandleMessage(ctx context.Context, raw []byte) error {
	var env analyticsEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("analytics: decode envelope: %w", err)
	}

	switch env.EventType {
	case "ProcessPathCreated", "ProcessPathUpdated", "ProcessPathDeactivated":
	default:
		return nil
	}

	isNew, err := c.Processed.MarkProcessed(ctx, env.EventId)
	if err != nil {
		return fmt.Errorf("analytics: mark processed: %w", err)
	}
	if !isNew {
		return nil
	}

	switch env.EventType {
	case "ProcessPathCreated":
		return c.Projection.ApplyProcessPathCreated(ctx, env.EventId, env.OccurredAt)
	case "ProcessPathUpdated":
		return c.Projection.ApplyProcessPathUpdated(ctx, env.EventId, env.OccurredAt)
	case "ProcessPathDeactivated":
		return c.Projection.ApplyProcessPathDeactivated(ctx, env.EventId, env.OccurredAt)
	default:
		return nil
	}
}
