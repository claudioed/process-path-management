package kafka_test

import (
	"encoding/json"
	"testing"
	"time"

	outboundkafka "github.com/claudioed/process-path-management/internal/adapters/outbound/kafka"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

func TestAnalyticsEncoder_Encode_ProcessPathCreated(t *testing.T) {
	enc := outboundkafka.NewAnalyticsEncoder(func() string { return "evt-1" })
	now := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)

	got, err := enc.Encode(shared.ProcessPathCreated{
		PathId:               "PICK",
		MatchPrefix:          "pick",
		Direct:               true,
		RequiredCapabilities: []shared.Capability{"pick"},
		At:                   now,
	}, "evt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Topic != outboundkafka.AnalyticsTopic {
		t.Fatalf("topic = %q, want %q", got.Topic, outboundkafka.AnalyticsTopic)
	}
	if got.Key != "PICK" {
		t.Fatalf("key = %q, want PICK", got.Key)
	}
	if got.EventType != outboundkafka.EventTypeProcessPathCreated {
		t.Fatalf("event type = %q, want %q", got.EventType, outboundkafka.EventTypeProcessPathCreated)
	}

	var env outboundkafka.AnalyticsEnvelope
	if err := json.Unmarshal(got.Value, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.EventId != "evt-1" {
		t.Fatalf("event_id = %q, want evt-1", env.EventId)
	}
	if env.Source != outboundkafka.Source {
		t.Fatalf("source = %q, want %q", env.Source, outboundkafka.Source)
	}
	if env.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", env.SchemaVersion)
	}
	if !env.OccurredAt.Equal(now) {
		t.Fatalf("occurred_at = %v, want %v", env.OccurredAt, now)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if _, ok := data["match_prefix"]; !ok {
		t.Fatal("expected match_prefix in analytics data for ProcessPathCreated")
	}
}

func TestAnalyticsEncoder_Encode_ProcessPathDeactivated_OmitsDefinitionFields(t *testing.T) {
	enc := outboundkafka.NewAnalyticsEncoder(func() string { return "evt-2" })

	got, err := enc.Encode(shared.ProcessPathDeactivated{
		PathId: "PICK",
		At:     time.Now(),
	}, "evt-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env outboundkafka.AnalyticsEnvelope
	if err := json.Unmarshal(got.Value, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if _, present := data["match_prefix"]; present {
		t.Fatal("expected match_prefix to be omitted on a Deactivated analytics event")
	}
	if _, present := data["required_capabilities"]; present {
		t.Fatal("expected required_capabilities to be omitted on a Deactivated analytics event")
	}
}

func TestAnalyticsEncoder_Encode_UnknownEventType_ReturnsError(t *testing.T) {
	enc := outboundkafka.NewAnalyticsEncoder(func() string { return "evt-3" })

	_, err := enc.Encode(unmarshalableEvent{}, "evt-3")
	if err == nil {
		t.Fatal("expected an error for an unrecognized event type")
	}
}

func TestAnalyticsEncoder_SatisfiesEncoderInterface(t *testing.T) {
	var _ outboundkafka.Encoder = outboundkafka.NewAnalyticsEncoder(func() string { return "x" })
}
