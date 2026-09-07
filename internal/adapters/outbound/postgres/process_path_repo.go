package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// ProcessPathRepo is a pgxpool-backed implementation of
// ports.ProcessPathRepo. RequiredCapabilities is stored as a Postgres
// text[] column — a real array type, not a serialized JSON blob, so it
// stays queryable (e.g. "which paths require hazmat") without a JSON
// operator.
type ProcessPathRepo struct {
	pool *pgxpool.Pool
}

// NewProcessPathRepo constructs a ProcessPathRepo over pool.
func NewProcessPathRepo(pool *pgxpool.Pool) *ProcessPathRepo {
	return &ProcessPathRepo{pool: pool}
}

func (r *ProcessPathRepo) Save(ctx context.Context, p *processpath.ProcessPath) error {
	_, err := querierFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO process_paths (id, match_prefix, direct, required_capabilities, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		  SET match_prefix          = EXCLUDED.match_prefix,
		      required_capabilities = EXCLUDED.required_capabilities,
		      status                = EXCLUDED.status,
		      updated_at            = EXCLUDED.updated_at
	`, string(p.ID()), p.MatchPrefix(), p.Direct(), capabilitiesToStrings(p.RequiredCapabilities()), string(p.Status()), p.CreatedAt(), p.UpdatedAt())
	return err
}

func (r *ProcessPathRepo) FindByID(ctx context.Context, id shared.PathId) (*processpath.ProcessPath, error) {
	row := querierFrom(ctx, r.pool).QueryRow(ctx, `
		SELECT id, match_prefix, direct, required_capabilities, status, created_at, updated_at
		FROM process_paths
		WHERE id = $1
	`, string(id))
	return scanProcessPath(row)
}

func (r *ProcessPathRepo) ListActive(ctx context.Context) ([]*processpath.ProcessPath, error) {
	return r.list(ctx, `
		SELECT id, match_prefix, direct, required_capabilities, status, created_at, updated_at
		FROM process_paths
		WHERE status = 'ACTIVE'
		ORDER BY id
	`)
}

func (r *ProcessPathRepo) ListAll(ctx context.Context) ([]*processpath.ProcessPath, error) {
	return r.list(ctx, `
		SELECT id, match_prefix, direct, required_capabilities, status, created_at, updated_at
		FROM process_paths
		ORDER BY id
	`)
}

func (r *ProcessPathRepo) list(ctx context.Context, query string) ([]*processpath.ProcessPath, error) {
	rows, err := querierFrom(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*processpath.ProcessPath
	for rows.Next() {
		p, err := scanProcessPath(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanProcessPath(row pgx.Row) (*processpath.ProcessPath, error) {
	var (
		id                   string
		matchPrefix          string
		direct               bool
		requiredCapabilities []string
		status               string
		createdAt, updatedAt time.Time
	)
	err := row.Scan(&id, &matchPrefix, &direct, &requiredCapabilities, &status, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return processpath.Rehydrate(
		shared.PathId(id),
		matchPrefix,
		direct,
		stringsToCapabilities(requiredCapabilities),
		processpath.Status(status),
		createdAt,
		updatedAt,
	), nil
}

func capabilitiesToStrings(caps []shared.Capability) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = string(c)
	}
	return out
}

func stringsToCapabilities(ss []string) []shared.Capability {
	out := make([]shared.Capability, len(ss))
	for i, s := range ss {
		out[i] = shared.Capability(s)
	}
	return out
}
