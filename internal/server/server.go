package server

import (
	"encoding/json"
	"github.com/stockyard-dev/stockyard-switchboard/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.mux.HandleFunc("GET /api/services", s.list)
	s.mux.HandleFunc("POST /api/services", s.register)
	s.mux.HandleFunc("GET /api/services/{id}", s.get)
	s.mux.HandleFunc("PUT /api/services/{id}", s.update)
	s.mux.HandleFunc("DELETE /api/services/{id}", s.deregister)
	s.mux.HandleFunc("POST /api/services/{id}/heartbeat", s.heartbeat)
	s.mux.HandleFunc("GET /api/discover", s.discover)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/switchboard/"})
	})
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", http.StatusFound)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"services": orEmpty(s.db.List())})
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var svc store.Service
	json.NewDecoder(r.Body).Decode(&svc)
	if svc.Name == "" {
		writeErr(w, 400, "name required")
		return
	}
	s.db.Register(&svc)
	writeJSON(w, 201, s.db.Get(svc.ID))
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	svc := s.db.Get(r.PathValue("id"))
	if svc == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, svc)
}
func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ex := s.db.Get(id)
	if ex == nil {
		writeErr(w, 404, "not found")
		return
	}
	var svc store.Service
	json.NewDecoder(r.Body).Decode(&svc)
	if svc.Name == "" {
		svc.Name = ex.Name
	}
	if svc.URL == "" {
		svc.URL = ex.URL
	}
	if svc.Tags == nil {
		svc.Tags = ex.Tags
	}
	if svc.Status == "" {
		svc.Status = ex.Status
	}
	s.db.Update(id, &svc)
	writeJSON(w, 200, s.db.Get(id))
}
func (s *Server) deregister(w http.ResponseWriter, r *http.Request) {
	s.db.Deregister(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}
func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	s.db.Heartbeat(r.PathValue("id"))
	writeJSON(w, 200, s.db.Get(r.PathValue("id")))
}
func (s *Server) discover(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"services": orEmpty(s.db.Discover(r.URL.Query().Get("tag")))})
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.db.Stats()
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "switchboard", "total": st.Total, "up": st.Up})
}
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

// ─── personalization (auto-added) ──────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("%s: warning: could not parse config.json: %v", "switchboard", err)
		return
	}
	s.pCfg = cfg
	log.Printf("%s: loaded personalization from %s", "switchboard", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, 400)
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		http.Error(w, `{"error":"save failed"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":"saved"}`))
}
