package cptschedule

import (
	"time"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// CutoffSnapshot is the wire-shape-agnostic snapshot of one Cutoff
// carried on CPTScheduleChanged. Kept as its own type (rather than
// reusing Cutoff directly) so the event payload's shape is decoupled
// from the aggregate's internal representation, mirroring how
// shared.ProcessPathCreated/Updated already carry their own plain
// fields rather than a *processpath.ProcessPath.
type CutoffSnapshot struct {
	CptId           string
	LocalTime       string
	DaysOfWeek      []Weekday
	ShipMethod      string
	EligiblePathIds []shared.PathId
}

// CPTScheduleChanged is raised whenever a CPTSchedule is defined or
// revised. It carries the FULL schedule (a snapshot, not a diff) — same
// self-sufficient-event convention as ProcessPathCreated/Updated already
// use in this codebase (ADR 0010): a fresh consumer replaying from
// FirstOffset needs no prior state to build its read model.
type CPTScheduleChanged struct {
	SiteId   shared.SiteId
	Timezone string
	Cutoffs  []CutoffSnapshot
	At       time.Time
}

func (e CPTScheduleChanged) EventName() string     { return "CPTScheduleChanged" }
func (e CPTScheduleChanged) OccurredAt() time.Time { return e.At }

// ToSnapshot folds a CPTSchedule aggregate's current state into the
// CPTScheduleChanged event payload.
func ToSnapshot(s *CPTSchedule, now time.Time) CPTScheduleChanged {
	cutoffs := make([]CutoffSnapshot, 0, len(s.Cutoffs()))
	for _, c := range s.Cutoffs() {
		cutoffs = append(cutoffs, CutoffSnapshot{
			CptId:           c.CptId(),
			LocalTime:       c.LocalTime(),
			DaysOfWeek:      c.DaysOfWeek(),
			ShipMethod:      c.ShipMethod(),
			EligiblePathIds: c.EligiblePathIds(),
		})
	}
	return CPTScheduleChanged{
		SiteId:   s.SiteId(),
		Timezone: s.Timezone(),
		Cutoffs:  cutoffs,
		At:       now,
	}
}
