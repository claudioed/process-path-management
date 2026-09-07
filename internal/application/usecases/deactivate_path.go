package usecases

import (
	"context"

	"github.com/claudioed/process-path-management/internal/application/ports"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// DeactivatePath retires a path and publishes ProcessPathDeactivated.
// Idempotent end-to-end: deactivating an already-deactivated path
// succeeds without republishing the event (mirrors
// ProcessPath.Deactivate's own idempotency, extended here so a retried
// HTTP call or redelivered command never double-publishes).
type DeactivatePath struct {
	Repo      ports.ProcessPathRepo
	Publisher ports.EventPublisher
	Clock     ports.Clock
	// UnitOfWork brackets Save + Publish atomically (ADR 0003); nil means
	// no transactional backing (see DefinePath).
	UnitOfWork ports.UnitOfWork
}

func (uc *DeactivatePath) Execute(ctx context.Context, id shared.PathId) error {
	p, err := uc.Repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPathNotFound
	}
	if !p.IsActive() {
		return nil
	}

	now := uc.Clock.Now()
	p.Deactivate(now)
	return atomically(ctx, uc.UnitOfWork, func(ctx context.Context) error {
		if err := uc.Repo.Save(ctx, p); err != nil {
			return err
		}
		return uc.Publisher.Publish(ctx, shared.ProcessPathDeactivated{
			PathId: p.ID(),
			At:     now,
		})
	})
}
