package verifycase

import (
	"testing"
	"time"

	"signalflow/internal/audit"
	"signalflow/internal/intersection"
	"signalflow/internal/override"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// TestOverrideCancelUsesCurrentPlan verifies cancelling an override returns
// to the current plan, never the snapshot captured at takeover.
func TestOverrideCancelUsesCurrentPlan(t *testing.T) {
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
	phases := phase.NewMachine(fs)
	plans := plan.NewService(fs, phaseConfigs, intersections, recorder)
	overrides := override.NewServiceWithAudit(fs, phases, plans, intersections, recorder)

	planA, err := plans.Create("旧方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 30, YellowSeconds: 4, OffsetSeconds: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	planB, err := plans.Create("新方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 40, YellowSeconds: 4, OffsetSeconds: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plans.Publish(planA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := plans.SwitchActive(planA.ID, planA.Phases); err != nil {
		t.Fatal(err)
	}
	if _, err := plans.Publish(planB.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := plans.SwitchActive(planB.ID, planB.Phases); err != nil {
		t.Fatal(err)
	}
	in, err := intersections.Register("测试路口")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := intersections.Activate(in.ID, planA.ID); err != nil {
		t.Fatal(err)
	}
	stepsA, err := plans.StepsFor(planA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := phases.Start(in.ID, planA.ID, stepsA); err != nil {
		t.Fatal(err)
	}
	ov, err := overrides.Take(in.ID, "N-S", "green", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// The underlying plan changes while the override is active.
	if _, err := intersections.SetPlan(in.ID, planB.ID); err != nil {
		t.Fatal(err)
	}
	stepsB, err := plans.StepsFor(planB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := phases.Start(in.ID, planB.ID, stepsB); err != nil {
		t.Fatal(err)
	}

	if _, err := overrides.Cancel(ov.ID); err != nil {
		t.Fatal(err)
	}
	state, err := phases.Current(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.PlanID != planB.ID {
		t.Fatalf("cancel replayed a stale plan snapshot: phase plan = %s, want %s", state.PlanID, planB.ID)
	}
}
