package kafka_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	inboundkafka "github.com/claudioed/process-path-management/internal/adapters/inbound/kafka"
)

// call captures one projection-store method invocation.
type call struct {
	method  string
	eventId string
	at      time.Time
}

// fakeProjection records the calls the consumer makes so a test can
// assert the envelope was routed to the right method.
type fakeProjection struct {
	calls []call
}

func (f *fakeProjection) ApplyProcessPathCreated(_ context.Context, eventId string, at time.Time) error {
	f.calls = append(f.calls, call{method: "created", eventId: eventId, at: at})
	return nil
}

func (f *fakeProjection) ApplyProcessPathUpdated(_ context.Context, eventId string, at time.Time) error {
	f.calls = append(f.calls, call{method: "updated", eventId: eventId, at: at})
	return nil
}

func (f *fakeProjection) ApplyProcessPathDeactivated(_ context.Context, eventId string, at time.Time) error {
	f.calls = append(f.calls, call{method: "deactivated", eventId: eventId, at: at})
	return nil
}

// fakeProcessed is an in-memory ProcessedEvents.
type fakeProcessed struct {
	seen map[string]bool
}

func newFakeProcessed() *fakeProcessed { return &fakeProcessed{seen: map[string]bool{}} }

func (p *fakeProcessed) MarkProcessed(_ context.Context, eventId string) (bool, error) {
	if p.seen[eventId] {
		return false, nil
	}
	p.seen[eventId] = true
	return true, nil
}

func envelope(t *testing.T, eventId, eventType string, at time.Time, data map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	env := map[string]any{
		"event_id":       eventId,
		"event_type":     eventType,
		"occurred_at":    at.Format(time.RFC3339Nano),
		"source":         "process-path-management",
		"schema_version": 1,
		"data":           json.RawMessage(raw),
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return b
}

func TestAnalyticsConsumer_RoutesEachEventType(t *testing.T) {
	at := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		eventType  string
		data       map[string]any
		wantMethod string
	}{
		{"created", "ProcessPathCreated", map[string]any{"path_id": "PICK"}, "created"},
		{"updated", "ProcessPathUpdated", map[string]any{"path_id": "PICK"}, "updated"},
		{"deactivated", "ProcessPathDeactivated", map[string]any{"path_id": "PICK"}, "deactivated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj := &fakeProjection{}
			c := &inboundkafka.AnalyticsConsumer{Projection: proj, Processed: newFakeProcessed(), Logger: slog.Default()}

			raw := envelope(t, "e-"+tt.name, tt.eventType, at, tt.data)
			if err := c.HandleMessage(context.Background(), raw); err != nil {
				t.Fatalf("HandleMessage: %v", err)
			}
			if len(proj.calls) != 1 {
				t.Fatalf("calls = %d, want 1", len(proj.calls))
			}
			got := proj.calls[0]
			if got.method != tt.wantMethod {
				t.Errorf("method = %q, want %q", got.method, tt.wantMethod)
			}
			if !got.at.Equal(at) {
				t.Errorf("at = %v, want %v", got.at, at)
			}
		})
	}
}

func TestAnalyticsConsumer_Idempotent(t *testing.T) {
	at := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	proj := &fakeProjection{}
	c := &inboundkafka.AnalyticsConsumer{Projection: proj, Processed: newFakeProcessed(), Logger: slog.Default()}

	raw := envelope(t, "dup", "ProcessPathCreated", at, map[string]any{"path_id": "PICK"})
	for range 2 {
		if err := c.HandleMessage(context.Background(), raw); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
	}
	if len(proj.calls) != 1 {
		t.Fatalf("expected 1 apply for duplicate delivery, got %d", len(proj.calls))
	}
}

func TestAnalyticsConsumer_IgnoresUnknownEventType(t *testing.T) {
	proj := &fakeProjection{}
	processed := newFakeProcessed()
	c := &inboundkafka.AnalyticsConsumer{Projection: proj, Processed: processed, Logger: slog.Default()}

	raw := envelope(t, "e1", "SomethingElseHappened", time.Now(), map[string]any{"foo": "bar"})
	if err := c.HandleMessage(context.Background(), raw); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(proj.calls) != 0 {
		t.Fatalf("expected unknown event to make no call, got %d", len(proj.calls))
	}
	if processed.seen["e1"] {
		t.Error("non-projecting event should not be marked processed")
	}
}
