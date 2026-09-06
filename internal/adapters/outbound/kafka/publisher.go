// Package kafka provides the outbound adapter that publishes
// process-path-management domain events onto Kafka, satisfying
// ports.EventPublisher. This is the ONLY way fulfillment-execution,
// wes-work-planning, and workforce-management learn about a process-path
// change once this service replaces the static YAML catalogue they used
// to boot-load — see this repo's README for the full migration story.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// Topic is the topic this service publishes every ProcessPath* event to.
// A single topic (not one per event type) matches the fleet's existing
// convention (e.g. warehouse.fulfillment.events carries several event
// types, filtered by consumers on event_type) — consumers that only care
// about, say, deactivations still see every message, but the filter cost
// is negligible against the operational simplicity of one topic per
// bounded context.
const Topic = "warehouse.process-path-management.events"

// Source identifies this service in the "source" field of every envelope
// it publishes.
const Source = "process-path-management"

// The event types published on Topic — this service's own past-tense
// domain events, verbatim. Any other event_type appearing on this topic
// would be a bug in this publisher, not something a consumer should ever
// need to guard against.
const (
	EventTypeProcessPathCreated     = "ProcessPathCreated"
	EventTypeProcessPathUpdated     = "ProcessPathUpdated"
	EventTypeProcessPathDeactivated = "ProcessPathDeactivated"
)

// Envelope is the CloudEvents-like wrapper shared across all
// warehouse-systems services (see e.g. fulfillment-execution's own
// kafka.Envelope) — Data is left as `any` here (rather than a fixed
// struct like fulfillment-execution's single-event-type Envelope)
// because this service publishes three distinct event shapes onto the
// same topic.
type Envelope struct {
	EventId    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	Source     string    `json:"source"`
	Data       any       `json:"data"`
}

// ProcessPathData is the payload shape for ALL THREE event types on this
// topic. RequiredCapabilities is omitted (not empty-arrayed) on a
// ProcessPathDeactivated event, since a deactivation carries no
// definition data — only the PathId and the fact that it happened.
type ProcessPathData struct {
	PathId               string   `json:"path_id"`
	MatchPrefix          string   `json:"match_prefix,omitempty"`
	Direct               bool     `json:"direct,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
}

// Writer is the subset of *kafkago.Writer the Publisher needs, so tests
// can substitute a fake without a live broker.
type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafkago.Message) error
}

// Publisher publishes process-path-management domain events onto Kafka.
// It satisfies ports.EventPublisher.
type Publisher struct {
	Writer Writer
	NewId  func() string
}

// NewPublisher constructs a Publisher writing to Topic on brokers.
func NewPublisher(brokers []string, newId func() string) *Publisher {
	return &Publisher{
		Writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(brokers...),
			Topic:                  Topic,
			Balancer:               &kafkago.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
		NewId: newId,
	}
}

// Publish forwards event onto Kafka, keyed by PathId so every event for
// the same path lands on the same partition — a consumer replaying the
// topic sees a given path's Created/Updated/Deactivated events in
// publish order, never interleaved with another path's out of order.
func (p *Publisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	var (
		pathId string
		data   ProcessPathData
		typ    string
	)
	switch e := event.(type) {
	case shared.ProcessPathCreated:
		pathId = string(e.PathId)
		typ = EventTypeProcessPathCreated
		data = ProcessPathData{
			PathId:               pathId,
			MatchPrefix:          e.MatchPrefix,
			Direct:               e.Direct,
			RequiredCapabilities: capabilitiesToStrings(e.RequiredCapabilities),
		}
	case shared.ProcessPathUpdated:
		pathId = string(e.PathId)
		typ = EventTypeProcessPathUpdated
		data = ProcessPathData{
			PathId:               pathId,
			MatchPrefix:          e.MatchPrefix,
			Direct:               e.Direct,
			RequiredCapabilities: capabilitiesToStrings(e.RequiredCapabilities),
		}
	case shared.ProcessPathDeactivated:
		pathId = string(e.PathId)
		typ = EventTypeProcessPathDeactivated
		data = ProcessPathData{PathId: pathId}
	default:
		// An event type this publisher does not know how to serialize.
		// Every event this service's use cases raise today is one of
		// the three above; a future new event type must be added here
		// explicitly rather than silently dropped.
		return fmt.Errorf("kafka: unknown event type %T", event)
	}

	env := Envelope{
		EventId:    p.NewId(),
		EventType:  typ,
		OccurredAt: event.OccurredAt(),
		Source:     Source,
		Data:       data,
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("kafka: marshal envelope: %w", err)
	}

	msg := kafkago.Message{Key: []byte(pathId), Value: payload}
	if err := p.Writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka: publish %s: %w", typ, err)
	}
	return nil
}

// Close releases the underlying Kafka writer.
func (p *Publisher) Close() error {
	if w, ok := p.Writer.(*kafkago.Writer); ok {
		return w.Close()
	}
	return nil
}

func capabilitiesToStrings(caps []shared.Capability) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = string(c)
	}
	return out
}
