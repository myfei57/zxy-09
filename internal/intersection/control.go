package intersection

import (
	"fmt"
)

// ReleaseForDegrade drops normal control from an intersection, leaving no
// active control mode. It is an internal step of the degraded handover and
// must never be called before the degradation record is durable.
func (s *Service) ReleaseForDegrade(id string) (Intersection, error) {
	in, err := s.Get(id)
	if err != nil {
		return Intersection{}, err
	}
	if in.ControlMode != ModeNormal {
		return Intersection{}, fmt.Errorf("intersection %s is not under normal control", id)
	}
	in.ControlMode = ModeNone
	in.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}

// EnterDegraded marks an intersection as degraded and switches its control
// mode to the degraded driver.
func (s *Service) EnterDegraded(id, faultID string) (Intersection, error) {
	in, err := s.Get(id)
	if err != nil {
		return Intersection{}, err
	}
	in.Status = StatusDegraded
	in.ControlMode = ModeDegraded
	in.FaultID = faultID
	in.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}

// SwitchToDegraded performs the complete normal-to-degraded handover. It
// releases normal control and enters degraded mode as one transition so the
// intersection never observes a window without a valid control mode.
func (s *Service) SwitchToDegraded(id, faultID string) (Intersection, error) {
	if _, err := s.ReleaseForDegrade(id); err != nil {
		return Intersection{}, err
	}
	return s.EnterDegraded(id, faultID)
}
