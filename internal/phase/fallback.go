package phase

import "fmt"

// FallbackToPlan hands control of an intersection back to its scheduled plan
// after a manual override expires or is cancelled. The machine restarts at
// the first phase in red and clears the forced marker so the control loop
// resumes normal advancement.
func (m *Machine) FallbackToPlan(intersectionID, planID string, steps []Step) (State, error) {
	if len(steps) == 0 {
		return State{}, fmt.Errorf("plan %s has no phases", planID)
	}
	st, err := m.Current(intersectionID)
	if err != nil {
		return State{}, err
	}
	st.PlanID = planID
	st.StepIndex = 0
	st.PhaseID = steps[0].PhaseID
	st.Color = ColorRed
	st.Forced = false
	if err := m.Save(st); err != nil {
		return State{}, err
	}
	return st, nil
}
