package memory

import (
	"context"
	"sync"

	"github.com/claudioed/process-path-management/internal/domain/cptschedule"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// CPTScheduleRepo is an in-memory implementation of
// ports.CPTScheduleRepo — the zero-DATABASE_URL runtime path, same
// convention as ProcessPathRepo.
type CPTScheduleRepo struct {
	mu        sync.RWMutex
	schedules map[shared.SiteId]*cptschedule.CPTSchedule
}

// NewCPTScheduleRepo constructs an empty CPTScheduleRepo.
func NewCPTScheduleRepo() *CPTScheduleRepo {
	return &CPTScheduleRepo{schedules: make(map[shared.SiteId]*cptschedule.CPTSchedule)}
}

func (r *CPTScheduleRepo) Save(_ context.Context, s *cptschedule.CPTSchedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules[s.SiteId()] = s
	return nil
}

func (r *CPTScheduleRepo) FindBySiteID(_ context.Context, siteId shared.SiteId) (*cptschedule.CPTSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.schedules[siteId], nil
}
