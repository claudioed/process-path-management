# Testing discipline, CI matrix, and REST API reference

## REST API (inbound adapter — `internal/adapters/inbound/http`)

Six endpoints, all unauthenticated (ADR 0005 removed the REST auth layer
ADR 0004 had added — do not re-add auth without checking that ADR first):

| Method | Path | Use case |
| --- | --- | --- |
| `POST` | `/process-paths` | DefinePath — 201, 409 if id exists, 422 on invariant violation |
| `GET` | `/process-paths` | ListPaths — active-only default, `?all=true` for audit view |
| `GET` | `/process-paths/{pathId}` | GetPath — regardless of status |
| `PUT` | `/process-paths/{pathId}` | RevisePath — 422 if deactivated |
| `DELETE` | `/process-paths/{pathId}` | DeactivatePath — 204, idempotent |
| `GET` | `/healthz` | Liveness probe |

Every error is `application/problem+json` (RFC 7807), the fleet-standard
shape. `apis/openapi.yaml` is the full contract, Spectral-linted in CI
(`api-lint` job) — treat it as authoritative over any hand description
here, including this file.

## Testing Discipline

- `make check` after every change (fmt-check, vet, build, lint, test
  -race); `make check-all` (adds the 90% coverage gate on
  `./internal/domain/...` and `./internal/application/...`) before
  pushing.
- **Unit tests are colocated** (`process_path_test.go`, `events_test.go`,
  `usecases_test.go`, `unit_of_work_test.go`, `fakes_test.go`,
  `server_test.go`).
- **BDD**: `features/*.feature` (godog/Gherkin) — `define_and_read.feature`,
  `revise.feature`, `deactivate.feature`, driven through `features_test.go`.
  Run via `make bdd`.
- **Architecture fitness**: `internal/architecture/architecture_test.go`
  (arch-go) enforces the hexagonal dependency rule described above —
  every PR that reshapes package imports must keep this green, not just
  `go build`.
- **Integration** (`-tags=integration`, gated behind `DATABASE_URL`):
  `internal/adapters/outbound/postgres/*_integration_test.go` — the fleet
  convention is testcontainers-backed, never a skip-gated external broker
  (see the fleet-ops skill's Kafka-integration-testcontainers pitfall if
  adding Kafka-touching integration coverage here). Files behind
  `//go:build integration` are invisible to default `go build`/`go test
  ./...` — always also run `go build -tags=integration ./...` and `go vet
  -tags=integration ./...` before pushing any change to a constructor or
  interface signature integration tests call.
- **Mutation testing**: `.gremlins.yaml` targets `./internal/domain` only,
  threshold 99% (measured baseline was 100%, 11/11 mutants killed on
  2026-09-05) — "lock in today's quality," not an aspirational bar. Run
  via `make mutation`.
- **API contract linting**: `spectral lint` against both
  `apis/openapi.yaml` (`.spectral.yaml`) and `apis/asyncapi.yaml`
  (`.spectral.asyncapi.yaml`), CI job `api-lint`.
- CI (`.github/workflows/ci.yml`) full matrix: `lint`, `test`, `bdd`,
  `integration` (Postgres service container), `mutation-fast` (blocking),
  `api-lint`, `vuln` (govulncheck), `arch-test`, `helm-lint`/`trivy-scan`
  (gated to PRs targeting `main` only), `docker-publish` and `release`
  (main-only). Plus `.github/workflows/codeql.yml` and
  `.github/workflows/scorecard.yml`.
