package fault

import "errors"

// Clear confirms a fault is resolved: the cleared record is written, the
// control plane returns to normal, and the live signal output is re-armed so
// the field lamps leave the degraded pattern in step with the page state.
func (s *Service) Clear(intersectionID string) error {
	rec, active, err := s.activeFor(intersectionID)
	if err != nil {
		return err
	}
	if !active {
		return errors.New("intersection has no active fault to clear")
	}
	if err := s.markCleared(rec); err != nil {
		return err
	}
	if _, err := s.intersections.RestoreNormal(intersectionID); err != nil {
		return err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "fault.clear", "intersection", intersectionID, rec.String())
	}
	return nil
}
