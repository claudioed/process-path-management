package analyticsstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/adapters/outbound/analyticsstore"
	"github.com/claudioed/process-path-management/internal/analytics/report"
)

func TestMemoryStore_ProjectsAndDedupes(t *testing.T) {
	base := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	ctx := context.Background()
	s := analyticsstore.NewMemoryStore()

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
	}

	must(s.ApplyProcessPathCreated(ctx, "e1", base))
	must(s.ApplyProcessPathCreated(ctx, "e2", base))
	// duplicate event id -> counts once
	must(s.ApplyProcessPathUpdated(ctx, "dup", base))
	must(s.ApplyProcessPathUpdated(ctx, "dup", base))
	must(s.ApplyProcessPathDeactivated(ctx, "e3", base))

	rep, err := s.Query(ctx, report.ReportQuery{
		From:        base.Add(-24 * time.Hour),
		To:          base.Add(24 * time.Hour),
		Granularity: report.GranularityDay,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rep.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rep.Rows))
	}
	row := rep.Rows[0]
	if row.PathsDefined != 2 {
		t.Errorf("PathsDefined = %d, want 2", row.PathsDefined)
	}
	if row.PathsRevised != 1 {
		t.Errorf("PathsRevised = %d, want 1 (deduped)", row.PathsRevised)
	}
	if row.PathsDeactivated != 1 {
		t.Errorf("PathsDeactivated = %d, want 1", row.PathsDeactivated)
	}
}

func TestMemoryStore_FreshnessLag(t *testing.T) {
	ctx := context.Background()

	t.Run("empty store is zero lag", func(t *testing.T) {
		s := analyticsstore.NewMemoryStore()
		lag, err := s.FreshnessLag(ctx)
		if err != nil {
			t.Fatalf("FreshnessLag: %v", err)
		}
		if lag != 0 {
			t.Errorf("lag = %v, want 0", lag)
		}
	})

	t.Run("lag from latest applied event", func(t *testing.T) {
		latest := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
		now := latest.Add(90 * time.Second)
		s := analyticsstore.NewMemoryStore()
		s.Now = func() time.Time { return now }
		if err := s.ApplyProcessPathCreated(ctx, "e1", latest); err != nil {
			t.Fatalf("apply: %v", err)
		}
		lag, err := s.FreshnessLag(ctx)
		if err != nil {
			t.Fatalf("FreshnessLag: %v", err)
		}
		if lag != 90*time.Second {
			t.Errorf("lag = %v, want 90s", lag)
		}
	})
}
