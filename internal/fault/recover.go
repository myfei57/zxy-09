package fault

import "errors"

// Recover restores an intersection to normal control after a fault clears.
// The fault-cleared record must be durable before the intersection leaves the
// degraded state, otherwise the next occurrence of the same fault would be
// deduplicated against the still-active record and missed.
func (s *Service) Recover(intersectionID string) error {
	rec, active, err := s.activeFor(intersectionID)
	if err != nil {
		return err
	}
	if !active {
		return errors.New("intersection has no active fault to recover")
	}
	// Persist the cleared record before leaving the degraded state. If the
	// restore happened first and this write failed, the intersection would be
	// back under normal control while the record still read StatusFaulted, so
	// the next occurrence of the same fault would be deduplicated against the
	// still-active record and silently missed.
	if err := s.markCleared(rec); err != nil {
		return err
	}
	if _, err := s.intersections.RestoreNormal(intersectionID); err != nil {
		return err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "fault.recover", "intersection", intersectionID, rec.String())
	}
	return nil
}
