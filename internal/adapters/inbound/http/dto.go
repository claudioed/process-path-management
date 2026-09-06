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
}

// reviseProcessPathRequest is the PUT /process-paths/{pathId} request
// body. PathId and Direct are not revisable (see the aggregate's own doc
// comment on why Direct is immutable), so this DTO deliberately does not
// carry them.
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
	Status               string   `json:"status"`
	CreatedAt            string   `json:"createdAt"`
	UpdatedAt            string   `json:"updatedAt"`
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
