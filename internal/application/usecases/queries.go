package usecases

import (
	"context"

	"github.com/claudioed/process-path-management/internal/application/ports"
	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// GetPath returns one path by id, active or deactivated.
type GetPath struct {
	Repo ports.ProcessPathRepo
}

func (uc *GetPath) Execute(ctx context.Context, id shared.PathId) (*processpath.ProcessPath, error) {
	p, err := uc.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPathNotFound
	}
	return p, nil
}

// ListPaths returns paths for the operator SPA and consumer read models.
// activeOnly=true is the default view (and what every consuming service's
// own cache hydration uses); activeOnly=false is the SPA's audit view.
type ListPaths struct {
	Repo ports.ProcessPathRepo
}

func (uc *ListPaths) Execute(ctx context.Context, activeOnly bool) ([]*processpath.ProcessPath, error) {
	if activeOnly {
		return uc.Repo.ListActive(ctx)
	}
	return uc.Repo.ListAll(ctx)
}
