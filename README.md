# Process Path Management

> **⚠️ Study project.** This repository is an educational exercise in
> Domain-Driven Design applied to warehouse management/execution systems. It
> follows real industry-standard patterns and terminology (WMS/WES/WCS,
> CloudEvents-like envelopes, RFC 7807, hexagonal architecture) but is
> **not a production system** and is **not affiliated with, endorsed by, or
> representative of Amazon, Manhattan Associates, Blue Yonder, or any other
> company**.

The operator-configurable process-path catalogue for the `warehouse-systems`
fleet — a Generic Subdomain bounded context (same classification as
`facility-layout`), replacing a static YAML file
(`warehouse-infra/config/process-paths/sortable-fc.yaml`) previously
boot-loaded by `fulfillment-execution`, `wes-work-planning`, and
`workforce-management`.

📚 **Full documentation site:** https://claudioed.github.io/process-path-management/

## Why this context exists

Before this service existed, a path's canonical identity, its `matchPrefix`
match rule, and its required capabilities lived in a static YAML file
loaded once at boot by three separate services — a published language with
no single owner, no audit trail, and no way to revise it without a
coordinated redeploy of all three consumers. This service replaces that
file with a real bounded context: an aggregate, a REST API, and a
Kafka-published integration event. See
[ADR 0001](docs/docs/adr/0001-process-path-management-bounded-context.md)
for the full reasoning, including why propagation is Kafka-event-driven
rather than synchronous HTTP.

## Bounded-context boundary (read this first)

This service is the **SOURCE** of the process-path published language — it
has **no inbound Kafka consumer** and **no synchronous REST dependency** on
any other service. It publishes `ProcessPathCreated` / `ProcessPathUpdated`
/ `ProcessPathDeactivated` onto `warehouse.process-path-management.events`
when `EVENT_PUBLISHER=kafka`. `fulfillment-execution`, `wes-work-planning`
and `workforce-management` each consume that topic into a local catalogue
cache (cutover executed 2026-09-06, see ADR 0002). With a database
configured, events go through a **transactional outbox** — committed in
the same transaction as the aggregate and relayed to Kafka by an
in-process relay (ADR 0003) — so the store and the topic can never
diverge. See
[docs/docs/ecosystem/context-map.md](docs/docs/ecosystem/context-map.md)
for the full picture.

## Architecture

Hexagonal (ports & adapters), with a strict inward-only dependency rule —
**domain depends on nothing; application depends on domain; adapters
depend on application/domain** — identical in shape to every other service
in the fleet.

```
cmd/pathmgmt/                     main.go — the only composition root
internal/
  domain/
    processpath/                  ProcessPath aggregate
    shared/                       PathId, Capability, domain events
  application/
    ports/                        OUT: ProcessPathRepo, EventPublisher, Clock, PathMetrics
    usecases/                     DefinePath, RevisePath, DeactivatePath, GetPath, ListPaths
  adapters/
    inbound/http/                 chi handlers, DTOs, RFC 7807 error mapping
    outbound/postgres/            pgxpool repo, unit of work, outbox publisher + relay, golang-migrate runner
    outbound/memory/              in-memory repo for tests/local
    outbound/events/              log publisher (default)
    outbound/kafka/               Kafka publisher (EVENT_PUBLISHER=kafka)
    outbound/telemetry/           OTel traces/metrics/logs
  architecture/                   arch-go fitness tests
migrations/                       golang-migrate SQL files
apis/openapi.yaml                 This service's OWN REST API (6 endpoints)
apis/asyncapi.yaml                What this service PUBLISHES (publisher-side contract)
features/                         godog/Gherkin BDD acceptance tests
docker-compose.yml                Local Postgres 16
docs/docs/adr/                    Architecture Decision Records
```

The domain layer is pure Go: no `chi`, no `pgx`, no `kafka-go`. No JSON
struct tags in the domain packages.

## Business rules worth knowing before you read the code

- **A path's identity is permanent once created.** Re-defining an id that
  already exists (active or deactivated) is rejected with 409, never a
  silent overwrite.
- **`matchPrefix` must be non-empty and lower-case, validated (not
  coerced).** A request with `"PICK"` as its matchPrefix is rejected, not
  silently lower-cased.
- **`requiredCapabilities` must be non-empty.** A path with zero required
  capabilities is not a meaningful business fact.
- **Deactivation is terminal and idempotent.** A deactivated path can never
  be revised again (422), and deactivating an already-deactivated path is a
  no-op 204, never a spurious error or a double-published event.
- **A no-op revision does not republish `ProcessPathUpdated`.** Consumers
  never have to diff two identical payloads to notice nothing changed.

## Running locally

### 1. Without a database or a broker (fastest, verified working)

With no `DATABASE_URL`, the service starts on the in-memory adapter and is
fully functional over REST:

```bash
go run ./cmd/pathmgmt
# {"level":"INFO","msg":"DATABASE_URL not set, using in-memory ProcessPathRepo"}
# {"level":"INFO","msg":"http server listening","addr":":8080"}
```

