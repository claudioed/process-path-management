package report_test

import (
	"context"
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/analytics/report"
)

// fakeStore is an in-memory implementation of both report ports used to
// exercise report derivation from a synthetic event sequence. It is a test
// double local to this package: the production stores live in the
// analyticsstore outbound adapter.
type fakeStore struct {
	seen map[string]bool
	rows map[report.RowKey]*report.Row
}

func newFakeStore() *fakeStore {
	return &fakeStore{seen: map[string]bool{}, rows: map[report.RowKey]*report.Row{}}
}

func dayBucket(t time.Time) time.Time { return t.UTC().Truncate(24 * time.Hour) }

func (s *fakeStore) dup(eventId string) bool {
	if s.seen[eventId] {
		return true
	}
	s.seen[eventId] = true
	return false
}

func (s *fakeStore) row(at time.Time) *report.Row {
	k := report.RowKey{DayBucket: dayBucket(at)}
	r, ok := s.rows[k]
	if !ok {
		r = &report.Row{Key: k}
		s.rows[k] = r
	}
	return r
}

func (s *fakeStore) ApplyProcessPathCreated(_ context.Context, eventId string, at time.Time) error {
	if s.dup(eventId) {
		return nil
	}
	s.row(at).PathsDefined++
	return nil
}

func (s *fakeStore) ApplyProcessPathUpdated(_ context.Context, eventId string, at time.Time) error {
	if s.dup(eventId) {
		return nil
	}
	s.row(at).PathsRevised++
	return nil
}

func (s *fakeStore) ApplyProcessPathDeactivated(_ context.Context, eventId string, at time.Time) error {
	if s.dup(eventId) {
		return nil
	}
	s.row(at).PathsDeactivated++
	return nil
}

func (s *fakeStore) Query(_ context.Context, q report.ReportQuery) (report.CatalogueReport, error) {
	out := report.CatalogueReport{}
	for k, r := range s.rows {
		if k.DayBucket.Before(q.From) || !k.DayBucket.Before(q.To) {
			continue
		}
		out.Rows = append(out.Rows, *r)
	}
	return out, nil
}

func (s *fakeStore) FreshnessLag(_ context.Context) (time.Duration, error) { return 0, nil }

func TestCatalogueReport_DerivesFromEventSequence(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	s := newFakeStore()
	ctx := context.Background()

	// One day of catalogue changes: two paths defined, one revised, one
	// deactivated.
	must(t, s.ApplyProcessPathCreated(ctx, "e1", base))
	must(t, s.ApplyProcessPathCreated(ctx, "e2", base))
	must(t, s.ApplyProcessPathUpdated(ctx, "e3", base))
	must(t, s.ApplyProcessPathDeactivated(ctx, "e4", base))

	rep, err := s.Query(ctx, report.ReportQuery{
		From:        base.Add(-24 * time.Hour),
		To:          base.Add(24 * time.Hour),
		Granularity: report.GranularityDay,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	bucket := dayBucket(base)
	row := findRow(rep, report.RowKey{DayBucket: bucket})
	if row == nil {
		t.Fatal("no row for the day bucket")
	}
	if row.PathsDefined != 2 {
		t.Errorf("PathsDefined = %d, want 2", row.PathsDefined)
	}
	if row.PathsRevised != 1 {
		t.Errorf("PathsRevised = %d, want 1", row.PathsRevised)
	}
	if row.PathsDeactivated != 1 {
		t.Errorf("PathsDeactivated = %d, want 1", row.PathsDeactivated)
	}
}

func TestCatalogueReport_FiltersAndIdempotency(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	ctx := context.Background()

	tests := []struct {
		name  string
		query report.ReportQuery
		want  int // number of rows expected
	}{
		{"no filter", report.ReportQuery{From: base.Add(-24 * time.Hour), To: base.Add(24 * time.Hour), Granularity: report.GranularityDay}, 1},
		{"window excludes all", report.ReportQuery{From: base.Add(30 * 24 * time.Hour), To: base.Add(60 * 24 * time.Hour), Granularity: report.GranularityDay}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newFakeStore()
			// Apply the same event twice with the same eventId -> counts once.
			must(t, s.ApplyProcessPathCreated(ctx, "dup", base))
			must(t, s.ApplyProcessPathCreated(ctx, "dup", base))

			rep, err := s.Query(ctx, tt.query)
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			if len(rep.Rows) != tt.want {
				t.Errorf("rows = %d, want %d", len(rep.Rows), tt.want)
			}
			if tt.name == "no filter" {
				row := findRow(rep, report.RowKey{DayBucket: dayBucket(base)})
				if row == nil || row.PathsDefined != 1 {
					t.Errorf("dedupe failed: row = %v", row)
				}
			}
		})
	}
}

func findRow(rep report.CatalogueReport, k report.RowKey) *report.Row {
	for i := range rep.Rows {
		if rep.Rows[i].Key == k {
			return &rep.Rows[i]
		}
	}
	return nil
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
}
