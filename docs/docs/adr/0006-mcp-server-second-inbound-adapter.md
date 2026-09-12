---
id: 0006-mcp-server-second-inbound-adapter
slug: /adr/0006-mcp-server-second-inbound-adapter
title: 0006. MCP server as a second inbound adapter
sidebar_label: 0006. MCP server adapter
description: ADR 0006 — this service gains a Model Context Protocol inbound adapter (cmd/mcp), exposing the process-path catalogue read-only to AI tooling over the same GetPath and ListPaths use cases the HTTP adapter already uses.
---

# 0006. MCP server as a second inbound adapter

## Status

Accepted — implemented in the same change that introduced this record.

## Context

This service — the fleet's Open Host Service / Published Language SOURCE
for `warehouse.process-path-management.events` — had **zero MCP
presence**: no `cmd/mcp`, no `internal/adapters/inbound/mcp` package. Five
other bounded contexts in this fleet (fulfillment-execution,
wes-work-planning, inventory-storage, workforce-management,
facility-layout) already ship an MCP inbound adapter following the same
established shape (each context's own `ADR-0008`-equivalent), so AI
tooling (warehouse-ops-agent and any other MCP client) can query those
contexts directly. This service was the one gap: the process-path
catalogue — the exact definitions every one of those five contexts either
already consumes or is meant to consume — had no equivalent read surface
for AI tooling, only its REST API.

The catalogue is also small, stable, read-mostly data (a path's identity,
match prefix, direct flag, required capabilities, and lifecycle status) —
a natural fit for exposing read-only, with no risk profile that write
tools would carry.

## Decision

Add `cmd/mcp` as a second, independent composition root and deployable,
mirroring `cmd/pathmgmt`'s own wiring style (same `DATABASE_URL`-present-
or-absent mode switch between the Postgres and in-memory
`ProcessPathRepo`, same non-blocking OTel setup, same `getenv` helper
pattern):

- **New package `internal/adapters/inbound/mcp/`** (sibling to
  `internal/adapters/inbound/http/`). It depends inward on
  `internal/application/usecases`, `internal/application/ports`, and
  `internal/domain/**` only — never on `internal/adapters/outbound/**` —
  enforced by this repo's own arch-go fitness test
  (`internal/architecture/architecture_test.go`, `make arch-test`), which
  passes unchanged with this package added.
- **No new use cases.** The adapter reuses the exact same
  `usecases.GetPath` and `usecases.ListPaths` structs `cmd/pathmgmt`
  already wires into the HTTP adapter, via two narrow local read-port
  interfaces (`GetPathQuery`, `ListPathsQuery`) satisfied directly by
  those structs — no new outbound port was needed, since their existing
  `Execute` signatures already fit.
- **Two read-only tools for v1**, matching the fleet-wide convention that
  every bounded context's own MCP server ships read-only tools first:
  - `get_process_path(pathId)` → the full `ProcessPath` definition
    (`pathId`, `matchPrefix`, `direct`, `requiredCapabilities`, `status`,
    `active`, `createdAt`, `updatedAt`) via `GetPath`, or a tool-level
    error when `usecases.ErrPathNotFound`.
  - `list_process_paths(activeOnly=true)` → an array of the same DTO via
    `ListPaths`.

  Both tools are exactly the `ProcessPath` aggregate's own accessors —
  `ID()`, `MatchPrefix()`, `Direct()`, `RequiredCapabilities()`,
  `Status()`, `IsActive()`, `CreatedAt()`, `UpdatedAt()` — nothing more,
  and both carry `mcp.ToolAnnotations{ReadOnlyHint: true}`.
- **No auth layer.** The fleet-wide static-bearer REST+MCP auth standard
  (adopted here in ADR 0004) was rolled back fleet-wide in ADR 0005; this
  service's own REST API and its new MCP server are both mounted
  unauthenticated, matching the current live convention across every
  sibling context's own `cmd/mcp`.
- **Streamable HTTP**, served via
  `mcp.NewStreamableHTTPHandler`, mounted at both `/` and `/mcp` (the
  fleet's `*_MCP_ENDPOINT` convention), plus an unauthenticated
  `GET /healthz` for the Kubernetes probes.
- **Helm**: `charts/process-path-management` gains
  `templates/mcp-deployment.yaml`, `templates/mcp-service.yaml`, the
  `process-path-management.mcpFullname` helper, and a `mcp:` values
  block, all following the exact reference pattern
  inventory-storage's chart already ships (`mcp.enabled: false` by
  default).

## Consequences

**Positive**
- AI tooling gains a direct, read-only view of the process-path catalogue
  — the same definitions fulfillment-execution, wes-work-planning, and
  workforce-management consume from Kafka — without needing to speak this
  service's REST API or duplicate its DTO mapping.
- Zero duplication of business logic: both tools are thin wrappers over
  the same use cases the HTTP adapter already exercises and tests.
- The hexagonal boundary is unchanged and independently verified —
  `internal/adapters/inbound/mcp` cannot reach into
  `internal/adapters/outbound/**`, so a future MCP write tool (should one
  ever be added) will have to go through a use case, never around one.

**Negative / accepted**
- A second deployable to build, chart, and operate (mirrors every sibling
  context that already made this trade).
- v1 is read-only by design; if an MCP write tool is ever proposed
  (e.g. `deactivate_process_path`), it should follow the exact same
  reuse-the-use-case pattern this ADR establishes, with its own explicit
  `DestructiveHint`/`IdempotentHint` annotations — not bundled into this
  change.
