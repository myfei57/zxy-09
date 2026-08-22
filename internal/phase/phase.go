package phase

import (
	"fmt"
	"time"

	"signalflow/internal/store"
)

// Colors of the lamp sequence. A phase always moves red -> red-yellow ->
// green -> yellow -> red; transitions are never skipped.
const (
	ColorRed       = "red"
	ColorRedYellow = "red-yellow"
	ColorGreen     = "green"
	ColorYellow    = "yellow"
)

// Step describes one phase inside a timing plan: its order, the green hold
// time and the yellow clearance time applied by the control loop.
type Step struct {
	PhaseID       string `json:"phase_id"`
	Order         int    `json:"order"`
	GreenSeconds  int    `json:"green_seconds"`
	YellowSeconds int    `json:"yellow_seconds"`
}

// State is the persisted position of the phase machine for one intersection.
type State struct {
	IntersectionID string `json:"intersection_id"`
	PlanID         string `json:"plan_id"`
	StepIndex      int    `json:"step_index"`
	PhaseID        string `json:"phase_id"`
	Color          string `json:"color"`
	Forced         bool   `json:"forced"`
	UpdatedAt      string `json:"updated_at"`
}

// Machine runs the signal phase machine for every intersection. It owns the
// per-intersection phase state record; the engine feeds it steps from the
// active timing plan.
type Machine struct {
	fs    *store.FileStore
	clock func() time.Time
}

// NewMachine creates the phase machine over the shared store.
func NewMachine(fs *store.FileStore) *Machine {
	return &Machine{fs: fs, clock: time.Now}
}

func (m *Machine) now() time.Time {
	return m.clock().UTC()
}

// Path resolves the phase state record for an intersection.
func (m *Machine) Path(intersectionID string) string {
	return m.fs.Path("phase", intersectionID)
}

// Current loads the phase state of an intersection.
func (m *Machine) Current(intersectionID string) (State, error) {
	var st State
	if err := m.fs.ReadJSON(m.Path(intersectionID), &st); err != nil {
		return State{}, err
	}
	return st, nil
}

// Save persists a phase state record.
func (m *Machine) Save(st State) error {
	st.UpdatedAt = m.now().Format(time.RFC3339)
	return m.fs.WriteJSON(m.Path(st.IntersectionID), st)
}

// Start begins a plan at its first phase in red.
func (m *Machine) Start(intersectionID, planID string, steps []Step) (State, error) {
	if len(steps) == 0 {
		return State{}, fmt.Errorf("plan %s has no phases", planID)
	}
	st := State{
		IntersectionID: intersectionID,
		PlanID:         planID,
		StepIndex:      0,
		PhaseID:        steps[0].PhaseID,
		Color:          ColorRed,
	}
	if err := m.Save(st); err != nil {
		return State{}, err
	}
	return st, nil
}

// Force holds the machine at a manually selected phase and color.
func (m *Machine) Force(intersectionID, phaseID, color string) (State, error) {
	st, err := m.Current(intersectionID)
	if err != nil {
		if err == store.ErrNotFound {
			st = State{IntersectionID: intersectionID}
		} else {
			return State{}, err
		}
	}
	st.StepIndex = -1
	st.PhaseID = phaseID
	st.Color = color
	st.Forced = true
	if err := m.Save(st); err != nil {
		return State{}, err
	}
	return st, nil
}

// Advance moves the machine one step forward in the lamp sequence. The color
// sequence is red -> red-yellow -> green -> yellow -> red; reaching yellow
// from green advances to the next phase's red.
func (m *Machine) Advance(st State, steps []Step) (State, error) {
	if len(steps) == 0 {
		return State{}, fmt.Errorf("plan %s has no phases", st.PlanID)
	}
	if st.Forced {
		return State{}, fmt.Errorf("phase machine %s is under manual override", st.IntersectionID)
	}
	if st.StepIndex < 0 || st.StepIndex >= len(steps) {
		st.StepIndex = 0
	}
	switch st.Color {
	case ColorRed:
		st.Color = ColorRedYellow
	case ColorRedYellow:
		st.Color = ColorGreen
	case ColorGreen:
		st.Color = ColorYellow
	case ColorYellow:
		st.StepIndex = (st.StepIndex + 1) % len(steps)
		st.Color = ColorRed
	default:
		return State{}, fmt.Errorf("phase machine %s is in unknown color %q", st.IntersectionID, st.Color)
	}
	st.PhaseID = steps[st.StepIndex].PhaseID
	if err := m.Save(st); err != nil {
		return State{}, err
	}
	return st, nil
}
