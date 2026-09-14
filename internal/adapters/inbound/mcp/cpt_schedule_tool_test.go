package mcp

import (
	"context"
	"testing"

	"github.com/claudioed/process-path-management/internal/adapters/outbound/memory"
	"github.com/claudioed/process-path-management/internal/application/usecases"
	"github.com/claudioed/process-path-management/internal/domain/cptschedule"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// cptScheduleHarness builds Deps.GetCPTSchedule over a real in-memory
// CPTScheduleRepo, seeded directly with a domain aggregate.
type cptScheduleHarness struct {
	deps Deps
	repo *memory.CPTScheduleRepo
}

func newCPTScheduleHarness(t *testing.T) *cptScheduleHarness {
	t.Helper()
	repo := memory.NewCPTScheduleRepo()
	return &cptScheduleHarness{
		deps: Deps{
			GetCPTSchedule: &usecases.GetCPTSchedule{Repo: repo},
		},
		repo: repo,
	}
}

func (h *cptScheduleHarness) seedSchedule(t *testing.T, siteId shared.SiteId) *cptschedule.CPTSchedule {
	t.Helper()
	c, err := cptschedule.NewCutoff("sp1-1500", "15:00", []cptschedule.Weekday{cptschedule.Monday}, "ground", []shared.PathId{"PICK"})
	if err != nil {
		t.Fatalf("seedSchedule: build cutoff: %v", err)
	}
	s, err := cptschedule.Define(siteId, "America/Sao_Paulo", []cptschedule.Cutoff{c}, base)
	if err != nil {
		t.Fatalf("seedSchedule: define: %v", err)
	}
	if err := h.repo.Save(context.Background(), s); err != nil {
		t.Fatalf("seedSchedule: save: %v", err)
	}
	return s
}

func TestGetCPTSchedule(t *testing.T) {
	h := newCPTScheduleHarness(t)
	h.seedSchedule(t, "sp1")

	out, err := h.deps.getCPTSchedule(context.Background(), getCPTScheduleInput{SiteId: "sp1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SiteId != "sp1" || out.Timezone != "America/Sao_Paulo" {
		t.Fatalf("unexpected dto: %+v", out)
	}
	if len(out.Cutoffs) != 1 {
		t.Fatalf("want 1 cutoff, got %d", len(out.Cutoffs))
	}
	c := out.Cutoffs[0]
	if c.CptId != "sp1-1500" || c.LocalTime != "15:00" || c.ShipMethod != "ground" {
		t.Fatalf("unexpected cutoff dto: %+v", c)
	}
	if len(c.EligiblePathIds) != 1 || c.EligiblePathIds[0] != "PICK" {
		t.Fatalf("unexpected eligiblePathIds: %v", c.EligiblePathIds)
	}
}

func TestGetCPTScheduleNotFound(t *testing.T) {
	h := newCPTScheduleHarness(t)
	if _, err := h.deps.getCPTSchedule(context.Background(), getCPTScheduleInput{SiteId: "sp1"}); err == nil {
		t.Fatal("expected error for an unknown siteId")
	}
}

func TestGetCPTScheduleRejectsEmptySiteId(t *testing.T) {
	h := newCPTScheduleHarness(t)
	if _, err := h.deps.getCPTSchedule(context.Background(), getCPTScheduleInput{SiteId: ""}); err == nil {
		t.Fatal("expected error for empty siteId")
	}
}
