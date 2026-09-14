package cptschedule

import (
	"errors"
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

func validCutoff(t *testing.T) Cutoff {
	t.Helper()
	c, err := NewCutoff("sp1-1500", "15:00", []Weekday{Monday, Tuesday, Wednesday, Thursday, Friday}, "ground", []shared.PathId{"PICK"})
	if err != nil {
		t.Fatalf("unexpected error building a valid cutoff: %v", err)
	}
	return c
}

// --- Cutoff invariants -----------------------------------------------------

func TestNewCutoff_RejectsEmptyCptId(t *testing.T) {
	_, err := NewCutoff("", "15:00", []Weekday{Monday}, "ground", []shared.PathId{"PICK"})
	if !errors.Is(err, ErrEmptyCptId) {
		t.Fatalf("want ErrEmptyCptId, got %v", err)
	}
}

func TestNewCutoff_RejectsEmptyLocalTime(t *testing.T) {
	_, err := NewCutoff("sp1-1500", "", []Weekday{Monday}, "ground", []shared.PathId{"PICK"})
	if !errors.Is(err, ErrEmptyLocalTime) {
		t.Fatalf("want ErrEmptyLocalTime, got %v", err)
	}
}

func TestNewCutoff_RejectsMalformedLocalTime(t *testing.T) {
	for _, bad := range []string{"3pm", "15:00:00", "25:00", "15:60", "1500", "15-00"} {
		_, err := NewCutoff("sp1-1500", bad, []Weekday{Monday}, "ground", []shared.PathId{"PICK"})
		if !errors.Is(err, ErrInvalidLocalTime) {
			t.Fatalf("localTime=%q: want ErrInvalidLocalTime, got %v", bad, err)
		}
	}
}

func TestNewCutoff_AcceptsValidLocalTimeBoundaries(t *testing.T) {
	for _, good := range []string{"00:00", "23:59", "09:05", "15:00"} {
		if _, err := NewCutoff("sp1-1500", good, []Weekday{Monday}, "ground", []shared.PathId{"PICK"}); err != nil {
			t.Fatalf("localTime=%q: unexpected error: %v", good, err)
		}
	}
}

func TestNewCutoff_RejectsEmptyDaysOfWeek(t *testing.T) {
	_, err := NewCutoff("sp1-1500", "15:00", nil, "ground", []shared.PathId{"PICK"})
	if !errors.Is(err, ErrNoDaysOfWeek) {
		t.Fatalf("want ErrNoDaysOfWeek, got %v", err)
	}
}

func TestNewCutoff_RejectsInvalidDayOfWeek(t *testing.T) {
	_, err := NewCutoff("sp1-1500", "15:00", []Weekday{"Funday"}, "ground", []shared.PathId{"PICK"})
	if !errors.Is(err, ErrInvalidDayOfWeek) {
		t.Fatalf("want ErrInvalidDayOfWeek, got %v", err)
	}
}

func TestNewCutoff_RejectsEmptyShipMethod(t *testing.T) {
	_, err := NewCutoff("sp1-1500", "15:00", []Weekday{Monday}, "", []shared.PathId{"PICK"})
	if !errors.Is(err, ErrEmptyShipMethod) {
		t.Fatalf("want ErrEmptyShipMethod, got %v", err)
	}
}

func TestNewCutoff_RejectsEmptyEligiblePathIds(t *testing.T) {
	_, err := NewCutoff("sp1-1500", "15:00", []Weekday{Monday}, "ground", nil)
	if !errors.Is(err, ErrNoEligiblePathIds) {
		t.Fatalf("want ErrNoEligiblePathIds, got %v", err)
	}
}

func TestCutoff_Accessors_ReturnDefensiveCopies(t *testing.T) {
	c := validCutoff(t)
	days := c.DaysOfWeek()
	days[0] = "Funday"
	if c.DaysOfWeek()[0] != Monday {
		t.Fatal("expected internal daysOfWeek to be unaffected by mutating the returned slice")
	}
	ids := c.EligiblePathIds()
	ids[0] = "MUTATED"
	if c.EligiblePathIds()[0] != "PICK" {
		t.Fatal("expected internal eligiblePathIds to be unaffected by mutating the returned slice")
	}
}

// --- CPTSchedule.Define invariants ------------------------------------------

func TestDefine_RejectsEmptyTimezone(t *testing.T) {
	_, err := Define("sp1", "", []Cutoff{validCutoff(t)}, time.Now())
	if !errors.Is(err, ErrEmptyTimezone) {
		t.Fatalf("want ErrEmptyTimezone, got %v", err)
	}
}

func TestDefine_RejectsUnrecognizedTimezone(t *testing.T) {
	_, err := Define("sp1", "Not/AZone", []Cutoff{validCutoff(t)}, time.Now())
	if !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("want ErrInvalidTimezone, got %v", err)
	}
}

func TestDefine_RejectsNoCutoffs(t *testing.T) {
	_, err := Define("sp1", "America/Sao_Paulo", nil, time.Now())
	if !errors.Is(err, ErrNoCutoffs) {
		t.Fatalf("want ErrNoCutoffs, got %v", err)
	}
}

