package detector

import (
	"time"

	"signalflow/internal/plan"
	"signalflow/internal/store"
)

// Sample is one raw detector reading: the number of vehicles observed on an
// approach during a measurement window.
type Sample struct {
	ID             string    `json:"id"`
	IntersectionID string    `json:"intersection_id"`
	PhaseID        string    `json:"phase_id"`
	Vehicles       int       `json:"vehicles"`
	At             time.Time `json:"at"`
}

// Service receives detector samples and applies the measured volumes to the
// committed timing plan of the intersection.
type Service struct {
	fs    *store.FileStore
	plans PlanEditor
	clock func() time.Time
}

// PlanEditor is the subset of the plan service used by the detector path.
type PlanEditor interface {
	CurrentForIntersection(intersectionID string) (plan.Plan, error)
	ApplyVolumes(id string, volumes map[string]int) (plan.Plan, error)
	DraftOf(id string) (plan.Plan, error)
}

// NewService creates the detector service.
func NewService(fs *store.FileStore, plans PlanEditor) *Service {
	return &Service{fs: fs, plans: plans, clock: time.Now}
}

func (s *Service) now() time.Time {
	return s.clock().UTC()
}

// Path resolves the stored sample record.
func (s *Service) Path(id string) string {
	return s.fs.Path("detector", id)
}
