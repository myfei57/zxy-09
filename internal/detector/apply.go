package detector

import (
	"signalflow/internal/plan"
)

// ApplyVolumes folds the measured volumes of an intersection into its
// committed timing plan. The data path resolves the committed plan and routes
// the update through the plan's commit gate; a plan edit that is still in
// progress is never mutated.
func (s *Service) ApplyVolumes(intersectionID string, volumes map[string]int) (plan.Plan, error) {
	current, err := s.plans.CurrentForIntersection(intersectionID)
	if err != nil {
		return plan.Plan{}, err
	}
	return s.plans.ApplyVolumes(current.ID, volumes)
}
