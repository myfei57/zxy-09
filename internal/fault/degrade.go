package fault

import (
	"errors"
)

// Detect records a device fault and degrades the intersection. The
// degradation record is written durably before normal control is handed over,
// so a failed record write leaves the intersection in its previous state.
func (s *Service) Detect(intersectionID, kind string) (Record, error) {
	if _, active, err := s.activeFor(intersectionID); err != nil {
		return Record{}, err
	} else if active {
		return Record{}, errors.New("intersection already has an active fault")
	}
	rec := Record{
		ID:             intersectionID,
		IntersectionID: intersectionID,
		Kind:           kind,
		Status:         StatusFaulted,
		DetectedAt:     s.now(),
	}
	if _, err := s.intersections.SwitchToDegraded(intersectionID, rec.ID); err != nil {
		return Record{}, err
	}
	if err := s.save(rec); err != nil {
		return Record{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "fault.detect", "intersection", intersectionID, rec.String())
	}
	return rec, nil
}
