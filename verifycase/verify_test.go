package verifycase

import (
	"testing"
	"time"

	"signalflow/internal/audit"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/schedule"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// TestRolloverKeepsTransitionPhase verifies a schedule rollover moves through
// the transition phase instead of jumping straight to the new green.
func TestRolloverKeepsTransitionPhase(t *testing.T) {
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
	now := time.Date(2026, 8, 22, 0, 5, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	schedules := schedule.NewServiceWithClock(fs, phases, plans, clock)

	planA, err := plans.Create("早高峰", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 35, YellowSeconds: 4, OffsetSeconds: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	planB, err := plans.Create("平峰", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 20, YellowSeconds: 3, OffsetSeconds: 0},
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
	if _, err := schedules.Assign(in.ID, []schedule.Band{
		{Name: "早高峰", StartMinute: 0, PlanID: planA.ID},
		{Name: "平峰", StartMinute: 10, PlanID: planB.ID},
	}); err != nil {
		t.Fatal(err)
	}

	rolloverAt := time.Date(2026, 8, 22, 0, 15, 0, 0, time.UTC)
	if _, err := schedules.Rollover(in.ID, rolloverAt); err != nil {
		t.Fatal(err)
	}
	phaseState, err := phases.Current(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if phaseState.Color != phase.ColorYellow {
		t.Fatalf("rollover skipped the transition phase: color = %s", phaseState.Color)
	}
}
