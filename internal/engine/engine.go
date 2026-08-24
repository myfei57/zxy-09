package engine

import (
	"fmt"

	"signalflow/internal/audit"
	"signalflow/internal/config"
	"signalflow/internal/coord"
	"signalflow/internal/detector"
	"signalflow/internal/fault"
	"signalflow/internal/intersection"
	"signalflow/internal/override"
	"signalflow/internal/phase"
	"signalflow/internal/plan"
	"signalflow/internal/schedule"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// Engine wires every control-plane component together and drives the tick
// loop: override expiry, schedule rollover and phase advancement.
type Engine struct {
	cfg           config.Config
	Intersections *intersection.Service
	Plans         *plan.Service
	Phases        *phase.Machine
	Schedules     *schedule.Service
	Coord         *coord.Coordinator
	Detector      *detector.Service
	Faults        *fault.Service
	Overrides     *override.Service
	Audit         *audit.Service
	Signals       *signal.Driver
}

// Build constructs the full engine from configuration, opening the file store
// and wiring the dependency graph in one place.
func Build(cfg config.Config) (*Engine, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	fs := st.FS
	recorder := audit.NewService(fs)
	signals := signal.NewDriver(fs)
	intersections := intersection.NewService(fs, signals)
	phaseConfigs := phase.NewConfigStore(fs)
	phases := phase.NewMachine(fs)
	plans := plan.NewService(fs, phaseConfigs, intersections, recorder)
	coordinator := coord.NewCoordinatorWithAudit(fs, plans, recorder)
	plans.SetCoordRefresher(coordinator)
	schedules := schedule.NewService(fs, phases, plans)
	detectorSvc := detector.NewService(fs, plans)
	faults := fault.NewServiceWithAudit(fs, intersections, recorder)
	overrides := override.NewServiceWithAudit(fs, phases, plans, intersections, recorder)
	return &Engine{
		cfg:           cfg,
		Intersections: intersections,
		Plans:         plans,
		Phases:        phases,
		Schedules:     schedules,
		Coord:         coordinator,
		Detector:      detectorSvc,
		Faults:        faults,
		Overrides:     overrides,
		Audit:         recorder,
		Signals:       signals,
	}, nil
}

// ApplyPlanToIntersection attaches a plan to an intersection and restarts its
// phase machine on that plan.
func (e *Engine) ApplyPlanToIntersection(intersectionID, planID string) (intersection.Intersection, error) {
	in, err := e.Intersections.SetPlan(intersectionID, planID)
	if err != nil {
		return intersection.Intersection{}, err
	}
	steps, err := e.Plans.StepsFor(planID)
	if err != nil {
		return intersection.Intersection{}, err
	}
	if _, err := e.Phases.Start(intersectionID, planID, steps); err != nil {
		return intersection.Intersection{}, err
	}
	return in, nil
}

// ApplyDetectorSample records a raw detector reading and folds the measured
// volumes into the intersection's committed plan.
func (e *Engine) ApplyDetectorSample(sample detector.Sample) (plan.Plan, error) {
	recorded, err := e.Detector.RecordSample(sample)
	if err != nil {
		return plan.Plan{}, err
	}
	return e.Detector.ApplyVolumes(recorded.IntersectionID, map[string]int{recorded.PhaseID: recorded.Vehicles})
}

// IntersectionView is the consolidated console view of one intersection.
type IntersectionView struct {
	Intersection intersection.Intersection `json:"intersection"`
	Output       signal.Output             `json:"output"`
	Phase        *phase.State              `json:"phase,omitempty"`
	Override     *override.Override        `json:"override,omitempty"`
}

// Snapshot builds the consolidated view used by the console status endpoint.
func (e *Engine) Snapshot() ([]IntersectionView, error) {
	all, err := e.Intersections.List()
	if err != nil {
		return nil, err
	}
	views := make([]IntersectionView, 0, len(all))
	for _, in := range all {
		view := IntersectionView{Intersection: in}
		if out, err := e.Signals.Read(in.ID); err == nil {
			view.Output = out
		}
		if st, err := e.Phases.Current(in.ID); err == nil {
			copyState := st
			view.Phase = &copyState
		}
		if ov, active, err := e.Overrides.ActiveFor(in.ID); err == nil && active {
			copyOverride := ov
			view.Override = &copyOverride
		}
		views = append(views, view)
	}
	return views, nil
}

// OverrideTimeoutSeconds returns the configured default takeover duration.
func (e *Engine) OverrideTimeoutSeconds() int {
	return e.cfg.OverrideTimeoutSeconds
}
