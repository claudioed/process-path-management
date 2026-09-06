// Package usecases implements the application layer: one struct per use
// case, orchestrating the processpath domain package against the ports
// interfaces. No adapter/framework type ever appears here.
package usecases

import (
	"context"

	"github.com/claudioed/process-path-management/internal/application/ports"
	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// DefinePath defines a brand-new process path and publishes
// ProcessPathCreated. Rejects with ErrPathAlreadyExists if the PathId is
// already taken (active or deactivated) — see that sentinel's doc comment
// for why this is never a silent overwrite.
type DefinePath struct {
	Repo      ports.ProcessPathRepo
	Publisher ports.EventPublisher
	Clock     ports.Clock
	// Metrics is optional (fleet convention: a nil value means "not
	// instrumented", see ports.PathMetrics's doc comment).
	Metrics ports.PathMetrics
}

func (uc *DefinePath) Execute(ctx context.Context, id shared.PathId, matchPrefix string, direct bool, requiredCapabilities []shared.Capability) (*processpath.ProcessPath, error) {
	existing, err := uc.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		uc.recordRejected(ctx)
		return nil, ErrPathAlreadyExists
	}

	now := uc.Clock.Now()
	p, err := processpath.Define(id, matchPrefix, direct, requiredCapabilities, now)
	if err != nil {
		uc.recordRejected(ctx)
		return nil, err
	}
	if err := uc.Repo.Save(ctx, p); err != nil {
		return nil, err
	}
	if err := uc.Publisher.Publish(ctx, shared.ProcessPathCreated{
		PathId:               p.ID(),
		MatchPrefix:          p.MatchPrefix(),
		Direct:               p.Direct(),
		RequiredCapabilities: p.RequiredCapabilities(),
		At:                   now,
	}); err != nil {
		return nil, err
	}
	if uc.Metrics != nil {
		uc.Metrics.PathDefinitionAccepted(ctx)
	}
	return p, nil
}

func (uc *DefinePath) recordRejected(ctx context.Context) {
	if uc.Metrics != nil {
		uc.Metrics.PathDefinitionRejected(ctx)
	}
}
