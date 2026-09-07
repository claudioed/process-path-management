package usecases

import (
	"context"

	"github.com/claudioed/process-path-management/internal/application/ports"
	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// RevisePath updates an existing Active path's matchPrefix/
// requiredCapabilities and publishes ProcessPathUpdated -- but ONLY if
// the revision actually changes something (see ProcessPath.Revise's
// changed return value). A no-op revision request is a successful no-op,
// not an error and not a spuriously republished event.
type RevisePath struct {
	Repo      ports.ProcessPathRepo
	Publisher ports.EventPublisher
	Clock     ports.Clock
	// UnitOfWork brackets Save + Publish atomically (ADR 0003); nil means
	// no transactional backing (see DefinePath).
	UnitOfWork ports.UnitOfWork
}

func (uc *RevisePath) Execute(ctx context.Context, id shared.PathId, matchPrefix string, requiredCapabilities []shared.Capability) (*processpath.ProcessPath, error) {
	p, err := uc.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPathNotFound
	}

	now := uc.Clock.Now()
	changed, err := p.Revise(matchPrefix, requiredCapabilities, now)
	if err != nil {
		return nil, err
	}
	if !changed {
		return p, nil
	}
	err = atomically(ctx, uc.UnitOfWork, func(ctx context.Context) error {
		if err := uc.Repo.Save(ctx, p); err != nil {
			return err
		}
		return uc.Publisher.Publish(ctx, shared.ProcessPathUpdated{
			PathId:               p.ID(),
			MatchPrefix:          p.MatchPrefix(),
			Direct:               p.Direct(),
			RequiredCapabilities: p.RequiredCapabilities(),
			At:                   now,
		})
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}
