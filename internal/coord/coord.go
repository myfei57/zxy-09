package coord

import (
	"time"

	"github.com/google/uuid"

	"signalflow/internal/audit"
	"signalflow/internal/plan"
	"signalflow/internal/store"
)

// Group is one coordinated group: intersections that run the same timing plan
// and hold phase offsets so their greens form a wave.
type Group struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	PlanID      string         `json:"plan_id"`
	Members     []string       `json:"members"`
	Offsets     map[string]int `json:"offsets"`
	PlanVersion int            `json:"plan_version"`
	UpdatedAt   string         `json:"updated_at"`
}

// PlanLookup is implemented by the plan service.
type PlanLookup interface {
	Get(id string) (plan.Plan, error)
	OffsetsFor(id string) (plan.OffsetBaseline, error)
}

// Coordinator manages coordinated groups and their phase offsets.
type Coordinator struct {
	fs    *store.FileStore
	plans PlanLookup
	audit audit.Recorder
	clock func() time.Time
}

// NewCoordinator creates the coordination service.
func NewCoordinator(fs *store.FileStore, plans PlanLookup) *Coordinator {
	return &Coordinator{fs: fs, plans: plans, clock: time.Now}
}

// NewCoordinatorWithAudit wires the audit sink into the coordinator.
func NewCoordinatorWithAudit(fs *store.FileStore, plans PlanLookup, recorder audit.Recorder) *Coordinator {
	c := NewCoordinator(fs, plans)
	c.audit = recorder
	return c
}

func (c *Coordinator) now() string {
	return c.clock().UTC().Format(time.RFC3339)
}

// Path resolves the persisted record for a group.
func (c *Coordinator) Path(id string) string {
	return c.fs.Path("groups", id)
}

// Get loads one coordinated group.
func (c *Coordinator) Get(id string) (Group, error) {
	var g Group
	if err := c.fs.ReadJSON(c.Path(id), &g); err != nil {
		return Group{}, err
	}
	return g, nil
}

// List returns every coordinated group.
func (c *Coordinator) List() ([]Group, error) {
	ids, err := c.fs.List("groups")
	if err != nil {
		return nil, err
	}
	out := make([]Group, 0, len(ids))
	for _, id := range ids {
		g, err := c.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// Create registers a coordinated group bound to a timing plan.
func (c *Coordinator) Create(name, planID string) (Group, error) {
	baseline, err := c.plans.OffsetsFor(planID)
	if err != nil {
		return Group{}, err
	}
	g := Group{
		ID:          uuid.NewString(),
		Name:        name,
		PlanID:      planID,
		Members:     nil,
		Offsets:     map[string]int{},
		PlanVersion: baseline.Version,
		UpdatedAt:   c.now(),
	}
	if err := c.fs.WriteJSON(c.Path(g.ID), g); err != nil {
		return Group{}, err
	}
	return g, nil
}
