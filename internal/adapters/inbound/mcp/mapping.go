// Package mcp is the inbound Model Context Protocol adapter: it exposes
// this bounded context to the AI ecosystem as a second driving adapter
// over the same application-layer use cases the HTTP adapter uses. It is
// built on the official MCP Go SDK and served over Streamable HTTP.
//
// Per the fleet's now-established MCP pattern (see e.g.
// inventory-storage's ADR-0008), this package depends inward on the
// application layer (use cases) and the domain only — never on an
// outbound adapter. The composition root (cmd/mcp) wires the concrete
// ProcessPathRepo into the same GetPath/ListPaths use case structs
// cmd/pathmgmt already uses. Tool handlers call those use cases; domain
// structs never leak across the tool boundary.
package mcp

import (
	"context"

	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// GetPathQuery is the narrow read port the get_process_path tool needs.
// It is satisfied directly by *usecases.GetPath — no new outbound port is
// required, the use case's own Execute signature already fits.
type GetPathQuery interface {
	Execute(ctx context.Context, id shared.PathId) (*processpath.ProcessPath, error)
}

// ListPathsQuery is the narrow read port the list_process_paths tool
// needs. Satisfied directly by *usecases.ListPaths.
type ListPathsQuery interface {
	Execute(ctx context.Context, activeOnly bool) ([]*processpath.ProcessPath, error)
}

// processPathDTO is the tool-boundary representation of a ProcessPath
// aggregate: exactly the fields exposed by its own accessors (ID,
// MatchPrefix, Direct, RequiredCapabilities, Status, IsActive, CreatedAt,
// UpdatedAt), nothing more. Domain types never cross the tool boundary.
type processPathDTO struct {
	PathId               string   `json:"pathId"`
	MatchPrefix          string   `json:"matchPrefix"`
	Direct               bool     `json:"direct"`
	RequiredCapabilities []string `json:"requiredCapabilities"`
	Status               string   `json:"status"`
	Active               bool     `json:"active"`
	CreatedAt            string   `json:"createdAt"`
	UpdatedAt            string   `json:"updatedAt"`
}

// toProcessPathDTO folds a ProcessPath aggregate into the tool-boundary
// DTO via its own read accessors only.
func toProcessPathDTO(p *processpath.ProcessPath) processPathDTO {
	caps := p.RequiredCapabilities()
	capStrs := make([]string, 0, len(caps))
	for _, c := range caps {
		capStrs = append(capStrs, string(c))
	}
	return processPathDTO{
		PathId:               string(p.ID()),
		MatchPrefix:          p.MatchPrefix(),
		Direct:               p.Direct(),
		RequiredCapabilities: capStrs,
		Status:               string(p.Status()),
		Active:               p.IsActive(),
		CreatedAt:            p.CreatedAt().Format(timeLayout),
		UpdatedAt:            p.UpdatedAt().Format(timeLayout),
	}
}

// timeLayout is RFC 3339, matching what every other JSON boundary in this
// fleet uses for timestamps.
const timeLayout = "2006-01-02T15:04:05Z07:00"
