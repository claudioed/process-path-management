package shared

import "time"

// DomainEvent is the minimal shape every event raised by this domain
// implements — mirrors the same lightweight interface used across the
// fleet's other services (see e.g. labor-performance's domain/shared
// package), which the outbound Kafka adapter wraps in a CloudEvents
// envelope, not the domain layer.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// ProcessPathCreated is raised the first time a PathId is defined. Carries
// enough of the definition for a consumer to build its local read model
// without a follow-up query — mirrors this fleet's existing pattern
// (e.g. TaskCompleted's own self-sufficiency) of events being usable
// standalone rather than as change-notification-plus-refetch.
type ProcessPathCreated struct {
	PathId               PathId
	MatchPrefix          string
	Direct               bool
	RequiredCapabilities []Capability
	At                   time.Time
}

func (e ProcessPathCreated) EventName() string     { return "ProcessPathCreated" }
func (e ProcessPathCreated) OccurredAt() time.Time { return e.At }

// ProcessPathUpdated is raised when an ACTIVE path's matchPrefix or
// requiredCapabilities is revised. Not raised for a no-op update (see
// ProcessPath.Revise's early-return) — consumers should not have to
// diff two payloads to notice nothing changed.
type ProcessPathUpdated struct {
	PathId               PathId
	MatchPrefix          string
	Direct               bool
	RequiredCapabilities []Capability
	At                   time.Time
}

func (e ProcessPathUpdated) EventName() string     { return "ProcessPathUpdated" }
func (e ProcessPathUpdated) OccurredAt() time.Time { return e.At }

// ProcessPathDeactivated is raised when a path is retired. Consumers
// (fulfillment-execution, wes-work-planning, workforce-management) must
// stop accepting NEW work against this path once they observe this event,
// but this service takes no position on in-flight work already assigned
// to it — that is each consumer's own operational concern, exactly the
// same "read model, not a command" posture ADR-0002 (workforce-management)
// established for PathUnderstaffed.
type ProcessPathDeactivated struct {
	PathId PathId
	At     time.Time
}

func (e ProcessPathDeactivated) EventName() string     { return "ProcessPathDeactivated" }
func (e ProcessPathDeactivated) OccurredAt() time.Time { return e.At }
