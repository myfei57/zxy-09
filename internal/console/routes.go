package console

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) routes() {
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/intersections", http.StatusFound)
	})
	s.router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	s.router.Get("/ui/intersections", s.page("intersections.html"))
	s.router.Get("/ui/plans", s.page("plans.html"))
	s.router.Get("/ui/groups", s.page("groups.html"))
	s.router.Get("/ui/audit", s.page("audit.html"))

	s.router.Route("/api", func(r chi.Router) {
		r.Get("/status", s.status)
		r.Route("/intersections", func(r chi.Router) {
			r.Get("/", s.listIntersections)
			r.Post("/", s.createIntersection)
			r.Get("/{id}", s.getIntersection)
			r.Post("/{id}/activate", s.activateIntersection)
			r.Post("/{id}/plan", s.switchIntersectionPlan)
			r.Post("/{id}/fault", s.simulateFault)
			r.Post("/{id}/recover", s.recoverIntersection)
			r.Post("/{id}/clear", s.clearIntersection)
		})
		r.Route("/plans", func(r chi.Router) {
			r.Get("/", s.listPlans)
			r.Post("/", s.createPlan)
			r.Get("/{id}", s.getPlan)
			r.Post("/{id}/publish", s.publishPlan)
			r.Post("/{id}/switch", s.switchPlan)
			r.Post("/{id}/edit", s.editPlan)
			r.Post("/{id}/commit", s.commitPlan)
		})
		r.Route("/groups", func(r chi.Router) {
			r.Get("/", s.listGroups)
			r.Post("/", s.createGroup)
			r.Post("/{id}/join", s.joinGroup)
			r.Post("/{id}/leave", s.leaveGroup)
			r.Get("/{id}/wave", s.groupWave)
		})
		r.Post("/detector/samples", s.detectorSample)
		r.Route("/overrides", func(r chi.Router) {
			r.Get("/", s.listOverrides)
			r.Post("/", s.takeOverride)
			r.Post("/{id}/cancel", s.cancelOverride)
		})
		r.Get("/audit", s.auditList)
	})
}
