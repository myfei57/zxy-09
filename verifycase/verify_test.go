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

// TestOverrideExpiryFallsBackToPlan verifies an expired manual override hands
// control back to the scheduled plan instead of keeping the forced phase.
func TestOverrideExpiryFallsBackToPlan(t *testing.T) {
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

	p, err := plans.Create("测试方案", []plan.PhaseConfig{
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
	in, err := intersections.Register("测试路口")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := intersections.Activate(in.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	steps, err := plans.StepsFor(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := phases.Start(in.ID, p.ID, steps); err != nil {
		t.Fatal(err)
	}

	ov, err := overrides.Take(in.ID, "N-S", "green", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	state, err := phases.Current(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Forced {
		t.Fatal("override did not force the phase")
	}
	if _, err := overrides.Expire(ov.ID); err != nil {
		t.Fatal(err)
	}
	state, err = phases.Current(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Forced {
		t.Fatalf("expired override kept the forced phase: %+v", state)
	}
}
