package plan

// OffsetBaseline describes the coordination offsets contained in a plan's
// phase configuration.
type OffsetBaseline struct {
	PlanID  string
	Version int
	Offsets []int
}

// OffsetsFor derives the ordered coordination offsets from a plan. Each phase
// contributes its offset seconds in plan order; coordinated groups use this
// baseline when computing per-member offsets.
func (s *Service) OffsetsFor(id string) (OffsetBaseline, error) {
	p, err := s.Get(id)
	if err != nil {
		return OffsetBaseline{}, err
	}
	baseline := OffsetBaseline{PlanID: p.ID, Version: p.Version}
	for _, pc := range p.Phases {
		baseline.Offsets = append(baseline.Offsets, pc.OffsetSeconds)
	}
	return baseline, nil
}