### 2. With Postgres

```bash
docker compose up -d postgres          # Postgres 16 on localhost:5436

export DATABASE_URL='postgres://pathmgmt:pathmgmt@localhost:5436/pathmgmt?sslmode=disable'
go run ./cmd/pathmgmt                  # migrations run automatically at startup
```

### 3. With Kafka publishing enabled

By default this service logs its domain events instead of publishing them.
**Kafka publishing requires `EVENT_PUBLISHER=kafka`:**

```bash
export EVENT_PUBLISHER=kafka
export KAFKA_BROKERS=localhost:9092
go run ./cmd/pathmgmt
# {"level":"INFO","msg":"kafka event publishing enabled (direct, no outbox: DATABASE_URL not set)","brokers":["localhost:9092"],"topic":"warehouse.process-path-management.events"}
```

With `DATABASE_URL` also set, the use cases write events into the
`outbox_events` table inside the same transaction as the `process_paths`
change, and a relay goroutine drains that table onto the topic
(`"kafka event publishing enabled (transactional outbox)"` /
`"outbox relay running"` at startup). This is the mode the cluster runs.

### 4. With Docker

```bash
docker build -t process-path-management:local .
docker run --rm -p 8080:8080 \
  -e DATABASE_URL='postgres://pathmgmt:pathmgmt@host.docker.internal:5436/pathmgmt?sslmode=disable' \
  process-path-management:local
curl -s localhost:8080/healthz
```

### 5. With Helm (kind / any Kubernetes cluster)

```bash
helm install process-path-management charts/process-path-management \
  --set database.url='postgres://pathmgmt:pathmgmt@postgres:5432/pathmgmt?sslmode=disable'
kubectl port-forward svc/process-path-management 8080:80
curl -s localhost:8080/healthz
```

See `charts/process-path-management/values.yaml` for the full configuration
surface (`kafka.enabled`, `config.eventPublisher`, `otel.enabled`,
`ingress`, the additive `gatewayApi` HTTPRoute block, autoscaling, an
`existingSecret` pattern for `DATABASE_URL`).

### Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Listen address. |
| `DATABASE_URL` | *(unset)* | Postgres DSN. Unset ⇒ in-memory adapter. |
| `MIGRATIONS_PATH` | `migrations` | golang-migrate source directory. |
| `EVENT_PUBLISHER` | `log` | `log` (default) or `kafka`. Kafka publishing requires this to be set to `kafka`. |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated broker addresses (only read when `EVENT_PUBLISHER=kafka`). |
| `OUTBOX_RELAY_INTERVAL` | `1s` | How long the outbox relay sleeps between empty passes (only used when both `DATABASE_URL` and `EVENT_PUBLISHER=kafka` are set). |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://localhost:5189` | Comma-separated allowed origins. |
| `OTEL_SERVICE_NAME` | `process-path-management` | OTel `service.name` resource attribute. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP/gRPC Collector endpoint. |
| `LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error`. |

## API

Six endpoints. The full contract, including the RFC 7807 error schema, is in
[`apis/openapi.yaml`](apis/openapi.yaml).

| Method | Path | Use case |
| --- | --- | --- |
| `POST` | `/process-paths` | DefinePath |
| `GET` | `/process-paths` | ListPaths (`?all=true` for the audit view) |
| `GET` | `/process-paths/{pathId}` | GetPath |
| `PUT` | `/process-paths/{pathId}` | RevisePath |
| `DELETE` | `/process-paths/{pathId}` | DeactivatePath |
| `GET` | `/healthz` | Liveness probe |

Every error response is `application/problem+json` (RFC 7807), the same
shape every other service in this fleet emits.

