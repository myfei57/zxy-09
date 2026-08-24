package plan

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"signalflow/internal/audit"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/store"
)

// Status tracks the publication lifecycle of a timing plan.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusActive    Status = "active"
)

// PhaseConfig describes one phase of a timing plan together with its green
// hold, yellow clearance and coordination offset.
type PhaseConfig struct {
	PhaseID       string `json:"phase_id"`
	Order         int    `json:"order"`
	GreenSeconds  int    `json:"green_seconds"`
	YellowSeconds int    `json:"yellow_seconds"`
	OffsetSeconds int    `json:"offset_seconds"`
}

// Plan is the persisted timing plan record.
type Plan struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Status       Status        `json:"status"`
	Version      int           `json:"version"`
	Phases       []PhaseConfig `json:"phases"`
	VolumeFactor float64       `json:"volume_factor"`
	UpdatedAt    string        `json:"updated_at"`
}

// CoordRefresher is implemented by the coordination component so a plan
// switch can push the new plan generation into every coordinated group.
type CoordRefresher interface {
	ApplyPlanSwitch(planID string) error
}

// Service manages timing plans: creation, editing, durable commit, publish
// and activation. Publishing persists the phase configuration before the plan
// is allowed to switch, and activation refreshes coordination offsets only
// after the new generation is durable.
type Service struct {
	fs            *store.FileStore
	phaseConfigs  *phase.ConfigStore
	intersections *intersection.Service
	audit         audit.Recorder
	coord         CoordRefresher
	clock         func() time.Time
}

// NewService creates the plan service.
func NewService(fs *store.FileStore, phaseConfigs *phase.ConfigStore, intersections *intersection.Service, recorder audit.Recorder) *Service {
	return &Service{
		fs:            fs,
		phaseConfigs:  phaseConfigs,
		intersections: intersections,
		audit:         recorder,
		clock:         time.Now,
	}
}

// SetCoordRefresher wires the coordination callback used on activation.
func (s *Service) SetCoordRefresher(refresher CoordRefresher) {
	s.coord = refresher
}

func (s *Service) now() string {
	return s.clock().UTC().Format(time.RFC3339)
}

// Path resolves the persisted record for a plan.
func (s *Service) Path(id string) string {
	return s.fs.Path("plans", id)
}

// StagingPath resolves the in-progress draft of a plan edit.
func (s *Service) StagingPath(id string) string {
	return s.fs.Path("planstaging", id)
}

// Get loads one timing plan.
func (s *Service) Get(id string) (Plan, error) {
	var p Plan
	if err := s.fs.ReadJSON(s.Path(id), &p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

// List returns every timing plan sorted by id.
func (s *Service) List() ([]Plan, error) {
	ids, err := s.fs.List("plans")
	if err != nil {
		return nil, err
	}
	out := make([]Plan, 0, len(ids))
	for _, id := range ids {
		p, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Create registers a new draft plan with the given phases.
func (s *Service) Create(name string, phases []PhaseConfig) (Plan, error) {
	if err := phase.Validate(stepsOf(phases)); err != nil {
		return Plan{}, err
	}
	p := Plan{
		ID:           uuid.NewString(),
		Name:         name,
		Status:       StatusDraft,
		Version:      1,
		Phases:       phases,
		VolumeFactor: 1.0,
		UpdatedAt:    s.now(),
	}
	if err := s.fs.WriteJSON(s.Path(p.ID), p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

// StepsFor converts a plan into the phase machine steps it should execute.
func (s *Service) StepsFor(id string) ([]phase.Step, error) {
	p, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return stepsOf(p.Phases), nil
}

func stepsOf(phases []PhaseConfig) []phase.Step {
	steps := make([]phase.Step, 0, len(phases))
	for _, pc := range phases {
		steps = append(steps, phase.Step{
			PhaseID:       pc.PhaseID,
			Order:         pc.Order,
			GreenSeconds:  pc.GreenSeconds,
			YellowSeconds: pc.YellowSeconds,
		})
	}
	return steps
}

// String returns a stable description of a plan for audit entries.
func (p Plan) String() string {
	return fmt.Sprintf("%s(v%d/%s)", p.Name, p.Version, p.Status)
}
