package override

import (
	"time"

	"github.com/google/uuid"

	"signalflow/internal/audit"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/store"
)

// Override is one manual takeover of an intersection's signal control.
type Override struct {
	ID             string    `json:"id"`
	IntersectionID string    `json:"intersection_id"`
	PlanID         string    `json:"plan_id"`
	PlanVersion    int       `json:"plan_version"`
	PhaseID        string    `json:"phase_id"`
	Color          string    `json:"color"`
	StartedAt      time.Time `json:"started_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Expired        bool      `json:"expired"`
	Cancelled      bool      `json:"cancelled"`
}

// PhaseDriver is the phase-machine surface used by the override flow.
type PhaseDriver interface {
	Current(intersectionID string) (phase.State, error)
	Force(intersectionID, phaseID, color string) (phase.State, error)
	FallbackToPlan(intersectionID, planID string, steps []phase.Step) (phase.State, error)
}

// PlanProvider is the plan-service surface used by the override flow.
type PlanProvider interface {
	CurrentForIntersection(intersectionID string) (plan.Plan, error)
	StepsFor(id string) ([]phase.Step, error)
}

// IntersectionGetter is the intersection-service surface used to validate
// takeover targets.
type IntersectionGetter interface {
	Get(id string) (intersection.Intersection, error)
}

// Service manages manual overrides: takeover, expiry fallback and cancel.
type Service struct {
	fs            *store.FileStore
	phases        PhaseDriver
	plans         PlanProvider
	intersections IntersectionGetter
	audit         audit.Recorder
	clock         func() time.Time
}

// NewService creates the override service.
func NewService(fs *store.FileStore, phases PhaseDriver, plans PlanProvider, intersections IntersectionGetter) *Service {
	return &Service{fs: fs, phases: phases, plans: plans, intersections: intersections, clock: time.Now}
}

// NewServiceWithAudit wires the audit sink into the override service.
func NewServiceWithAudit(fs *store.FileStore, phases PhaseDriver, plans PlanProvider, intersections IntersectionGetter, recorder audit.Recorder) *Service {
	s := NewService(fs, phases, plans, intersections)
	s.audit = recorder
	return s
}

func (s *Service) now() time.Time {
	return s.clock().UTC()
}

// Path resolves the persisted override record.
func (s *Service) Path(id string) string {
	return s.fs.Path("overrides", id)
}

// Get loads one override record.
func (s *Service) Get(id string) (Override, error) {
	var ov Override
	if err := s.fs.ReadJSON(s.Path(id), &ov); err != nil {
		return Override{}, err
	}
	return ov, nil
}

// List returns every override record.
func (s *Service) List() ([]Override, error) {
	ids, err := s.fs.List("overrides")
	if err != nil {
		return nil, err
	}
	out := make([]Override, 0, len(ids))
	for _, id := range ids {
		ov, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, ov)
	}
	return out, nil
}

// ActiveFor returns the non-expired override of an intersection, if any.
func (s *Service) ActiveFor(intersectionID string) (Override, bool, error) {
	records, err := s.List()
	if err != nil {
		return Override{}, false, err
	}
	for _, ov := range records {
		if ov.IntersectionID == intersectionID && !ov.Expired && !ov.Cancelled {
			return ov, true, nil
		}
	}
	return Override{}, false, nil
}

func (s *Service) save(ov Override) error {
	return s.fs.WriteJSON(s.Path(ov.ID), ov)
}

// String renders a compact summary of an override.
func (o Override) String() string {
	return o.IntersectionID + "@" + o.PhaseID + "/" + o.Color
}

// NewID allocates an override id; exported so tests and console code can keep
// ids stable.
func NewID() string {
	return uuid.NewString()
}