Every route, including `/process-paths*`, is unauthenticated — there is no
REST auth layer in front of this API (see ADR 0005, which supersedes ADR
0004's earlier bearer-key adoption).

### Curl walkthrough

**Health:**

```bash
curl -s localhost:8080/healthz
# {"status":"ok"}
```

**Define a path:**

```bash
curl -s -X POST localhost:8080/process-paths \
  -H 'Content-Type: application/json' \
  -d '{"pathId":"PICK","matchPrefix":"pick","direct":true,"requiredCapabilities":["pick"]}'
# 201 Created
```

**List (active only by default):**

```bash
curl -s localhost:8080/process-paths
curl -s 'localhost:8080/process-paths?all=true'   # includes deactivated
```

**Revise:**

```bash
curl -s -X PUT localhost:8080/process-paths/PICK \
  -H 'Content-Type: application/json' \
  -d '{"matchPrefix":"pick-zone-a","requiredCapabilities":["pick","hazmat"]}'
# 200 OK
```

**Deactivate:**

```bash
curl -s -X DELETE localhost:8080/process-paths/PICK
# 204 No Content
```

## Quality gate

Every `make` target mirrors a step in `.github/workflows/ci.yml`, so the
same feedback CI gives you post-push is available locally, pre-commit:

```bash
make check       # fmt-check + vet + build + lint + test -race
make check-all   # check + coverage (gate: 90% on domain + application)
```

| Target | What it runs |
| --- | --- |
| `build` | `go build ./...` |
| `vet` | `go vet ./...` |
| `fmt` / `fmt-check` | `gofmt -w .` / fail if `gofmt -l .` is non-empty |
| `lint` | `golangci-lint run ./...` (CI pins `v2.13.1`) |
| `test` | `go test ./... -race` |
| `coverage` | coverage profile + the 90% gate |
| `bdd` | `go test ./... -run TestFeatures -v` (godog/Gherkin) |
| `arch-test` | `go test ./internal/architecture/... -v` (arch-go fitness) |
| `integration` | `go test -tags=integration ./... -race -count=1` (needs `DATABASE_URL`) |
| `mutation` | `gremlins unleash ./internal/domain` (see `.gremlins.yaml`) |

Additional verification surfaces, each with its own CI job:

```bash
go test ./... -run TestFeatures -v                  # BDD (godog/Gherkin) — 11/11 scenarios pass
go test ./internal/architecture/... -v               # arch-fitness (arch-go)
go test -tags=integration ./... -race -count=1       # Postgres integration
gremlins unleash ./internal/domain --workers 1 --timeout-coefficient 30   # mutation testing
helm lint charts/process-path-management
spectral lint apis/openapi.yaml --ruleset .spectral.yaml --fail-severity=warn
spectral lint apis/asyncapi.yaml --ruleset .spectral.asyncapi.yaml --fail-severity=warn
```

Git hooks are wired through [lefthook](https://github.com/evilmartians/lefthook)
— `pre-commit` runs fmt-check/vet/lint, `pre-push` runs `make check`. Hooks
are not tracked by git, so activate them once per clone:

```bash
brew install lefthook   # or: go install github.com/evilmartians/lefthook@latest
lefthook install
```

CI (`.github/workflows/ci.yml`) runs the full fleet-standard matrix:
**`lint`**, **`test`**, **`bdd`**, **`integration`** (Postgres service
container), **`mutation-fast`** (blocking, `./internal/domain`),
**`api-lint`** (Spectral against both `apis/openapi.yaml` and
`apis/asyncapi.yaml`), **`vuln`** (govulncheck), **`arch-test`** (arch-go
fitness tests), **`helm-lint`**/**`trivy-scan`** (gated to
pull-request-targeting-`main` only, per this fleet's convention —
this repo has no `main`-targeting PR yet, so these two jobs are expected
to skip, not fail), **`docker-publish`** (main-only, cosign keyless signing
+ SPDX SBOM attestation), and **`release`** (main-only, auto-tagged
GitHub release + published Helm chart). Plus `.github/workflows/codeql.yml`
(security-extended CodeQL analysis) and `.github/workflows/scorecard.yml`
(OpenSSF Scorecard).

**Mutation testing baseline (measured, not fabricated):** run locally on
2026-09-05 against `./internal/domain` (the only aggregate in this
service's domain layer today) — **11 mutants total, all killed, zero
survivors: 100.00% efficacy, 100.00% mutator coverage.** `.gremlins.yaml`
sets the threshold to 99 (strictly below the measured 100%), the same
"lock in today's quality" philosophy every other repo in this fleet's
`.gremlins.yaml` uses.

## Helm chart

`charts/process-path-management/` ships with BOTH a default-enabled-capable
`ingress.yaml` (disabled by default, `ingress.enabled=false`) AND an
additive `gatewayApi`/`httproute.yaml` template (also disabled by default),
matching the dual ingress/Gateway API pattern this fleet's other charts
have all migrated to. `helm lint` and two real `helm template` renders
(default `Ingress`, and `--set gatewayApi.enabled=true` with a populated
`parentRefs`/`hosts` value producing a correctly-shaped `HTTPRoute` with
`sectionName: http`) were verified locally before this chart shipped.

## Known gaps

- **No consumer wired in any of the three intended downstream repos.**
  `fulfillment-execution`, `wes-work-planning`, and `workforce-management`
  do not yet have a Kafka consumer for
  `warehouse.process-path-management.events` — this service's publisher is
  real and tested, but genuinely unconsumed today. See
  [docs/docs/ecosystem/context-map.md](docs/docs/ecosystem/context-map.md).

## Architecture Decision Records

1. [0001 — Process Path Management as a new Generic Subdomain bounded context](docs/docs/adr/0001-process-path-management-bounded-context.md)
2. [0002 — Cutting the fleet over from the static YAML catalogue to this service's events](docs/docs/adr/0002-yaml-to-kafka-cutover.md)
3. [0003 — Transactional outbox for the process-path Published Language](docs/docs/adr/0003-transactional-outbox.md)
4. [0004 — Adopting the fleet REST identity standard (static bearer keys, read/read-write scopes)](docs/docs/adr/0004-rest-auth-adoption.md) (superseded by 0005)
5. [0005 — Removing the REST auth layer](docs/docs/adr/0005-remove-rest-auth.md)

## License

MIT (or match the other repos' licensing — TBD).
