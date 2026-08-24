package plan

import (
	"errors"
	"math"
)

// BeginEdit stages a new draft of a plan edit. The staged draft is not
// visible to detectors or the phase machine until Commit succeeds.
func (s *Service) BeginEdit(id string, phases []PhaseConfig) (Plan, error) {
	p, err := s.Get(id)
	if err != nil {
		return Plan{}, err
	}
	if p.Status == StatusActive {
		return Plan{}, errors.New("an active plan cannot be edited")
	}
	draft := p
	draft.Phases = phases
	if err := s.fs.WriteJSON(s.StagingPath(id), draft); err != nil {
		return Plan{}, err
	}
	return draft, nil
}

// Commit durably applies a staged plan edit and bumps the plan version. A
// failed commit leaves the running plan untouched.
func (s *Service) Commit(id string) (Plan, error) {
	var draft Plan
	if err := s.fs.ReadJSON(s.StagingPath(id), &draft); err != nil {
		return Plan{}, err
	}
	draft.Version++
	draft.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), draft); err != nil {
		return Plan{}, err
	}
	if err := s.fs.Remove(s.StagingPath(id)); err != nil {
		return Plan{}, err
	}
	if s.audit != nil {
		_, _ = s.audit.Record("console", "plan.commit", "plan", id, draft.String())
	}
	return draft, nil
}

// ApplyVolumes folds measured detector volumes into a committed plan. The
// update goes through the same durable record as the plan itself and is only
// accepted for committed plans, so a rejected edit can never be polluted by
// detector data.
func (s *Service) ApplyVolumes(id string, volumes map[string]int) (Plan, error) {
	p, err := s.Get(id)
	if err != nil {
		return Plan{}, err
	}
	total := 0
	for _, v := range volumes {
		total += v
	}
	if total > 0 {
		ratio := 1.0 + math.Min(float64(total)/1000.0, 0.3)
		p.VolumeFactor = math.Round(ratio*100) / 100
	} else {
		p.VolumeFactor = 1.0
	}
	p.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(id), p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

// DraftOf reads the current staging draft for a plan, returning ErrNotFound
// when no edit is in progress.
func (s *Service) DraftOf(id string) (Plan, error) {
	var draft Plan
	if err := s.fs.ReadJSON(s.StagingPath(id), &draft); err != nil {
		return Plan{}, err
	}
	return draft, nil
}
