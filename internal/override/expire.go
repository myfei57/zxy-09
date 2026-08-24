package override

import "time"

// ExpireDue expires every override whose deadline has passed, returning the
// overrides that transitioned.
func (s *Service) ExpireDue(now time.Time) ([]Override, error) {
	records, err := s.List()
	if err != nil {
		return nil, err
	}
	var expired []Override
	for _, ov := range records {
		if ov.Expired || ov.Cancelled {
			continue
		}
		if !now.Before(ov.ExpiresAt) {
			done, err := s.Expire(ov.ID)
			if err != nil {
				return expired, err
			}
			expired = append(expired, done)
		}
	}
	return expired, nil
}

// Expire ends one override and hands control back to the scheduled plan: the
// phase machine falls back to the intersection's current plan instead of
// keeping the manually forced phase.
func (s *Service) Expire(id string) (Override, error) {
	ov, err := s.Get(id)
	if err != nil {
		return Override{}, err
	}
	if ov.Expired || ov.Cancelled {
		return Override{}, ErrNoOverride
	}
	// Resolve the live plan at expiry time and hand control back to it, so a
	// plan switched during the override is honored rather than replaying the
	// takeover snapshot. FallbackToPlan clears the forced marker and restarts
	// the machine at the plan's first phase in red, letting the control loop
	// resume normal advancement.
	current, err := s.plans.CurrentForIntersection(ov.IntersectionID)
	if err != nil {
		return Override{}, err
	}
	steps, err := s.plans.StepsFor(current.ID)
	if err != nil {
		return Override{}, err
	}
	if _, err := s.phases.FallbackToPlan(ov.IntersectionID, current.ID, steps); err != nil {
		return Override{}, err
	}
	ov.Expired = true
	if err := s.save(ov); err != nil {
		return Override{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "override.expire", "intersection", ov.IntersectionID, ov.String())
	}
	return ov, nil
}
