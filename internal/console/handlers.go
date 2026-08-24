package console

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"signalflow/internal/detector"
	"signalflow/internal/plan"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, value any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(value)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	views, err := s.engine.Snapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) listIntersections(w http.ResponseWriter, r *http.Request) {
	all, err := s.engine.Intersections.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) createIntersection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	in, err := s.engine.Intersections.Register(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

func (s *Server) getIntersection(w http.ResponseWriter, r *http.Request) {
	in, err := s.engine.Intersections.Get(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) activateIntersection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanID string `json:"plan_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	in, err := s.engine.Intersections.Activate(chi.URLParam(r, "id"), req.PlanID)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) switchIntersectionPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanID string `json:"plan_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	in, err := s.engine.ApplyPlanToIntersection(chi.URLParam(r, "id"), req.PlanID)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) simulateFault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind string `json:"kind"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Kind == "" {
		req.Kind = "detector-offline"
	}
	rec, err := s.engine.Faults.Detect(chi.URLParam(r, "id"), req.Kind)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) recoverIntersection(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.Faults.Recover(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "recovered"})
}

func (s *Server) clearIntersection(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.Faults.Clear(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	all, err := s.engine.Plans.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) createPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string             `json:"name"`
		Phases []plan.PhaseConfig `json:"phases"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	p, err := s.engine.Plans.Create(req.Name, req.Phases)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.engine.Plans.Get(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) publishPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.engine.Plans.Publish(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) switchPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phases []plan.PhaseConfig `json:"phases"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	p, err := s.engine.Plans.SwitchActive(chi.URLParam(r, "id"), req.Phases)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) editPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phases []plan.PhaseConfig `json:"phases"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	draft, err := s.engine.Plans.BeginEdit(chi.URLParam(r, "id"), req.Phases)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (s *Server) commitPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.engine.Plans.Commit(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	all, err := s.engine.Coord.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		PlanID string `json:"plan_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	g, err := s.engine.Coord.Create(req.Name, req.PlanID)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) joinGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IntersectionID string `json:"intersection_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	g, err := s.engine.Coord.Join(chi.URLParam(r, "id"), req.IntersectionID)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) leaveGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IntersectionID string `json:"intersection_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	g, err := s.engine.Coord.Leave(chi.URLParam(r, "id"), req.IntersectionID)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) groupWave(w http.ResponseWriter, r *http.Request) {
	report, err := s.engine.Coord.ValidateGreenWave(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) detectorSample(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IntersectionID string `json:"intersection_id"`
		PhaseID        string `json:"phase_id"`
		Vehicles       int    `json:"vehicles"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sample := detector.Sample{
		IntersectionID: req.IntersectionID,
		PhaseID:        req.PhaseID,
		Vehicles:       req.Vehicles,
	}
	p, err := s.engine.ApplyDetectorSample(sample)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) listOverrides(w http.ResponseWriter, r *http.Request) {
	all, err := s.engine.Overrides.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) takeOverride(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IntersectionID string `json:"intersection_id"`
		PhaseID        string `json:"phase_id"`
		Color          string `json:"color"`
		Duration       int    `json:"duration_seconds"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Duration <= 0 {
		req.Duration = s.engine.OverrideTimeoutSeconds()
	}
	if req.Color == "" {
		req.Color = "green"
	}
	ov, err := s.engine.Overrides.Take(req.IntersectionID, req.PhaseID, req.Color, time.Duration(req.Duration)*time.Second)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, ov)
}

func (s *Server) cancelOverride(w http.ResponseWriter, r *http.Request) {
	ov, err := s.engine.Overrides.Cancel(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) auditList(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	entries, err := s.engine.Audit.List(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
