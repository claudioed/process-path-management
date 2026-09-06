package shared

import (
	"testing"
	"time"
)

func TestDomainEvents_EventNameAndOccurredAt(t *testing.T) {
	now := time.Now()

	created := ProcessPathCreated{PathId: "PICK", MatchPrefix: "pick", Direct: true, RequiredCapabilities: []Capability{"pick"}, At: now}
	if created.EventName() != "ProcessPathCreated" {
		t.Fatalf("want EventName ProcessPathCreated, got %s", created.EventName())
	}
	if created.OccurredAt() != now {
		t.Fatal("want OccurredAt to equal At")
	}

	updated := ProcessPathUpdated{PathId: "PICK", MatchPrefix: "pick-zone-a", Direct: true, RequiredCapabilities: []Capability{"pick", "hazmat"}, At: now}
	if updated.EventName() != "ProcessPathUpdated" {
		t.Fatalf("want EventName ProcessPathUpdated, got %s", updated.EventName())
	}
	if updated.OccurredAt() != now {
		t.Fatal("want OccurredAt to equal At")
	}

	deactivated := ProcessPathDeactivated{PathId: "PICK", At: now}
	if deactivated.EventName() != "ProcessPathDeactivated" {
		t.Fatalf("want EventName ProcessPathDeactivated, got %s", deactivated.EventName())
	}
	if deactivated.OccurredAt() != now {
		t.Fatal("want OccurredAt to equal At")
	}
}
