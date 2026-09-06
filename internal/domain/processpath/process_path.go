// Package processpath implements the ProcessPath aggregate: the
// operator-configurable definition of one process path (its canonical
// identity, the path_id-family match rule downstream consumers use, and
// the capabilities a station/associate must hold to work it).
//
// This aggregate replaces what was, before this service existed, a
// static YAML file (warehouse-infra/config/process-paths/sortable-fc.yaml)
// loaded once at boot by three other services. The schema is carried
// over field-for-field from that file's own documented schema so this
// migration is a like-for-like data model change, not a redesign —
// see that file's own header comment for the original field rationale.
package processpath

import (
	"errors"
	"strings"
	"time"

	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// ErrEmptyMatchPrefix, ErrMatchPrefixNotLowercase, and
// ErrNoRequiredCapabilities enforce this aggregate's construction
// invariants. A malformed path definition must never be constructible —
// there is no "fix it later" state, matching this fleet's existing rule
// that a boot-time catalogue failure is loud, never a silent partial
// entry (see the retired YAML file's own header comment on this point).
var (
	ErrEmptyMatchPrefix        = errors.New("processpath: matchPrefix must not be empty")
	ErrMatchPrefixNotLowercase = errors.New("processpath: matchPrefix must be lower-case")
	ErrNoRequiredCapabilities  = errors.New("processpath: requiredCapabilities must be non-empty")
)

// ErrPathDeactivated is returned by any mutation attempted against a
// deactivated path. A deactivated path is a closed historical record —
// exactly the same "frozen, never mutated after close" posture
// labor-performance's LaborStandard.Close takes, for the same reason:
// anything that already read this path's definition before deactivation
// must remain able to trust it did not change out from under it.
var ErrPathDeactivated = errors.New("processpath: path is deactivated")

// Status is the activation lifecycle of a ProcessPath. There is no
// "draft" state — a path is live the instant it is defined, since the
// whole purpose of this service is operators configuring paths that
// take effect immediately, not a review workflow.
type Status string

const (
	StatusActive      Status = "ACTIVE"
	StatusDeactivated Status = "DEACTIVATED"
)

// ProcessPath is the aggregate root. Identity (PathId) is immutable once
// constructed; MatchPrefix and RequiredCapabilities may be revised while
// Active via Revise; Direct is immutable (it describes a structural fact
// about the path's routing shape, not an operational parameter operators
// tune — see the retired YAML file's own comment reserving it for a
// future multi-hop topology, not day-to-day editing).
type ProcessPath struct {
	id                   shared.PathId
	matchPrefix          string
	direct               bool
	requiredCapabilities []shared.Capability
	status               Status
	createdAt            time.Time
	updatedAt            time.Time
}

// Define constructs a brand-new, Active ProcessPath. Returns one of the
// Err* sentinels above if id/matchPrefix/requiredCapabilities are
// malformed. matchPrefix is validated lower-case here (not
// lower-cased for the caller) so persisted data is exactly what was
// validated — never a silently-transformed value.
func Define(id shared.PathId, matchPrefix string, direct bool, requiredCapabilities []shared.Capability, now time.Time) (*ProcessPath, error) {
	if err := validate(matchPrefix, requiredCapabilities); err != nil {
		return nil, err
	}
	return &ProcessPath{
		id:                   id,
		matchPrefix:          matchPrefix,
		direct:               direct,
		requiredCapabilities: append([]shared.Capability(nil), requiredCapabilities...),
		status:               StatusActive,
		createdAt:            now,
		updatedAt:            now,
	}, nil
}

// Rehydrate reconstructs a ProcessPath from persisted state without
// re-validating construction invariants (used by repository adapters) —
// same pattern as labor-performance's standard.Rehydrate.
func Rehydrate(id shared.PathId, matchPrefix string, direct bool, requiredCapabilities []shared.Capability, status Status, createdAt, updatedAt time.Time) *ProcessPath {
	return &ProcessPath{
		id:                   id,
		matchPrefix:          matchPrefix,
		direct:               direct,
		requiredCapabilities: requiredCapabilities,
		status:               status,
		createdAt:            createdAt,
		updatedAt:            updatedAt,
	}
}

// Revise updates matchPrefix/requiredCapabilities on an Active path.
// Returns ErrPathDeactivated if the path is not Active. Returns true if
// anything actually changed (the caller uses this to decide whether to
// raise ProcessPathUpdated — a no-op revision raises nothing, so
// consumers never have to diff two identical payloads to notice nothing
// changed).
func (p *ProcessPath) Revise(matchPrefix string, requiredCapabilities []shared.Capability, now time.Time) (changed bool, err error) {
	if p.status != StatusActive {
		return false, ErrPathDeactivated
	}
	if err := validate(matchPrefix, requiredCapabilities); err != nil {
		return false, err
	}
	if p.matchPrefix == matchPrefix && capabilitiesEqual(p.requiredCapabilities, requiredCapabilities) {
		return false, nil
	}
	p.matchPrefix = matchPrefix
	p.requiredCapabilities = append([]shared.Capability(nil), requiredCapabilities...)
	p.updatedAt = now
	return true, nil
}

// Deactivate retires the path. Idempotent: deactivating an already
// deactivated path is a no-op success (matching the fleet's established
// "duplicate/redelivered command is a no-op, not an error" convention —
// see fulfillment-execution's WorkPool.Complete), so a retried operator
// action or a redelivered command never surfaces a spurious error.
func (p *ProcessPath) Deactivate(now time.Time) {
	if p.status == StatusDeactivated {
		return
	}
	p.status = StatusDeactivated
	p.updatedAt = now
}

func (p *ProcessPath) ID() shared.PathId   { return p.id }
func (p *ProcessPath) MatchPrefix() string { return p.matchPrefix }
func (p *ProcessPath) Direct() bool        { return p.direct }
func (p *ProcessPath) RequiredCapabilities() []shared.Capability {
	return append([]shared.Capability(nil), p.requiredCapabilities...)
}
func (p *ProcessPath) Status() Status       { return p.status }
func (p *ProcessPath) IsActive() bool       { return p.status == StatusActive }
func (p *ProcessPath) CreatedAt() time.Time { return p.createdAt }
func (p *ProcessPath) UpdatedAt() time.Time { return p.updatedAt }

func validate(matchPrefix string, requiredCapabilities []shared.Capability) error {
	if matchPrefix == "" {
		return ErrEmptyMatchPrefix
	}
	if matchPrefix != strings.ToLower(matchPrefix) {
		return ErrMatchPrefixNotLowercase
	}
	if len(requiredCapabilities) == 0 {
		return ErrNoRequiredCapabilities
	}
	return nil
}

func capabilitiesEqual(a, b []shared.Capability) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
