package webgateway

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	maxSearchQueryBytes = 500
	maxResolveBodyBytes = 64 << 10
	maxIdentifierBytes  = 512
)

type Server struct {
	backend Backend
	token   string
}

type resolveRequest struct {
	ProviderID      string         `json:"providerId"`
	TrackID         string         `json:"trackId"`
	Quality         string         `json:"quality,omitempty"`
	PreparedContext map[string]any `json:"preparedContext,omitempty"`
}

func NewHandler(backend Backend, token string) http.Handler {
	server := &Server{backend: backend, token: strings.TrimSpace(token)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", server.handleHealth)
	mux.HandleFunc("GET /providers", server.handleProviders)
	mux.HandleFunc("GET /search", server.handleSearch)
	mux.HandleFunc("POST /resolve", server.handleResolve)
	return server.authenticate(mux)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	if s.token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if len(provided) != len(s.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	providerCount := s.backend.ProviderCount()
	readiness := "ready"
	if providerCount <= 0 {
		readiness = "no_providers"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"configured":    true,
		"ready":         providerCount > 0,
		"readiness":     readiness,
		"providerCount": providerCount,
		"auth": map[string]bool{
			"required":      s.token != "",
			"authenticated": true,
		},
		"capabilities": map[string]bool{
			"search":    true,
			"resolve":   true,
			"providers": true,
		},
	})
}

func (s *Server) handleProviders(w http.ResponseWriter, _ *http.Request) {
	payload, err := s.backend.Providers()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeRawJSON(w, http.StatusOK, payload)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "search query is required")
		return
	}
	if len(query) > maxSearchQueryBytes {
		writeError(w, http.StatusBadRequest, "search query is too long")
		return
	}

	limit := 20
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	if limit > 50 {
		limit = 50
	}

	payload, err := s.backend.Search(query, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeRawJSON(w, http.StatusOK, payload)
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, maxResolveBodyBytes)
	defer body.Close()

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request resolveRequest
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}

	request.ProviderID = strings.TrimSpace(request.ProviderID)
	request.TrackID = strings.TrimSpace(request.TrackID)
	request.Quality = strings.TrimSpace(request.Quality)
	if request.ProviderID == "" || request.TrackID == "" {
		writeError(w, http.StatusBadRequest, "providerId and trackId are required")
		return
	}
	if len(request.ProviderID) > maxIdentifierBytes || len(request.TrackID) > maxIdentifierBytes || len(request.Quality) > maxIdentifierBytes {
		writeError(w, http.StatusBadRequest, "provider, track, or quality identifier is too long")
		return
	}

	payload, err := s.backend.ResolveStream(request.ProviderID, request.TrackID, request.Quality, request.PreparedContext)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeRawJSON(w, http.StatusOK, payload)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("unexpected second JSON value")
	}
	return err
}

func writeRawJSON(w http.ResponseWriter, status int, payload json.RawMessage) {
	if !json.Valid(payload) {
		writeError(w, http.StatusBadGateway, "backend returned invalid JSON")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
