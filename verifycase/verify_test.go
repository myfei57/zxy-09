package verifycase

import (
	"testing"

	"signalflow/internal/audit"
	"signalflow/internal/fault"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// TestFaultClearRearmsSignalOutput verifies clearing a fault re-arms the live
// signal output in step with the control-plane state.
func TestFaultClearRearmsSignalOutput(t *testing.T) {
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
	faults := fault.NewServiceWithAudit(fs, intersections, recorder)

	p, err := plans.Create("测试方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 30, YellowSeconds: 4, OffsetSeconds: 0},
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
	if _, err := faults.Detect(in.ID, "detector-offline"); err != nil {
		t.Fatal(err)
	}
	if err := signals.Write(in.ID, signal.Degraded()); err != nil {
		t.Fatal(err)
	}
	if err := faults.Clear(in.ID); err != nil {
		t.Fatal(err)
	}
	out, err := signals.Read(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out.Manual || out.Color != "red" {
		t.Fatalf("live output not re-armed after fault clear: %+v", out)
	}
}
