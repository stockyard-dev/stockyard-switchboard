package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stockyard-dev/stockyard-switchboard/internal/store"
)

type Server struct {
	db  *store.DB
	mux *http.ServeMux
}

func New(db *store.DB) *Server {
	s := &Server{db: db, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/services", s.list)
	s.mux.HandleFunc("POST /api/services", s.create)
	s.mux.HandleFunc("GET /api/services/{id}", s.get)
	s.mux.HandleFunc("DELETE /api/services/{id}", s.del)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func wj(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func we(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" { http.Redirect(w, r, "/ui", http.StatusFound); return }
	http.NotFound(w, r)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	wj(w, map[string]any{"status": "ok", "service": "stockyard-switchboard", "time": time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	wj(w, map[string]any{"services": s.db.Count()})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	items := s.db.List()
	if items == nil { items = []store.Service{} }
	wj(w, map[string]any{"services": items, "count": len(items)})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var e store.Service
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		we(w, 400, "invalid JSON"); return
	}
	if err := s.db.Create(&e); err != nil {
		we(w, 500, fmt.Sprintf("create: %v", err)); return
	}
	w.WriteHeader(201)
	wj(w, e)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	e := s.db.Get(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, e)
}

func (s *Server) del(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Delete(r.PathValue("id")); err != nil {
		we(w, 500, fmt.Sprintf("delete: %v", err)); return
	}
	wj(w, map[string]string{"status": "deleted"})
}
