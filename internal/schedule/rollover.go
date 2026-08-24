package schedule

import (
	"fmt"
	"time"

	"signalflow/internal/phase"
)

// RolloverResult describes one completed band rollover.
type RolloverResult struct {
	IntersectionID string
	FromBand       string
	ToBand         string
	PlanID         string
}

// RolloverDue checks every schedule and rolls over those whose active band no
// longer matches the current time.
func (s *Service) RolloverDue(now time.Time) ([]RolloverResult, error) {
	schedules, err := s.List()
	if err != nil {
		return nil, err
	}
	var results []RolloverResult
	for _, sched := range schedules {
		index, err := BandFor(sched.Bands, now)
		if err != nil {
			return nil, err
		}
		if index == sched.CurrentIndex {
			continue
		}
		result, err := s.Rollover(sched.IntersectionID, now)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Rollover moves one intersection onto the plan of the band covering now. The
// phase machine is asked to apply the rollover through its transition
// sequence, so the intersection never jumps straight to the new green.
func (s *Service) Rollover(intersectionID string, now time.Time) (RolloverResult, error) {
	sched, err := s.Get(intersectionID)
	if err != nil {
		return RolloverResult{}, err
	}
	index, err := BandFor(sched.Bands, now)
	if err != nil {
		return RolloverResult{}, err
	}
	if index == sched.CurrentIndex {
		return RolloverResult{}, nil
	}
	from := sched.Bands[sched.CurrentIndex].Name
	to := sched.Bands[index]
	sched.CurrentIndex = index
	sched.UpdatedAt = s.now()
	if err := s.fs.WriteJSON(s.Path(intersectionID), sched); err != nil {
		return RolloverResult{}, err
	}
	steps, err := s.plans.StepsFor(to.PlanID)
	if err != nil {
		return RolloverResult{}, err
	}
	st, err := s.phases.Current(intersectionID)
	if err != nil {
		return RolloverResult{}, err
	}
	st.PlanID = to.PlanID
	st.StepIndex = 0
	st.PhaseID = steps[0].PhaseID
	st.Color = phase.ColorGreen
	st.Forced = false
	if err := s.fs.WriteJSON(s.fs.Path("phase", intersectionID), st); err != nil {
		return RolloverResult{}, err
	}
	return RolloverResult{
		IntersectionID: intersectionID,
		FromBand:       from,
		ToBand:         to.Name,
		PlanID:         to.PlanID,
	}, nil
}

// BandFor picks the band whose StartMinute is the greatest value not after
// now's minute-of-day, wrapping to the first band after midnight.
func BandFor(bands []Band, now time.Time) (int, error) {
	if len(bands) == 0 {
		return 0, fmt.Errorf("no bands available")
	}
	minute := now.Hour()*60 + now.Minute()
	best := -1
	for i, band := range bands {
		if band.StartMinute <= minute {
			if best == -1 || band.StartMinute > bands[best].StartMinute {
				best = i
			}
		}
	}
	if best == -1 {
		best = 0
	}
	return best, nil
}
