package coord

// computeOffsets assigns each member the coordination offset of the phase at
// its position in the member list, cycling through the plan's phase offsets.
func computeOffsets(members []string, baseline []int) map[string]int {
	offsets := make(map[string]int, len(members))
	for i, member := range members {
		if len(baseline) == 0 {
			offsets[member] = 0
			continue
		}
		offsets[member] = baseline[i%len(baseline)]
	}
	return offsets
}

// Recalculate recomputes the offsets of a group from its current member list
// and the plan's coordination baseline, then persists the group.
func (c *Coordinator) Recalculate(g Group) error {
	baseline, err := c.plans.OffsetsFor(g.PlanID)
	if err != nil {
		return err
	}
	g.Offsets = computeOffsets(g.Members, baseline.Offsets)
	g.PlanVersion = baseline.Version
	g.UpdatedAt = c.now()
	return c.fs.WriteJSON(c.Path(g.ID), g)
}

// ApplyPlanSwitch refreshes every group bound to a plan after the plan gains
// a new active generation. Groups that already track the latest generation
// are left untouched.
func (c *Coordinator) ApplyPlanSwitch(planID string) error {
	baseline, err := c.plans.OffsetsFor(planID)
	if err != nil {
		return err
	}
	groups, err := c.List()
	if err != nil {
		return err
	}
	for _, g := range groups {
		if g.PlanID != planID {
			continue
		}
		if g.PlanVersion >= baseline.Version {
			continue
		}
		g.Offsets = computeOffsets(g.Members, baseline.Offsets)
		g.PlanVersion = baseline.Version
		g.UpdatedAt = c.now()
		if err := c.fs.WriteJSON(c.Path(g.ID), g); err != nil {
			return err
		}
	}
	return nil
}
