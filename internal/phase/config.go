package phase

import (
	"fmt"

	"signalflow/internal/store"
)

// Snapshot is the durably stored phase configuration of a timing plan. It is
// written by the plan publisher before a plan is allowed to become active, so
// the phase machine can always reconstruct the exact sequence it must run.
type Snapshot struct {
	PlanID  string `json:"plan_id"`
	Version int    `json:"version"`
	Steps   []Step `json:"steps"`
}

// ConfigStore persists and loads phase configuration snapshots.
type ConfigStore struct {
	fs *store.FileStore
}

// NewConfigStore creates the phase configuration store.
func NewConfigStore(fs *store.FileStore) *ConfigStore {
	return &ConfigStore{fs: fs}
}

// Path resolves the snapshot file for a plan.
func (c *ConfigStore) Path(planID string) string {
	return c.fs.Path("phaseconfigs", planID)
}

// Save durably writes the phase configuration snapshot for a plan.
func (c *ConfigStore) Save(snap Snapshot) error {
	if len(snap.Steps) == 0 {
		return fmt.Errorf("phase config for plan %s has no steps", snap.PlanID)
	}
	return c.fs.WriteJSON(c.Path(snap.PlanID), snap)
}

// Load reads the phase configuration snapshot for a plan.
func (c *ConfigStore) Load(planID string) (Snapshot, error) {
	var snap Snapshot
	if err := c.fs.ReadJSON(c.Path(planID), &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}
