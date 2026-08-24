package phase

import "fmt"

// ApplyRollover moves an intersection from its current phase onto the first
// phase of a new plan while preserving the intergreen sequence: the machine
// enters yellow first and only reaches the new green after the clearance
// completes. Skipping the transition would let two conflicting streams share
// the intersection without a gap.
func (m *Machine) ApplyRollover(st State, steps []Step) (State, error) {
	if len(steps) == 0 {
		return State{}, fmt.Errorf("plan %s has no phases", st.PlanID)
	}
	next := State{
		IntersectionID: st.IntersectionID,
		PlanID:         st.PlanID,
		StepIndex:      0,
		PhaseID:        steps[0].PhaseID,
		Color:          ColorYellow,
	}
	if err := m.Save(next); err != nil {
		return State{}, err
	}
	return next, nil
}

// Validate checks that a step list is a valid phase sequence: unique order
// values and at least one step.
func Validate(steps []Step) error {
	if len(steps) == 0 {
		return fmt.Errorf("phase sequence is empty")
	}
	seen := make(map[int]string, len(steps))
	for _, step := range steps {
		if step.PhaseID == "" {
			return fmt.Errorf("phase step has no phase id")
		}
		if existing, ok := seen[step.Order]; ok {
			return fmt.Errorf("phase order %d used by both %s and %s", step.Order, existing, step.PhaseID)
		}
		seen[step.Order] = step.PhaseID
	}
	return nil
}
