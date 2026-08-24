package engine

import (
	"signalflow/internal/plan"
	"signalflow/internal/schedule"
)

// SeedDemo registers a small demonstration network so the console pages have
// content on first start. It exercises every domain component through its
// real call chains.
func (e *Engine) SeedDemo() error {
	plans, err := e.Plans.List()
	if err != nil {
		return err
	}
	if len(plans) > 0 {
		return nil
	}
	peak, err := e.Plans.Create("早高峰方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 35, YellowSeconds: 4, OffsetSeconds: 0},
		{PhaseID: "E-W", Order: 1, GreenSeconds: 25, YellowSeconds: 4, OffsetSeconds: 10},
	})
	if err != nil {
		return err
	}
	offpeak, err := e.Plans.Create("平峰方案", []plan.PhaseConfig{
		{PhaseID: "N-S", Order: 0, GreenSeconds: 20, YellowSeconds: 3, OffsetSeconds: 0},
		{PhaseID: "E-W", Order: 1, GreenSeconds: 20, YellowSeconds: 3, OffsetSeconds: 8},
	})
	if err != nil {
		return err
	}
	if _, err := e.Plans.Publish(peak.ID); err != nil {
		return err
	}
	if _, err := e.Plans.Publish(offpeak.ID); err != nil {
		return err
	}
	if _, err := e.Plans.SwitchActive(peak.ID, peak.Phases); err != nil {
		return err
	}
	if _, err := e.Plans.SwitchActive(offpeak.ID, offpeak.Phases); err != nil {
		return err
	}
	group, err := e.Coord.Create("中山路绿波", peak.ID)
	if err != nil {
		return err
	}
	for _, name := range []string{"中山路-北口", "中山路-南口", "学院路-东口"} {
		in, err := e.Intersections.Register(name)
		if err != nil {
			return err
		}
		if _, err := e.Intersections.Activate(in.ID, peak.ID); err != nil {
			return err
		}
		steps, err := e.Plans.StepsFor(peak.ID)
		if err != nil {
			return err
		}
		if _, err := e.Phases.Start(in.ID, peak.ID, steps); err != nil {
			return err
		}
		if _, err := e.Coord.Join(group.ID, in.ID); err != nil {
			return err
		}
		if _, err := e.Schedules.Assign(in.ID, []schedule.Band{
			{Name: "早高峰", StartMinute: 7 * 60, PlanID: peak.ID},
			{Name: "平峰", StartMinute: 9 * 60, PlanID: offpeak.ID},
		}); err != nil {
			return err
		}
	}
	_, _ = e.Audit.Record("system", "demo.seed", "network", "demo", "演示数据初始化完成")
	return nil
}
