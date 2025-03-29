package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/marcusziade/jewornotjew/pkg/db"
)

// Server represents the API server
type Server struct {
	db     *db.DB
	router *mux.Router
}

// NewServer creates a new API server
func NewServer(db *db.DB) *Server {
	s := &Server{
		db:     db,
		router: mux.NewRouter(),
	}
	s.routes()
	return s
}

// routes sets up the routes for the API server
func (s *Server) routes() {
	s.router.HandleFunc("/api/profiles", s.listProfiles).Methods("GET")
	s.router.HandleFunc("/api/profiles/{name}", s.getProfile).Methods("GET")
	s.router.HandleFunc("/api/search", s.searchProfiles).Methods("GET")
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// ListenAndServe starts the API server
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("API server listening on %s", addr)
	return http.ListenAndServe(addr, s)
}

// listProfiles handles GET /api/profiles
func (s *Server) listProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.db.ListProfiles()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list profiles: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profiles); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// getProfile handles GET /api/profiles/{name}
func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	profile, err := s.db.GetProfile(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Profile not found: %s", name), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get profile: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// searchProfiles handles GET /api/search?q=query
func (s *Server) searchProfiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	profiles, err := s.db.SearchProfiles(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search profiles: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profiles); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}