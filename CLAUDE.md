# Project: Process Path Management (Generic Subdomain)

> This is the **ninth** bounded-context Go service in the warehouse-systems
> fleet (after order-management, inventory-storage, wes-work-planning,
> workforce-management, fulfillment-execution, facility-layout,
> warehouse-ops-agent, labor-performance). See ADR 0001 for the full
> decision record.

The fleet's **operator-configurable process-path catalogue**: a path's
canonical identity (`PathId`, e.g. `PICK`, `PACK`, `REBIN`, `SLAM`), the
`matchPrefix` rule downstream consumers use to resolve a caller-supplied id
to a path family, whether it is `Direct`, and the capabilities a
station/associate must hold to work it. Before this service existed, that
definition lived in a static YAML file
(`warehouse-infra/config/process-paths/sortable-fc.yaml`) loaded once at
boot by `fulfillment-execution`, `wes-work-planning`, and
`workforce-management` — a published language with no single owner, no
audit trail, and no way to revise without a coordinated redeploy of all
three consumers. This service replaces that file as the single source of
truth.

## Strategic classification (read this before writing any code)

**Generic Subdomain**, the same bucket as `facility-layout` — well
understood, not a competitive differentiator, but needed identically by
multiple existing contexts, so it is extracted rather than duplicated or
left as an unowned static file (ADR 0001).

This service is the **Open Host Service / Published Language SOURCE** for
`warehouse.process-path-management.events`, with `fulfillment-execution`
(Core), `wes-work-planning` (Core), and `workforce-management`
(Supporting) as its **Conformist** consumers. It has **zero inbound
dependency** — no inbound Kafka consumer, no synchronous REST dependency
in either direction. It never calls into any other service, synchronously
or otherwise; every change propagates exclusively via Kafka.

**Known, honestly-documented gap:** as of the last ADR update, none of the
three intended consumers has a Kafka consumer wired to this topic yet —
see `docs/docs/ecosystem/context-map.md` for the current, real state
before assuming an integration is live. Verify against that file (or the
sibling repos directly) rather than assuming this note is still current.

## Architecture (NON-NEGOTIABLE)

Hexagonal / Ports & Adapters. Strict inward-only dependency rule —
**domain depends on nothing; application depends on domain; adapters
depend on application/domain** — enforced by `internal/architecture/architecture_test.go`
(arch-go fitness tests, `make arch-test`), not just convention:

- domain (`internal/domain/**`) may only depend on other domain packages.
- application (`internal/application/**`) may only depend on domain +
  application.
- inbound adapters never depend on outbound adapters, and vice versa.
- nothing under `internal/**` may import `cmd/**` — `cmd` is a leaf
  composition root, never a dependency of the layers it wires.

No framework, HTTP, Kafka, or SQL types in the domain layer. No JSON
struct tags in the domain packages.

```
cmd/pathmgmt/                     main.go — the only composition root
internal/
  domain/
    processpath/                  ProcessPath aggregate (process_path.go)
    shared/                       PathId, Capability value objects; domain events
  application/
    ports/                        OUT: ProcessPathRepo, EventPublisher, UnitOfWork, Clock, PathMetrics
    usecases/                     DefinePath, RevisePath, DeactivatePath, GetPath, ListPaths
  adapters/
    inbound/http/                 chi handlers, DTOs, RFC 7807 error mapping (server.go)
    outbound/postgres/            pgxpool repo, unit of work, outbox publisher + relay, golang-migrate runner
    outbound/memory/               in-memory repo for tests/local (also the zero-DATABASE_URL runtime path)
    outbound/events/               log publisher (default when EVENT_PUBLISHER != kafka)
    outbound/kafka/                Kafka publisher (EVENT_PUBLISHER=kafka), topic constant
    outbound/telemetry/            OTel traces/metrics/logs
  architecture/                    arch-go fitness tests (architecture_test.go)
migrations/                        golang-migrate SQL files (0001_init, 0002_outbox)
apis/openapi.yaml                  This service's OWN REST API (6 endpoints)
apis/asyncapi.yaml                 What this service PUBLISHES (publisher-side contract only — no consumer side)
features/                          godog/Gherkin BDD acceptance tests
web/                                process-path-mfe: Vite + React Module Federation remote (operator SPA)
docker-compose.yml                  Local Postgres 16
docs/docs/adr/                      Architecture Decision Records (Nygard format)
```

This service has **no inbound Kafka consumer package** — unlike most of
the fleet, it never subscribes to anyone else's topic. Its only Kafka role
is the outbound publisher (`internal/adapters/outbound/kafka`).

### Persistence and event delivery mode matrix

Composition root (`cmd/pathmgmt/main.go`) picks the mode at boot from
`DATABASE_URL` and `EVENT_PUBLISHER`:

| `DATABASE_URL` | `EVENT_PUBLISHER` | Publisher wired          | Outbox relay |
|----------------|--------------------|---------------------------|--------------|
| unset          | `log` (default)     | log                       | none         |
| unset          | `kafka`             | direct Kafka (no outbox)  | none         |
| set            | `log`                | log                       | none         |
| set            | `kafka`             | **transactional outbox**  | **yes**      |

