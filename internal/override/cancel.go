package override

// Cancel ends a manual override immediately and returns control to the
// intersection's current plan. The live plan is resolved at cancel time, so
// a plan switched during the override is honored instead of replaying the
// snapshot captured at takeover.
func (s *Service) Cancel(id string) (Override, error) {
	ov, err := s.Get(id)
	if err != nil {
		return Override{}, err
	}
	if ov.Expired || ov.Cancelled {
		return Override{}, ErrNoOverride
	}
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
	ov.Cancelled = true
	if err := s.save(ov); err != nil {
		return Override{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "override.cancel", "intersection", ov.IntersectionID, ov.String())
	}
	return ov, nil
}
