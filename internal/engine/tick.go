package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"signalflow/internal/intersection"
	"signalflow/internal/phase"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// Run starts the control tick loop and blocks until the context is cancelled.
func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(e.cfg.TickSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if err := e.Tick(now); err != nil {
				return err
			}
		}
	}
}

// Tick advances the control plane one step: expired overrides hand control
// back, schedules roll over at band boundaries, and phase machines move
// through their lamp sequences.
func (e *Engine) Tick(now time.Time) error {
	if _, err := e.Overrides.ExpireDue(now); err != nil {
		return err
	}
	rollovers, err := e.Schedules.RolloverDue(now)
	if err != nil {
		return err
	}
	for _, rollover := range rollovers {
		_, _ = e.Audit.Record(
			"engine",
			"schedule.rollover",
			"intersection",
			rollover.IntersectionID,
			fmt.Sprintf("%s -> %s plan %s", rollover.FromBand, rollover.ToBand, rollover.PlanID),
		)
	}
	return e.refreshOutputs(now)
}

func (e *Engine) refreshOutputs(now time.Time) error {
	all, err := e.Intersections.List()
	if err != nil {
		return err
	}
	for _, in := range all {
		switch in.ControlMode {
		case intersection.ModeDegraded:
			if err := e.Signals.Write(in.ID, signal.Degraded()); err != nil {
				return err
			}
		case intersection.ModeNormal:
			if err := e.stepNormal(in, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) stepNormal(in intersection.Intersection, now time.Time) error {
	steps, err := e.Plans.StepsFor(in.PlanID)
	if err != nil {
		return err
	}
	st, err := e.Phases.Current(in.ID)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		st, err = e.Phases.Start(in.ID, in.PlanID, steps)
		if err != nil {
			return err
		}
	}
	if st.Forced {
		return e.Signals.Write(in.ID, signal.Output{PhaseID: st.PhaseID, Color: st.Color, Manual: true})
	}
	started, _ := time.Parse(time.RFC3339, st.UpdatedAt)
	if now.Sub(started) >= holdSeconds(st, steps) {
		st, err = e.Phases.Advance(st, steps)
		if err != nil {
			return err
		}
	}
	return e.Signals.Write(in.ID, signal.Output{PhaseID: steps[st.StepIndex].PhaseID, Color: st.Color, Manual: false})
}

// holdSeconds returns how long the current lamp color must stay on.
func holdSeconds(st phase.State, steps []phase.Step) time.Duration {
	if st.StepIndex < 0 || st.StepIndex >= len(steps) {
		return time.Second
	}
	switch st.Color {
	case phase.ColorGreen:
		return time.Duration(steps[st.StepIndex].GreenSeconds) * time.Second
	case phase.ColorYellow:
		return time.Duration(steps[st.StepIndex].YellowSeconds) * time.Second
	default:
		return time.Second
	}
}

// SeedDemo registers a small demonstration network so the console pages have
// content on first start. It exercises every domain component through its
// real call chains.
