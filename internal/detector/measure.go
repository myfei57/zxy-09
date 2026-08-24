package detector

import (
	"github.com/google/uuid"
)

// RecordSample persists one raw detector reading.
func (s *Service) RecordSample(sample Sample) (Sample, error) {
	if sample.ID == "" {
		sample.ID = uuid.NewString()
	}
	if sample.At.IsZero() {
		sample.At = s.now()
	}
	if err := s.fs.WriteJSON(s.Path(sample.ID), sample); err != nil {
		return Sample{}, err
	}
	return sample, nil
}
