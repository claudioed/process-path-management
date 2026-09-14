package cptschedule

import (
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

func TestCPTScheduleChanged_EventNameAndOccurredAt(t *testing.T) {
	now := time.Now()
	e := CPTScheduleChanged{SiteId: "sp1", Timezone: "America/Sao_Paulo", At: now}
	if e.EventName() != "CPTScheduleChanged" {
		t.Fatalf("want EventName CPTScheduleChanged, got %s", e.EventName())
	}
	if e.OccurredAt() != now {
		t.Fatal("want OccurredAt to equal At")
	}
}

func TestToSnapshot_CarriesFullScheduleState(t *testing.T) {
	c, err := NewCutoff("sp1-1500", "15:00", []Weekday{Monday, Tuesday}, "ground", []shared.PathId{"PICK", "PACK"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	s, err := Define("sp1", "America/Sao_Paulo", []Cutoff{c}, time.Now())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	now := time.Now()

	snap := ToSnapshot(s, now)
	if snap.SiteId != "sp1" || snap.Timezone != "America/Sao_Paulo" {
		t.Fatalf("unexpected snapshot header: %+v", snap)
	}
	if snap.At != now {
		t.Fatal("expected At to equal the time passed to ToSnapshot")
	}
	if len(snap.Cutoffs) != 1 {
		t.Fatalf("want 1 cutoff snapshot, got %d", len(snap.Cutoffs))
	}
	got := snap.Cutoffs[0]
	if got.CptId != "sp1-1500" || got.LocalTime != "15:00" || got.ShipMethod != "ground" {
		t.Fatalf("unexpected cutoff snapshot: %+v", got)
	}
	if len(got.DaysOfWeek) != 2 || got.DaysOfWeek[0] != Monday || got.DaysOfWeek[1] != Tuesday {
		t.Fatalf("unexpected daysOfWeek: %v", got.DaysOfWeek)
	}
	if len(got.EligiblePathIds) != 2 || got.EligiblePathIds[0] != "PICK" || got.EligiblePathIds[1] != "PACK" {
		t.Fatalf("unexpected eligiblePathIds: %v", got.EligiblePathIds)
	}
}

func TestToSnapshot_EmptyScheduleProducesEmptyCutoffsSlice(t *testing.T) {
	// Rehydrate with zero cutoffs is a state Define would reject, but is
	// reachable via a corrupted/manually-edited row -- ToSnapshot must
	// not panic on it.
	s := Rehydrate("sp1", "America/Sao_Paulo", nil, time.Now(), time.Now())
	snap := ToSnapshot(s, time.Now())
	if len(snap.Cutoffs) != 0 {
		t.Fatalf("want 0 cutoff snapshots, got %d", len(snap.Cutoffs))
	}
}
