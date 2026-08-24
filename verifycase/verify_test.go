package verifycase

import (
	"os"
	"testing"

	"signalflow/internal/audit"
	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// TestPlanSwitchesAfterPhaseDurable verifies a plan only becomes published
// after its phase configuration is durably stored.
func TestPlanSwitchesAfterPhaseDurable(t *testing.T) {
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

	p, err := plans.Create("测试方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 30, YellowSeconds: 4, OffsetSeconds: 0},
		{PhaseID: "E-W", Order: 1, GreenSeconds: 25, YellowSeconds: 4, OffsetSeconds: 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Block the phase configuration write by putting a directory at the
	// snapshot path; the durable write must fail before the plan switches.
	block := fs.Path("phaseconfigs", p.ID)
	if err := os.MkdirAll(block, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := plans.Publish(p.ID); err == nil {
		t.Fatal("publish should fail when the phase configuration cannot be written")
	}
	got, err := plans.Get(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != plan.StatusDraft {
		t.Fatalf("plan switched before phase config durable: status = %s", got.Status)
	}
}
