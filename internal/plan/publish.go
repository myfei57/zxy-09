package plan

import (
	"errors"

	"signalflow/internal/phase"
)

// Publish moves a draft plan into the published state. The phase
// configuration is written durably before the plan status changes, so a
// failed phase write can never leave an intersection running a published plan
// whose phase sequence was never stored.
func (s *Service) Publish(id string) (Plan, error) {
	p, err := s.Get(id)
	if err != nil {
		return Plan{}, err
	}
	if p.Status != StatusDraft {
		return Plan{}, errors.New("only a draft plan can be published")
	}
	snap := phase.Snapshot{
		PlanID:  p.ID,
		Version: p.Version,
		Steps:   stepsOf(p.Phases),
	}
	if err := s.phaseConfigs.Save(snap); err != nil {
		return Plan{}, err
	}
	p.Status = StatusPublished
	p.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(p.ID), p); err != nil {
		return Plan{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "plan.publish", "plan", p.ID, p.String())
	}
	return p, nil
}
