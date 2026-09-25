package usecases

import (
	"context"

	"github.com/claudioed/process-path-management/internal/application/ports"
	"github.com/claudioed/process-path-management/internal/domain/cptschedule"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// DefineCPTSchedule defines or wholesale-revises a site's CPTSchedule and
// publishes CPTScheduleChanged (ADR 0010). There is no separate
// "RevisePath"-style use case: a CPTSchedule has no partial-update
// semantics — every PUT replaces the schedule's timezone/cutoffs in full,
// matching the aggregate's own Define/Revise (Revise mirrors Define's
// signature exactly).
//
// The one cross-aggregate invariant ADR 0010 calls out — every cutoff's
// eligiblePathIds must reference an Active ProcessPath in this service's
// own store — is enforced HERE, via the injected ProcessPathRepo, not in
// the cptschedule domain package: it is a use-case-level check against a
// sibling aggregate's repo, never a foreign key.
type DefineCPTSchedule struct {
	Repo            ports.CPTScheduleRepo
	ProcessPathRepo ports.ProcessPathRepo
	Publisher       ports.EventPublisher
	Clock           ports.Clock
	// UnitOfWork brackets Save + Publish atomically (ADR 0003); nil means
	// no transactional backing (see DefinePath).
	UnitOfWork ports.UnitOfWork
}

func (uc *DefineCPTSchedule) Execute(ctx context.Context, siteId shared.SiteId, timezone string, cutoffs []cptschedule.Cutoff) (*cptschedule.CPTSchedule, error) {
	existing, err := uc.Repo.FindBySiteID(ctx, siteId)
	if err != nil {
		return nil, err
	}

	// The cross-aggregate eligiblePathIds check: every referenced PathId
	// must resolve to an Active ProcessPath in this service's own store.
	// Checked against the union across all cutoffs up front, before any
	// domain construction, so a caller gets one clear error rather than a
	// partial write.
	if err := uc.validateEligiblePathIds(ctx, cutoffs); err != nil {
		return nil, err
	}

	now := uc.Clock.Now()

	var schedule *cptschedule.CPTSchedule
	if existing == nil {
		schedule, err = cptschedule.Define(siteId, timezone, cutoffs, now)
		if err != nil {
			return nil, err
		}
	} else {
		schedule = existing
		changed, err := schedule.Revise(timezone, cutoffs, now)
		if err != nil {
			return nil, err
		}
		if !changed {
			return schedule, nil
		}
	}

	err = atomically(ctx, uc.UnitOfWork, func(ctx context.Context) error {
		if err := uc.Repo.Save(ctx, schedule); err != nil {
			return err
		}
		return uc.Publisher.Publish(ctx, cptschedule.ToSnapshot(schedule, now))
	})
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

// validateEligiblePathIds enforces ADR 0010's one cross-aggregate
// invariant: every eligiblePathIds entry across every cutoff must
// reference an Active ProcessPath. Deduplicates lookups so a PathId
// referenced by several cutoffs is only looked up once.
func (uc *DefineCPTSchedule) validateEligiblePathIds(ctx context.Context, cutoffs []cptschedule.Cutoff) error {
	checked := make(map[shared.PathId]bool)
	for _, c := range cutoffs {
		for _, id := range c.EligiblePathIds() {
			if checked[id] {
				continue
			}
			checked[id] = true
			p, err := uc.ProcessPathRepo.FindByID(ctx, id)
			if err != nil {
				return err
			}
			if p == nil || !p.IsActive() {
				return ErrIneligiblePathId
			}
		}
	}
	return nil
}

// GetCPTSchedule returns one site's schedule.
type GetCPTSchedule struct {
	Repo ports.CPTScheduleRepo
}

func (uc *GetCPTSchedule) Execute(ctx context.Context, siteId shared.SiteId) (*cptschedule.CPTSchedule, error) {
	s, err := uc.Repo.FindBySiteID(ctx, siteId)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrCPTScheduleNotFound
	}
	return s, nil
}
