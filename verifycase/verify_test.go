package verifycase

import (
	"os"
	"testing"

	"signalflow/internal/audit"
	"signalflow/internal/fault"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// TestRecoveryWaitsForClearedDurable verifies an intersection returns to
// normal control only after the fault-cleared record is durable.
func TestRecoveryWaitsForClearedDurable(t *testing.T) {
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

	// Make the fault record read-only so the cleared write fails while the
	// record stays readable for the recovery lookup.
	block := fs.Path("faults", in.ID)
	if err := os.Chmod(block, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := faults.Recover(in.ID); err == nil {
		t.Fatal("recovery should fail when the cleared record cannot be written")
	}
	got, err := intersections.Get(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != intersection.StatusDegraded {
		t.Fatalf("intersection left degraded state before cleared record durable: status = %s", got.Status)
	}
}
