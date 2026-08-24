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

// TestDegradeWaitsForRecordDurable verifies normal control is only released
// after the degradation record is durably stored.
func TestDegradeWaitsForRecordDurable(t *testing.T) {
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

	// Block the degradation record write by making the target read-only:
	// reads keep working while the durable rename fails.
	block := fs.Path("faults", in.ID)
	if err := os.WriteFile(block, []byte(`{"status":"cleared"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(block, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := faults.Detect(in.ID, "detector-offline"); err == nil {
		t.Fatal("degrade should fail when the fault record cannot be written")
	}
	got, err := intersections.Get(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ControlMode != intersection.ModeNormal {
		t.Fatalf("normal control released before degrade record durable: mode = %q", got.ControlMode)
	}
}
