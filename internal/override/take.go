package override

import (
	"errors"
	"time"
)

// Take starts a manual override of an intersection. The current plan is
// recorded as a snapshot, the phase machine is forced to the requested phase,
// and the override expires after the given duration unless cancelled first.
func (s *Service) Take(intersectionID, phaseID, color string, duration time.Duration) (Override, error) {
	if _, err := s.intersections.Get(intersectionID); err != nil {
		return Override{}, err
	}
	current, err := s.plans.CurrentForIntersection(intersectionID)
	if err != nil {
		return Override{}, err
	}
	now := s.now()
	ov := Override{
		ID:             NewID(),
		IntersectionID: intersectionID,
		PlanID:         current.ID,
		PlanVersion:    current.Version,
		PhaseID:        phaseID,
		Color:          color,
		StartedAt:      now,
		ExpiresAt:      now.Add(duration),
	}
	if err := s.save(ov); err != nil {
		return Override{}, err
	}
	if _, err := s.phases.Force(intersectionID, phaseID, color); err != nil {
		return Override{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "override.take", "intersection", intersectionID, ov.String())
	}
	return ov, nil
}

// ErrNoOverride guards callers that try to cancel or expire a takeover that
// does not exist.
var ErrNoOverride = errors.New("override not found")
