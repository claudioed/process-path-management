// Package http is the inbound chi adapter: DTOs, handlers, routing, and
// domain-error-to-HTTP-status mapping. Domain structs never cross this
// boundary — every response below is a DTO owned by this package.
package http

// defineProcessPathRequest is the POST /process-paths request body.
type defineProcessPathRequest struct {
	PathId               string   `json:"pathId"`
	MatchPrefix          string   `json:"matchPrefix"`
	Direct               bool     `json:"direct"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
	// DestinationLocationRole is an OPTIONAL declaration of what kind of
	// facility-layout LocationRole this path's completed work is
	// destined for (Drop | WorkCenter | Shipping) — see ADR 0006. Omitted
	// entirely means "no destination role declared", the default and
	// most common case.
	DestinationLocationRole string `json:"destinationLocationRole,omitempty"`
}

// reviseProcessPathRequest is the PUT /process-paths/{pathId} request
// body. PathId, Direct, and destinationLocationRole are not revisable
// (see the aggregate's own doc comment on why they are immutable), so
// this DTO deliberately does not carry them.
type reviseProcessPathRequest struct {
	MatchPrefix          string   `json:"matchPrefix"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
}

// processPathResponse is the response body for every endpoint that
// returns one ProcessPath.
type processPathResponse struct {
	PathId               string   `json:"pathId"`
	MatchPrefix          string   `json:"matchPrefix"`
	Direct               bool     `json:"direct"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
	// DestinationLocationRole is omitted entirely (not defaulted to "")
	// when no destination role was declared for this path — mirrors this
	// fleet's omit-when-unknown discipline (e.g. wes-work-planning's
	// travelDistanceM, ADR-0017 there).
	DestinationLocationRole string `json:"destinationLocationRole,omitempty"`
	Status                  string `json:"status"`
	CreatedAt               string `json:"createdAt"`
	UpdatedAt               string `json:"updatedAt"`
}

// problemDetails is the RFC 7807 (Problem Details for HTTP APIs) response
// body for every error this API returns.
type problemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}
