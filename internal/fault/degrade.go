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
	// Persist the degradation record before touching the control plane. If the
	// record write fails the intersection stays under normal control, so a
	// failed write can never leave it dropped between modes with no durable
	// record of why.
	if err := s.save(rec); err != nil {
		return Record{}, err
	}
	if _, err := s.intersections.SwitchToDegraded(intersectionID, rec.ID); err != nil {
		// The control-plane transition failed after the record was made
		// durable. Roll the record back so the next detection is not
		// deduplicated against a fault the intersection never entered.
		_ = s.fs.Remove(s.Path(rec.IntersectionID))
		return Record{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "fault.detect", "intersection", intersectionID, rec.String())
	}
	return rec, nil
}
