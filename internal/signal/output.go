package signal

import (
	"signalflow/internal/store"
)

// Output describes the live signal output for one intersection: which phase
// is active, which lamp color is showing, and whether the output is being
// held by a manual override.
type Output struct {
	PhaseID string `json:"phase_id"`
	Color   string `json:"color"`
	Manual  bool   `json:"manual"`
}

// Degraded returns the fixed output shown while an intersection is in
// degraded mode: a steady yellow flash on the fault phase.
func Degraded() Output {
	return Output{PhaseID: "fault", Color: "yellow", Manual: false}
}

// Driver owns the live signal output records. It is the only component that
// reads and writes what the field lamps actually display, so the console page
// and the fault/override flows share one source of truth.
type Driver struct {
	fs *store.FileStore
}

// NewDriver creates a signal driver backed by the shared file store.
func NewDriver(fs *store.FileStore) *Driver {
	return &Driver{fs: fs}
}

// Path resolves the output record for an intersection.
func (d *Driver) Path(intersectionID string) string {
	return d.fs.Path("signal", intersectionID)
}

// Write stores the live output for an intersection.
func (d *Driver) Write(intersectionID string, out Output) error {
	return d.fs.WriteJSON(d.Path(intersectionID), out)
}

// Read loads the live output for an intersection.
func (d *Driver) Read(intersectionID string) (Output, error) {
	var out Output
	if err := d.fs.ReadJSON(d.Path(intersectionID), &out); err != nil {
		return Output{}, err
	}
	return out, nil
}