func TestDefine_RejectsDuplicateCptId(t *testing.T) {
	c1 := validCutoff(t)
	c2, err := NewCutoff("sp1-1500", "18:00", []Weekday{Monday}, "same-day", []shared.PathId{"PACK"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = Define("sp1", "America/Sao_Paulo", []Cutoff{c1, c2}, time.Now())
	if !errors.Is(err, ErrDuplicateCptId) {
		t.Fatalf("want ErrDuplicateCptId, got %v", err)
	}
}

func TestDefine_ValidInput_ConstructsSchedule(t *testing.T) {
	now := time.Now()
	c := validCutoff(t)
	s, err := Define("sp1", "America/Sao_Paulo", []Cutoff{c}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.SiteId() != "sp1" || s.Timezone() != "America/Sao_Paulo" {
		t.Fatalf("unexpected fields: %+v", s)
	}
	if len(s.Cutoffs()) != 1 {
		t.Fatalf("want 1 cutoff, got %d", len(s.Cutoffs()))
	}
	if s.CreatedAt() != now || s.UpdatedAt() != now {
		t.Fatal("expected createdAt/updatedAt to both equal the construction time")
	}
}

func TestDefine_DoesNotEnforceEligiblePathIdsAgainstAnyStore(t *testing.T) {
	// The domain package itself accepts any non-empty eligiblePathIds --
	// the cross-aggregate "must be Active in this service's own store"
	// check is a use-case-level concern (ADR 0010), not a domain
	// invariant, since it needs the ProcessPathRepo.
	c, err := NewCutoff("sp1-1500", "15:00", []Weekday{Monday}, "ground", []shared.PathId{"DOES-NOT-EXIST"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := Define("sp1", "America/Sao_Paulo", []Cutoff{c}, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- CPTSchedule.Revise -----------------------------------------------------

func TestRevise_NoActualChange_ReturnsChangedFalse(t *testing.T) {
	c := validCutoff(t)
	s, _ := Define("sp1", "America/Sao_Paulo", []Cutoff{c}, time.Now())
	changed, err := s.Revise("America/Sao_Paulo", []Cutoff{c}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for an identical revision")
	}
}

func TestRevise_TimezoneChange_ReturnsChangedTrue(t *testing.T) {
	c := validCutoff(t)
	s, _ := Define("sp1", "America/Sao_Paulo", []Cutoff{c}, time.Now())
	later := time.Now().Add(time.Hour)
	changed, err := s.Revise("America/New_York", []Cutoff{c}, later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when timezone differs")
	}
	if s.Timezone() != "America/New_York" {
		t.Fatalf("want America/New_York, got %s", s.Timezone())
	}
	if s.UpdatedAt() != later {
		t.Fatal("expected updatedAt to advance to the revision time")
	}
}

func TestRevise_CutoffsChange_ReturnsChangedTrue(t *testing.T) {
	c1 := validCutoff(t)
	s, _ := Define("sp1", "America/Sao_Paulo", []Cutoff{c1}, time.Now())
	c2, err := NewCutoff("sp1-1800", "18:00", []Weekday{Tuesday}, "same-day", []shared.PathId{"PACK"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	changed, err := s.Revise("America/Sao_Paulo", []Cutoff{c1, c2}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when the cutoff set differs")
	}
	if len(s.Cutoffs()) != 2 {
		t.Fatalf("want 2 cutoffs, got %d", len(s.Cutoffs()))
	}
}

func TestRevise_InvalidInput_RejectedWithoutMutatingState(t *testing.T) {
	c := validCutoff(t)
	s, _ := Define("sp1", "America/Sao_Paulo", []Cutoff{c}, time.Now())
	if _, err := s.Revise("", []Cutoff{c}, time.Now()); !errors.Is(err, ErrEmptyTimezone) {
		t.Fatalf("want ErrEmptyTimezone, got %v", err)
	}
	if s.Timezone() != "America/Sao_Paulo" {
		t.Fatal("expected the schedule to be unchanged after a rejected revision")
	}
}

func TestRevise_DuplicateCptId_Rejected(t *testing.T) {
	c1 := validCutoff(t)
	s, _ := Define("sp1", "America/Sao_Paulo", []Cutoff{c1}, time.Now())
	c2, err := NewCutoff(c1.CptId(), "18:00", []Weekday{Tuesday}, "same-day", []shared.PathId{"PACK"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := s.Revise("America/Sao_Paulo", []Cutoff{c1, c2}, time.Now()); !errors.Is(err, ErrDuplicateCptId) {
		t.Fatalf("want ErrDuplicateCptId, got %v", err)
	}
}

// --- Rehydrate ---------------------------------------------------------------

func TestRehydrate_ReconstructsWithoutRevalidating(t *testing.T) {
	created := time.Now().Add(-time.Hour)
	updated := time.Now()
	// Rehydrate deliberately accepts state Define would reject (e.g. an
	// empty timezone), mirroring processpath.Rehydrate's own posture:
	// repository adapters never re-run construction invariants on read.
	s := Rehydrate("sp1", "", nil, created, updated)
	if s.SiteId() != "sp1" {
		t.Fatalf("want siteId sp1, got %s", s.SiteId())
	}
	if s.CreatedAt() != created || s.UpdatedAt() != updated {
		t.Fatal("want createdAt/updatedAt to match the rehydrated values exactly")
	}
}

// --- AllEligiblePathIds ------------------------------------------------------

func TestAllEligiblePathIds_DedupesAndSortsAcrossCutoffs(t *testing.T) {
	c1, err := NewCutoff("sp1-1500", "15:00", []Weekday{Monday}, "ground", []shared.PathId{"PACK", "PICK"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	c2, err := NewCutoff("sp1-1800", "18:00", []Weekday{Tuesday}, "same-day", []shared.PathId{"PICK", "SLAM"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	s, err := Define("sp1", "America/Sao_Paulo", []Cutoff{c1, c2}, time.Now())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := s.AllEligiblePathIds()
	want := []shared.PathId{"PACK", "PICK", "SLAM"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}
