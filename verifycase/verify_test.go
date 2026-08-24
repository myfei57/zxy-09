package verifycase

import (
	"testing"

	"signalflow/internal/audit"
	"signalflow/internal/coord"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// TestCoordOffsetsRefreshAfterPlanSwitch verifies a coordinated group picks
// up the offsets of the new plan generation after a plan switch.
func TestCoordOffsetsRefreshAfterPlanSwitch(t *testing.T) {
	root := t.TempDir()
	st, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	fs := st.FS
	recorder := audit.NewService(fs)
	signals := signal.NewDriver(fs)
	intersections := intersection.NewService(fs, signals)
	phaseConfigs := phase.NewConfigStore(fs)
	plans := plan.NewService(fs, phaseConfigs, intersections, recorder)
	coordinator := coord.NewCoordinatorWithAudit(fs, plans, recorder)
	plans.SetCoordRefresher(coordinator)

	p, err := plans.Create("绿波方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 30, YellowSeconds: 4, OffsetSeconds: 0},
		{PhaseID: "E-W", Order: 1, GreenSeconds: 25, YellowSeconds: 4, OffsetSeconds: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plans.Publish(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := plans.SwitchActive(p.ID, p.Phases); err != nil {
		t.Fatal(err)
	}
	g, err := coordinator.Create("中山路绿波", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Join(g.ID, "member-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Join(g.ID, "member-2"); err != nil {
		t.Fatal(err)
	}

	// Re-activate the plan with new offsets; the group must follow the new
	// generation instead of keeping the previous plan's offsets.
	newPhases := []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 30, YellowSeconds: 4, OffsetSeconds: 5},
		{PhaseID: "E-W", Order: 1, GreenSeconds: 25, YellowSeconds: 4, OffsetSeconds: 15},
	}
	if _, err := plans.SwitchActive(p.ID, newPhases); err != nil {
		t.Fatal(err)
	}

	got, err := coordinator.Get(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Offsets["member-1"] != 5 || got.Offsets["member-2"] != 15 {
		t.Fatalf("offsets not refreshed after plan switch: %v", got.Offsets)
	}
}
