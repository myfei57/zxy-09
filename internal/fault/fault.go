package fault

import (
	"fmt"
	"time"

	"signalflow/internal/audit"
	"signalflow/internal/intersection"
	"signalflow/internal/signal"
	"signalflow/internal/store"
)

// Status tracks whether a fault record is still active or has been cleared.
type Status string

const (
	StatusFaulted Status = "faulted"
	StatusCleared Status = "cleared"
)

// Record is the durable fault record for one detected device fault.
type Record struct {
	ID             string    `json:"id"`
	IntersectionID string    `json:"intersection_id"`
	Kind           string    `json:"kind"`
	Status         Status    `json:"status"`
	DetectedAt     time.Time `json:"detected_at"`
	ClearedAt      time.Time `json:"cleared_at"`
}

// FaultTarget is the intersection surface used by the fault service.
type FaultTarget interface {
	Get(id string) (intersection.Intersection, error)
	SwitchToDegraded(id, faultID string) (intersection.Intersection, error)
	RestoreNormal(id string) (intersection.Intersection, error)
	ReArmNormalOutput(id string) (signal.Output, error)
}

// Service detects device faults, degrades the affected intersection, and
// recovers it once the fault is confirmed cleared.
type Service struct {
	fs            *store.FileStore
	intersections FaultTarget
	audit         audit.Recorder
	clock         func() time.Time
}

// NewService creates the fault service.
func NewService(fs *store.FileStore, intersections FaultTarget) *Service {
	return &Service{fs: fs, intersections: intersections, clock: time.Now}
}

// NewServiceWithAudit wires the audit sink into the fault service.
func NewServiceWithAudit(fs *store.FileStore, intersections FaultTarget, recorder audit.Recorder) *Service {
	s := NewService(fs, intersections)
	s.audit = recorder
	return s
}

func (s *Service) now() time.Time {
	return s.clock().UTC()
}

// Path resolves the persisted fault record of an intersection. The control
// plane keeps one current fault record per intersection, keyed by the
// intersection id, so detection, recovery and clear all operate on the same
// durable record and deduplication is a simple status check.
func (s *Service) Path(intersectionID string) string {
	return s.fs.Path("faults", intersectionID)
}

// Load reads the current fault record of an intersection.
func (s *Service) Load(intersectionID string) (Record, error) {
	var rec Record
	if err := s.fs.ReadJSON(s.Path(intersectionID), &rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}

// List returns the current fault record of an intersection, or an empty
// slice when the intersection never had a fault.
func (s *Service) List(intersectionID string) ([]Record, error) {
	rec, err := s.Load(intersectionID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return []Record{rec}, nil
}

// activeFor returns the latest fault record of an intersection that is still
// marked faulted.
func (s *Service) activeFor(intersectionID string) (Record, bool, error) {
	records, err := s.List(intersectionID)
	if err != nil {
		return Record{}, false, err
	}
	for _, rec := range records {
		if rec.Status == StatusFaulted {
			return rec, true, nil
		}
	}
	return Record{}, false, nil
}

func (s *Service) save(rec Record) error {
	return s.fs.WriteJSON(s.Path(rec.IntersectionID), rec)
}

func (s *Service) markCleared(rec Record) error {
	rec.Status = StatusCleared
	rec.ClearedAt = s.now()
	return s.save(rec)
}

// String renders a compact summary of a fault record.
func (r Record) String() string {
	return fmt.Sprintf("%s/%s %s", r.IntersectionID, r.Kind, r.Status)
}
