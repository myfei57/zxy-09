package intersection

import (
	"errors"
	"fmt"
)

// Activate moves an unconfigured intersection into the active lifecycle state
// and attaches the timing plan it will execute.
func (s *Service) Activate(id, planID string) (Intersection, error) {
	in, err := s.Get(id)
	if err != nil {
		return Intersection{}, err
	}
	if in.Status != StatusUnconfigured {
		return Intersection{}, fmt.Errorf("intersection %s is not unconfigured", id)
	}
	if planID == "" {
		return Intersection{}, errors.New("plan is required to activate an intersection")
	}
	in.Status = StatusActive
	in.PlanID = planID
	in.ControlMode = ModeNormal
	in.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}

// SetPlan attaches a different timing plan to an active intersection. The
// caller is responsible for restarting the phase machine on the new plan.
func (s *Service) SetPlan(id, planID string) (Intersection, error) {
	in, err := s.Get(id)
	if err != nil {
		return Intersection{}, err
	}
	if in.Status != StatusActive && in.Status != StatusDegraded {
		return Intersection{}, fmt.Errorf("intersection %s cannot change plan in status %s", id, in.Status)
	}
	if planID == "" {
		return Intersection{}, errors.New("plan is required")
	}
	in.PlanID = planID
	in.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}
