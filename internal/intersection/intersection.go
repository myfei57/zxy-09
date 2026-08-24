package intersection

import (
	"time"

	"github.com/google/uuid"

	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// Status models the lifecycle of an intersection in the control plane.
type Status string

const (
	StatusUnconfigured Status = "unconfigured"
	StatusActive       Status = "active"
	StatusDegraded     Status = "degraded"
	StatusRecovering   Status = "recovering"
)

// ControlMode records who currently drives the signal outputs.
type ControlMode string

const (
	ModeNone     ControlMode = ""
	ModeNormal   ControlMode = "normal"
	ModeDegraded ControlMode = "degraded"
	ModeManual   ControlMode = "manual"
)

// Intersection is the persisted control-plane record for one signalized
// intersection.
type Intersection struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Status      Status      `json:"status"`
	PlanID      string      `json:"plan_id"`
	ControlMode ControlMode `json:"control_mode"`
	FaultID     string      `json:"fault_id"`
	UpdatedAt   string      `json:"updated_at"`
}

// Service manages intersection registration and control-plane state
// transitions. State changes are persisted before they are exposed to the
// rest of the system.
type Service struct {
	fs     *store.FileStore
	driver *signal.Driver
	clock  func() time.Time
}

// NewService creates the intersection service over the shared store and the
// live signal driver.
func NewService(fs *store.FileStore, driver *signal.Driver) *Service {
	return &Service{fs: fs, driver: driver, clock: time.Now}
}

func (s *Service) now() string {
	return s.clock().UTC().Format(time.RFC3339)
}

// Path resolves the persisted record for an intersection.
func (s *Service) Path(id string) string {
	return s.fs.Path("intersections", id)
}

// Get loads one intersection record.
func (s *Service) Get(id string) (Intersection, error) {
	var in Intersection
	if err := s.fs.ReadJSON(s.Path(id), &in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}

// List returns every registered intersection sorted by id.
func (s *Service) List() ([]Intersection, error) {
	ids, err := s.fs.List("intersections")
	if err != nil {
		return nil, err
	}
	out := make([]Intersection, 0, len(ids))
	for _, id := range ids {
		in, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, nil
}

// Register creates a new intersection in the unconfigured state.
func (s *Service) Register(name string) (Intersection, error) {
	in := Intersection{
		ID:          uuid.NewString(),
		Name:        name,
		Status:      StatusUnconfigured,
		ControlMode: ModeNone,
		UpdatedAt:   s.now(),
	}
	if err := s.fs.WriteJSON(s.Path(in.ID), in); err != nil {
		return Intersection{}, err
	}
	return in, nil
}
