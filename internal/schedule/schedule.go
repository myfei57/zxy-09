package schedule

import (
	"errors"
	"sort"
	"time"

	"signalflow/internal/phase"
	"signalflow/internal/store"
)

// Band is one time band of a daily schedule. StartMinute is minutes since
// midnight; PlanID names the timing plan active during the band.
type Band struct {
	Name        string `json:"name"`
	StartMinute int    `json:"start_minute"`
	PlanID      string `json:"plan_id"`
}

// Schedule is the persisted daily plan schedule of one intersection.
type Schedule struct {
	ID             string `json:"id"`
	IntersectionID string `json:"intersection_id"`
	Bands          []Band `json:"bands"`
	CurrentIndex   int    `json:"current_index"`
	UpdatedAt      string `json:"updated_at"`
}

// PlanGetter is implemented by the plan service and lets the scheduler
// resolve the phase steps of a band's plan.
type PlanGetter interface {
	StepsFor(id string) ([]phase.Step, error)
}

// PhaseMachine is the subset of the phase machine used by the scheduler.
type PhaseMachine interface {
	Current(intersectionID string) (phase.State, error)
	ApplyRollover(st phase.State, steps []phase.Step) (phase.State, error)
}

// Service manages daily schedules and rolls the active plan over when the
// current time enters a new band.
type Service struct {
	fs     *store.FileStore
	phases PhaseMachine
	plans  PlanGetter
	clock  func() time.Time
}

// NewService creates the schedule service.
func NewService(fs *store.FileStore, phases PhaseMachine, plans PlanGetter) *Service {
	return &Service{fs: fs, phases: phases, plans: plans, clock: time.Now}
}

// NewServiceWithClock is the test-facing constructor that pins the clock.
func NewServiceWithClock(fs *store.FileStore, phases PhaseMachine, plans PlanGetter, clock func() time.Time) *Service {
	s := NewService(fs, phases, plans)
	s.clock = clock
	return s
}

func (s *Service) now() string {
	return s.clock().UTC().Format(time.RFC3339)
}

// Path resolves the schedule record for an intersection.
func (s *Service) Path(intersectionID string) string {
	return s.fs.Path("schedules", intersectionID)
}

// Assign installs a band list for an intersection and starts on the first
// band.
func (s *Service) Assign(intersectionID string, bands []Band) (Schedule, error) {
	if len(bands) == 0 {
		return Schedule{}, errors.New("a schedule needs at least one band")
	}
	normalized := make([]Band, len(bands))
	copy(normalized, bands)
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].StartMinute < normalized[j].StartMinute
	})
	sched := Schedule{
		ID:             intersectionID,
		IntersectionID: intersectionID,
		Bands:          normalized,
		CurrentIndex:   0,
		UpdatedAt:      s.now(),
	}
	if err := s.fs.WriteJSON(s.Path(intersectionID), sched); err != nil {
		return Schedule{}, err
	}
	return sched, nil
}

// Get loads the schedule of an intersection.
func (s *Service) Get(intersectionID string) (Schedule, error) {
	var sched Schedule
	if err := s.fs.ReadJSON(s.Path(intersectionID), &sched); err != nil {
		return Schedule{}, err
	}
	return sched, nil
}

// List returns every schedule.
func (s *Service) List() ([]Schedule, error) {
	ids, err := s.fs.List("schedules")
	if err != nil {
		return nil, err
	}
	out := make([]Schedule, 0, len(ids))
	for _, id := range ids {
		sched, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, sched)
	}
	return out, nil
}
