package http_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/claudioed/process-path-management/internal/adapters/inbound/http"
	"github.com/claudioed/process-path-management/internal/analytics/report"
)

// fakeReportStore is a test double for report.ReportStore.
type fakeReportStore struct {
	report    report.CatalogueReport
	lag       time.Duration
	queryErr  error
	freshErr  error
	lastQuery report.ReportQuery
}

func (f *fakeReportStore) Query(_ context.Context, q report.ReportQuery) (report.CatalogueReport, error) {
	f.lastQuery = q
	return f.report, f.queryErr
}

func (f *fakeReportStore) FreshnessLag(_ context.Context) (time.Duration, error) {
	return f.lag, f.freshErr
}

func newReportsServer(store report.ReportStore) stdhttp.Handler {
	return http.NewReportsRouter(&http.ReportsHandlers{Store: store}, nil)
}

func TestReportsCatalogueGrowth_OK(t *testing.T) {
	bucket := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	store := &fakeReportStore{
		report: report.CatalogueReport{Rows: []report.Row{
			{
				Key:              report.RowKey{DayBucket: bucket},
				PathsDefined:     3,
				PathsRevised:     1,
				PathsDeactivated: 0,
			},
		}},
	}
	srv := newReportsServer(store)

	req := httptest.NewRequest(stdhttp.MethodGet,
		"/reports/catalogue-growth?from=2026-09-01T00:00:00Z&to=2026-09-08T00:00:00Z&granularity=day", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	if store.lastQuery.Granularity != report.GranularityDay {
		t.Errorf("granularity = %q, want day", store.lastQuery.Granularity)
	}

	var body struct {
		Rows []struct {
			DayBucket        string `json:"dayBucket"`
			PathsDefined     int    `json:"pathsDefined"`
			PathsRevised     int    `json:"pathsRevised"`
			PathsDeactivated int    `json:"pathsDeactivated"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(body.Rows))
	}
	row := body.Rows[0]
	if row.PathsDefined != 3 || row.PathsRevised != 1 || row.PathsDeactivated != 0 {
		t.Errorf("row = %+v", row)
	}
	if row.DayBucket != "2026-09-01T00:00:00Z" {
		t.Errorf("dayBucket = %q", row.DayBucket)
	}
}

func TestReportsCatalogueGrowth_MissingFromTo(t *testing.T) {
	srv := newReportsServer(&fakeReportStore{})
	tests := []struct {
		name string
		url  string
	}{
		{"no from", "/reports/catalogue-growth?to=2026-09-02T00:00:00Z"},
		{"no to", "/reports/catalogue-growth?from=2026-09-01T00:00:00Z"},
		{"bad from", "/reports/catalogue-growth?from=nope&to=2026-09-02T00:00:00Z"},
		{"bad granularity", "/reports/catalogue-growth?from=2026-09-01T00:00:00Z&to=2026-09-02T00:00:00Z&granularity=hour"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, tt.url, nil))
			if rec.Code != stdhttp.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("content-type = %q, want application/problem+json", ct)
			}
		})
	}
}

func TestReportsCatalogueGrowth_DefaultGranularity(t *testing.T) {
	store := &fakeReportStore{}
	srv := newReportsServer(store)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet,
		"/reports/catalogue-growth?from=2026-09-01T00:00:00Z&to=2026-09-08T00:00:00Z", nil))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if store.lastQuery.Granularity != report.GranularityDay {
		t.Errorf("default granularity = %q, want day", store.lastQuery.Granularity)
	}
}

func TestReportsFreshness_OK(t *testing.T) {
	store := &fakeReportStore{lag: 300 * time.Second}
	srv := newReportsServer(store)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/reports/catalogue-growth/freshness", nil))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		LagSeconds float64 `json:"lagSeconds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.LagSeconds != 300 {
		t.Errorf("lagSeconds = %v, want 300", body.LagSeconds)
	}
}

func TestReportsHealthz(t *testing.T) {
	srv := newReportsServer(&fakeReportStore{})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/healthz", nil))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
