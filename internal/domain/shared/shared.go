// Package shared holds value objects and identity types used across the
// process-path-management domain — nothing here is an aggregate.
package shared

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
