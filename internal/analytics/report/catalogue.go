// Package report holds the process-path-management "Process Path Catalogue
// Growth & Change" read model: the shapes of the analytical report the data
// product serves, the query that selects it, and the outbound ports the
// writer and reader adapters implement. It is a read-model region that
// depends on nothing else in this module — the OLTP domain and application
// layers must not import it, and it must not import them (ADR 0007, mirroring
// facility-layout's ADR-0010).
//
// Unlike facility-layout's site/zone-scoped report, this service's catalogue
// has no such spatial dimension: a process path is a single flat identity
// (PathId), so rows are bucketed by DAY alone, with no additional scope
// dimension.
package report

import "time"

// Granularity is the time-bucket resolution a report is rolled up to. The
// process-path catalogue changes slowly (an operator-configured reference
// catalogue, not a live transactional stream), so only daily buckets are
// modelled.
type Granularity string

const (
	// GranularityDay rolls rows up into UTC day buckets (midnight UTC).
	GranularityDay Granularity = "day"
)

// RowKey identifies a single catalogue-growth row: the UTC day bucket the
// row aggregates. There is no scope dimension (no site/zone) — the
// process-path catalogue is a single flat identity space.
type RowKey struct {
	DayBucket time.Time
}

// Row is one aggregated catalogue-growth row for a day bucket. Each counter
// tracks how many of a given catalogue-change event landed in the bucket.
type Row struct {
	Key RowKey
	// PathsDefined is the number of ProcessPathCreated events.
	PathsDefined int
	// PathsRevised is the number of ProcessPathUpdated events.
	PathsRevised int
	// PathsDeactivated is the number of ProcessPathDeactivated events.
	PathsDeactivated int
}

// CatalogueReport is the full result of a report query: the matching rows.
type CatalogueReport struct {
	Rows []Row
}

// ReportQuery selects and filters the rows a report covers. From is
// inclusive and To is exclusive, both compared against a row's DayBucket.
type ReportQuery struct {
	From        time.Time
	To          time.Time
	Granularity Granularity
}
