package intersection

import (
	"signalflow/internal/signal"
)

// ReArmNormalOutput resets the live output after a fault clears so the field
// lamps leave the degraded pattern and follow the plan again. The next
// control tick writes the actual scheduled phase.
func (s *Service) ReArmNormalOutput(id string) (signal.Output, error) {
	out := signal.Output{PhaseID: "", Color: "red", Manual: false}
	if err := s.driver.Write(id, out); err != nil {
		return signal.Output{}, err
	}
	return out, nil
}
