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

// TestCoordUsesCurrentMembersAfterJoin verifies offset calculation uses the
// current member list after an intersection joins a coordinated group.
func TestCoordUsesCurrentMembersAfterJoin(t *testing.T) {
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

	got, err := coordinator.Get(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Offsets["member-2"]; !ok {
		t.Fatalf("new member missing from green wave offsets: %v", got.Offsets)
	}
}
