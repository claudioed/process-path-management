package processpath

import (
	"errors"
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

func TestDefine_RejectsEmptyMatchPrefix(t *testing.T) {
	_, err := Define("PICK", "", true, []shared.Capability{"pick"}, time.Now())
	if !errors.Is(err, ErrEmptyMatchPrefix) {
		t.Fatalf("want ErrEmptyMatchPrefix, got %v", err)
	}
}

func TestDefine_RejectsUppercaseMatchPrefix(t *testing.T) {
	_, err := Define("PICK", "Pick", true, []shared.Capability{"pick"}, time.Now())
	if !errors.Is(err, ErrMatchPrefixNotLowercase) {
		t.Fatalf("want ErrMatchPrefixNotLowercase, got %v", err)
	}
}

func TestDefine_RejectsEmptyRequiredCapabilities(t *testing.T) {
	_, err := Define("PICK", "pick", true, nil, time.Now())
	if !errors.Is(err, ErrNoRequiredCapabilities) {
		t.Fatalf("want ErrNoRequiredCapabilities, got %v", err)
	}
}

func TestDefine_ValidInput_IsActiveWithMatchingFields(t *testing.T) {
	now := time.Now()
	p, err := Define("PICK", "pick", true, []shared.Capability{"pick"}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID() != "PICK" || p.MatchPrefix() != "pick" || !p.Direct() {
		t.Fatalf("unexpected fields: %+v", p)
	}
	if !p.IsActive() {
		t.Fatal("expected newly defined path to be Active")
	}
	if p.CreatedAt() != now || p.UpdatedAt() != now {
		t.Fatal("expected createdAt/updatedAt to both equal the construction time")
	}
}

func TestRevise_OnDeactivatedPath_ReturnsErrPathDeactivated(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	p.Deactivate(time.Now())
	_, err := p.Revise("pick-v2", []shared.Capability{"pick"}, time.Now())
	if !errors.Is(err, ErrPathDeactivated) {
		t.Fatalf("want ErrPathDeactivated, got %v", err)
	}
}

func TestRevise_NoActualChange_ReturnsChangedFalse(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	changed, err := p.Revise("pick", []shared.Capability{"pick"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for an identical revision")
	}
}

func TestRevise_ActualChange_ReturnsChangedTrueAndUpdatesFields(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	later := time.Now().Add(time.Hour)
	changed, err := p.Revise("pick-zone-a", []shared.Capability{"pick", "hazmat"}, later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if p.MatchPrefix() != "pick-zone-a" {
		t.Fatalf("want matchPrefix pick-zone-a, got %s", p.MatchPrefix())
	}
	if len(p.RequiredCapabilities()) != 2 {
		t.Fatalf("want 2 capabilities, got %d", len(p.RequiredCapabilities()))
	}
	if p.UpdatedAt() != later {
		t.Fatal("expected updatedAt to advance to the revision time")
	}
}

func TestRevise_InvalidInput_RejectedEvenOnActivePath(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	if _, err := p.Revise("", []shared.Capability{"pick"}, time.Now()); !errors.Is(err, ErrEmptyMatchPrefix) {
		t.Fatalf("want ErrEmptyMatchPrefix, got %v", err)
	}
}

func TestDeactivate_IsIdempotent(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	p.Deactivate(time.Now())
	if p.Status() != StatusDeactivated {
		t.Fatal("expected Deactivated after first call")
	}
	// second call must not panic or change status
	p.Deactivate(time.Now())
	if p.Status() != StatusDeactivated {
		t.Fatal("expected still Deactivated after redundant call")
	}
}

func TestRequiredCapabilities_ReturnsDefensiveCopy(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	caps := p.RequiredCapabilities()
	caps[0] = "mutated"
	if p.RequiredCapabilities()[0] != "pick" {
		t.Fatal("expected internal state to be unaffected by mutating the returned slice")
	}
}

func TestRehydrate_ReconstructsWithoutRevalidating(t *testing.T) {
	created := time.Now().Add(-time.Hour)
	updated := time.Now()
	// Rehydrate deliberately accepts state that Define would reject (e.g. an
	// uppercase matchPrefix), because it reconstructs already-persisted,
	// already-validated-at-write-time data -- repository adapters must
	// never re-run construction invariants on read.
	p := Rehydrate("PICK", "PICK-LEGACY", false, []shared.Capability{"pick"}, StatusDeactivated, created, updated)
	if p.ID() != "PICK" {
		t.Fatalf("want id PICK, got %s", p.ID())
	}
	if p.MatchPrefix() != "PICK-LEGACY" {
		t.Fatalf("want matchPrefix PICK-LEGACY, got %s", p.MatchPrefix())
	}
	if p.Direct() {
		t.Fatal("want direct=false")
	}
	if p.Status() != StatusDeactivated || p.IsActive() {
		t.Fatal("want rehydrated status Deactivated")
	}
	if p.CreatedAt() != created || p.UpdatedAt() != updated {
		t.Fatal("want createdAt/updatedAt to match the rehydrated values exactly")
	}
}

func TestRevise_DifferentCapabilityCount_ReturnsChangedTrue(t *testing.T) {
	p, _ := Define("PICK", "pick", true, []shared.Capability{"pick"}, time.Now())
	changed, err := p.Revise("pick", []shared.Capability{"pick", "hazmat"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when the capability count differs, even if matchPrefix is unchanged")
	}
}
