package kafka_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	outboundkafka "github.com/claudioed/process-path-management/internal/adapters/outbound/kafka"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

type fakeWriter struct {
	messages []kafkago.Message
}

func (w *fakeWriter) WriteMessages(_ context.Context, msgs ...kafkago.Message) error {
	w.messages = append(w.messages, msgs...)
	return nil
}

func TestPublish_ProcessPathCreated_WritesEnvelopeKeyedByPathId(t *testing.T) {
	w := &fakeWriter{}
	p := &outboundkafka.Publisher{Writer: w, NewId: func() string { return "evt-1" }}
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)

	err := p.Publish(context.Background(), shared.ProcessPathCreated{
		PathId:               "PICK",
		MatchPrefix:          "pick",
		Direct:               true,
		RequiredCapabilities: []shared.Capability{"pick"},
		At:                   now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(w.messages) != 1 {
		t.Fatalf("want 1 message, got %d", len(w.messages))
	}
	if string(w.messages[0].Key) != "PICK" {
		t.Fatalf("want message key PICK, got %s", w.messages[0].Key)
	}

	var env outboundkafka.Envelope
	if err := json.Unmarshal(w.messages[0].Value, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.EventType != outboundkafka.EventTypeProcessPathCreated {
		t.Fatalf("want event_type ProcessPathCreated, got %s", env.EventType)
	}
	if env.Source != outboundkafka.Source {
		t.Fatalf("want source %s, got %s", outboundkafka.Source, env.Source)
	}
}

func TestPublish_ProcessPathDeactivated_OmitsDefinitionFields(t *testing.T) {
	w := &fakeWriter{}
	p := &outboundkafka.Publisher{Writer: w, NewId: func() string { return "evt-2" }}

	err := p.Publish(context.Background(), shared.ProcessPathDeactivated{
		PathId: "PICK",
		At:     time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.messages[0].Value, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if _, present := data["match_prefix"]; present {
		t.Fatal("expected match_prefix to be omitted on a Deactivated event")
	}
	if _, present := data["required_capabilities"]; present {
		t.Fatal("expected required_capabilities to be omitted on a Deactivated event")
	}
}

func TestPublish_UnknownEventType_ReturnsError(t *testing.T) {
	w := &fakeWriter{}
	p := &outboundkafka.Publisher{Writer: w, NewId: func() string { return "evt-3" }}

	type unknownEvent struct{}
	err := p.Publish(context.Background(), unmarshalableEvent{})
	_ = unknownEvent{}
	if err == nil {
		t.Fatal("expected an error for an unrecognized event type")
	}
}

type unmarshalableEvent struct{}

func (unmarshalableEvent) EventName() string     { return "Unknown" }
func (unmarshalableEvent) OccurredAt() time.Time { return time.Now() }
