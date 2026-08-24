package intersection

import "fmt"

// ReturnNormal switches an intersection back to normal control while keeping
// any attached fault reference intact.
func (s *Service) ReturnNormal(id string) (Intersection, error) {
	in, err := s.Get(id)
	if err != nil {
		return Intersection{}, err
	}
	if in.Status != StatusDegraded && in.Status != StatusRecovering {
		return Intersection{}, fmt.Errorf("intersection %s cannot return to normal from status %s", id, in.Status)
	}
	in.Status = StatusActive
	in.ControlMode = ModeNormal
	in.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}

// RestoreNormal completes the recovery handover: it returns the intersection
// to normal control and detaches the fault reference in one persisted update.
func (s *Service) RestoreNormal(id string) (Intersection, error) {
	in, err := s.ReturnNormal(id)
	if err != nil {
		return Intersection{}, err
	}
	in.FaultID = ""
	in.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}