The cluster runs the last row. With no `DATABASE_URL`, the service is
fully functional over REST on the in-memory adapter (`internal/adapters/outbound/memory`)
— no Postgres required for local dev.

### Transactional outbox (ADR 0003 — this repo is the fleet's REFERENCE implementation)

Every write use case (`DefinePath`, `RevisePath`, `DeactivatePath`) runs
`Repo.Save` and `EventPublisher.Publish` inside one `ports.UnitOfWork.Execute`
scope. `postgres.UnitOfWork` opens a `pgx.Tx`, binds it to the context, and
commits or rolls back around the callback — either both the aggregate row
and the `outbox_events` row land, or neither does. `postgres.OutboxRelay`
runs as a goroutine inside the `pathmgmt` process (not a separate binary),
claiming up to 100 pending rows with `SELECT … FOR UPDATE SKIP LOCKED ORDER BY id`
per pass, sending them to Kafka in order, marking `published_at`. This
pattern is the template four sibling services (wes-work-planning,
fulfillment-execution, workforce-management, labor-performance) are meant
to follow — see ADR 0003 for the full rationale, including the concrete
store/topic divergence bug (ADR 0002) that motivated it.

## Key Commands

Every `make` target mirrors a step in `.github/workflows/ci.yml`:

```bash
make check         # FAST bundle: fmt-check vet build lint test — run after every change
make check-all      # check + coverage (90% gate on domain + application) — run before pushing
make build           # go build ./...
make vet             # go vet ./...
make fmt             # gofmt -w . (in place)
make fmt-check       # fail if gofmt -l . is non-empty
make lint            # golangci-lint run ./... (CI pins v2.13.1)
make test            # go test ./... -race
make coverage        # coverage profile + the 90% threshold gate
make bdd             # go test ./... -run TestFeatures -v (godog/Gherkin)
make arch-test       # go test ./internal/architecture/... -v (hexagonal dependency rule)
make integration     # go test -tags=integration ./... -race -count=1 (needs DATABASE_URL; testcontainers-backed)
make mutation        # gremlins unleash ./internal/domain (see .gremlins.yaml, threshold 99%)
```

Additional verification surfaces with their own CI job, run manually when
touching that surface:

```bash
spectral lint apis/openapi.yaml --ruleset .spectral.yaml --fail-severity=warn
spectral lint apis/asyncapi.yaml --ruleset .spectral.asyncapi.yaml --fail-severity=warn
helm lint charts/process-path-management
govulncheck ./...
```

Local run, no dependencies:

```bash
go run ./cmd/pathmgmt
# DATABASE_URL not set -> in-memory ProcessPathRepo; fully functional REST on :8080
```

With Postgres:

```bash
docker compose up -d postgres          # Postgres 16 on localhost:5436
export DATABASE_URL='postgres://pathmgmt:***@localhost:5436/pathmgmt?sslmode=disable'
go run ./cmd/pathmgmt                  # migrations run automatically at startup
```

With Kafka publishing:

```bash
export EVENT_PUBLISHER=kafka
export KAFKA_BROKERS=localhost:9092
go run ./cmd/pathmgmt
```

Docs site (Docusaurus, generated OpenAPI reference pages):

```bash
cd docs && npm ci && npm run gen-api-docs pathmgmt   # regenerate docs/docs/api-reference/rest/* from apis/openapi.yaml
npm run build                                          # full site build, verifies no broken links
```

Git hooks via [lefthook](https://github.com/evilmartians/lefthook) — not
tracked by git, activate once per clone: `lefthook install`. `pre-commit`
runs fmt-check/vet/lint; `pre-push` runs `make check`.

## Further reading (`.claude/rules/`)

Full ubiquitous language, aggregate invariants, and domain events:
`.claude/rules/domain-model.md`.

Full REST API endpoint table, testing discipline, and CI job matrix:
`.claude/rules/testing-and-api.md`.

## Docs site and GitFlow (repo-specific — do not "fix" without checking first)

- GitFlow: `develop` is the working branch; `main` is release-only,
  synced by explicit fast-forward.
- `.github/workflows/docs.yml` triggers on **push to `develop`** with
  paths `docs/**` (not `main`, and not on every push) — this is
  deliberate for this repo (its GitHub Pages `github-pages` deployment
  environment only allows the `develop` branch), matching every other
  service's docs pipeline in this fleet. Do not "fix" this to trigger off
  `main`.
- `docs/package.json`'s `gen-api-docs` script (`docusaurus gen-api-docs
  pathmgmt`) regenerates `docs/docs/api-reference/rest/*.api.mdx` from
  `apis/openapi.yaml`. Whenever `apis/openapi.yaml` changes (new
  operation, changed request/response shape, changed
  summary/description), regenerate and commit the `.mdx`/`.json`
  companions — they are committed generated output, not hand-written.

## What this service deliberately does not own

- Does not decide dispatch, routing, or task assignment — defines WHAT a
  path is and WHICH capabilities it requires; never claims, assigns, or
  completes work. That is `fulfillment-execution`'s job.
- Does not call any consumer synchronously, ever.
- Has no inbound Kafka consumer of its own.
