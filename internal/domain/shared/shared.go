// Package shared holds value objects and identity types used across the
// process-path-management domain — nothing here is an aggregate.
package shared

import "errors"

// PathId is the canonical identity of a process path (e.g. "PICK", "PACK",
// "REBIN", "SLAM"). It is the SAME identity fulfillment-execution's
// task.Type, wes-work-planning's WorkPool.PathId, and workforce-management's
// PathPlan.PathId all reference — this service is the one place that
// identity is DEFINED, not just consumed. Kept as a plain string type
// (not an enum) because the whole point of this service existing is that
// the valid set is operator-configurable, not compiled in.
type PathId string

// Capability is a named qualification a station/associate must hold to
// work a process path (e.g. "pick", "pack", "hazmat") — the exact same
// vocabulary workforce-management's Certification and
// fulfillment-execution's Station.Capability already use. This service
// does not invent a new capability vocabulary; it is the authoritative
// SOURCE for which capabilities a given path requires, so the existing
// vocabulary is carried here as a plain string, not redefined.
type Capability string

// DestinationLocationRole is an OPTIONAL declaration of what kind of
// facility-layout LocationRole a completed task on this path is destined
// for — e.g. a Pack path's output goes to a Drop location, a QC path's
// output goes to a WorkCenter, an outbound Rebin path's output goes to
// Shipping (ADR 0006, Phase B3 follow-on from facility-layout's own
// ADR-0016/0017).
//
// This is DECLARATIVE ROUTING INTENT ONLY — a fact this service publishes
// for a downstream consumer (e.g. fulfillment-execution routing a
// completed task, or a future console rendering "where does this path's
// output go") to interpret. This service never calls facility-layout to
// validate the value against a real, currently-modeled LocationRole: it
// has zero inbound dependency and never calls into any other service,
// synchronously or otherwise (see AGENTS.md's "What this service
// deliberately does not own"). The recognized value set below is a local,
// closed vocabulary matching facility-layout's own LocationRole enum
// (ADR-0016 there) — kept in sync by convention, not by a live lookup.
type DestinationLocationRole string

const (
	// DestinationLocationRoleUnset means no destination role was
	// declared for this path — the zero value, and a fully valid,
	// permanent state (most paths, e.g. Pick, have no single
	// destination role; their output goes wherever the next process
	// path in the chain sends it).
	DestinationLocationRoleUnset DestinationLocationRole = ""

	DestinationLocationRoleDrop       DestinationLocationRole = "Drop"
	DestinationLocationRoleWorkCenter DestinationLocationRole = "WorkCenter"
	DestinationLocationRoleShipping   DestinationLocationRole = "Shipping"
)

// ErrInvalidDestinationLocationRole is returned when a non-empty
// DestinationLocationRole is not one of the recognized values above.
var ErrInvalidDestinationLocationRole = errors.New("shared: destinationLocationRole must be one of Drop, WorkCenter, Shipping, or empty")

// ParseDestinationLocationRole validates s against the recognized value
// set. An empty string is always valid (DestinationLocationRoleUnset) —
// declaring a destination role is optional, not a required field on
// every path.
func ParseDestinationLocationRole(s string) (DestinationLocationRole, error) {
	switch DestinationLocationRole(s) {
	case DestinationLocationRoleUnset, DestinationLocationRoleDrop, DestinationLocationRoleWorkCenter, DestinationLocationRoleShipping:
		return DestinationLocationRole(s), nil
	default:
		return "", ErrInvalidDestinationLocationRole
	}
}
