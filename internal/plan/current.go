package plan

// CurrentForIntersection resolves the timing plan currently assigned to an
// intersection through its persisted plan reference.
func (s *Service) CurrentForIntersection(intersectionID string) (Plan, error) {
	in, err := s.intersections.Get(intersectionID)
	if err != nil {
		return Plan{}, err
	}
	return s.Get(in.PlanID)
}
