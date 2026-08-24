package plan

import (
	"errors"

	"signalflow/internal/phase"
)

// SwitchActive activates a published plan as a new generation carrying the
// given phase configuration. The phase configuration snapshot is written
// durably first, then the plan record; only after the new generation is on
// disk are the coordination offsets refreshed, so every group recomputes
// against the new offsets.
func (s *Service) SwitchActive(id string, phases []PhaseConfig) (Plan, error) {
	p, err := s.Get(id)
	if err != nil {
		return Plan{}, err
	}
	if p.Status != StatusPublished && p.Status != StatusActive {
		return Plan{}, errors.New("only a published plan can be activated")
	}
	snap := phase.Snapshot{
		PlanID:  p.ID,
		Version: p.Version + 1,
		Steps:   stepsOf(phases),
	}
	if err := s.phaseConfigs.Save(snap); err != nil {
		return Plan{}, err
	}
	p.Phases = phases
	p.Status = StatusActive
	p.Version++
	p.UpdatedAt = s.now()
	if s.coord != nil {
		if err := s.coord.ApplyPlanSwitch(p.ID); err != nil {
			return Plan{}, err
		}
	}
	if err := s.fs.WriteJSON(s.Path(p.ID), p); err != nil {
		return Plan{}, err
	}
	if _, err := s.phaseConfigs.Load(p.ID); err != nil {
		return Plan{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "plan.switch", "plan", p.ID, p.String())
	}
	return p, nil
}
