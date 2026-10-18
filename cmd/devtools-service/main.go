package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"example.com/devtools-build-rollout/internal/releasegate"
)

const flagKey = "devtools-build-diagnostics"

type server struct {
	flags *releasegate.Client
}

type rolloutRequest struct {
	Percentage float64 `json:"percentage"`
	Salt       string  `json:"salt"`
	StickyUnit string  `json:"sticky_unit"`
	Version    int     `json:"version"`
	RequestID  string  `json:"request_id"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	s := &server{flags: releasegate.NewClient(key)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /build-events", s.buildEvent)
	mux.HandleFunc("POST /release-operations/rollout", s.rollout)
	addr := ":8080"
	if value := os.Getenv("PORT"); value != "" {
		addr = ":" + value
	}
	log.Printf("devtools service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) buildEvent(w http.ResponseWriter, r *http.Request) {
	var event releasegate.BuildEvent
	if err := decodeJSON(r, &event); err != nil || event.BuildID == "" || event.Developer == "" || event.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "build_id, developer, and status are required"})
		return
	}
	enabled, err := s.flags.IsEnabled(r.Context(), flagKey)
	if err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, releasegate.DecideBuild(event, enabled))
}

func (s *server) rollout(w http.ResponseWriter, r *http.Request) {
	var input rolloutRequest
	if err := decodeJSON(r, &input); err != nil || input.RequestID == "" || input.Version < 1 || input.Percentage < 0 || input.Percentage > 100 || input.Salt == "" || input.StickyUnit == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id, version, salt, sticky_unit, and percentage from 0 to 100 are required"})
		return
	}
	err := s.flags.SetRollout(r.Context(), flagKey, releasegate.Rollout{
		Percentage: input.Percentage,
		Salt:       input.Salt,
		StickyUnit: input.StickyUnit,
		Version:    input.Version,
	}, input.RequestID)
	if err != nil {
		writeClientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flag": flagKey, "percentage": input.Percentage})
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeClientError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	var apiErr *releasegate.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		status = apiErr.StatusCode
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(jsonSize(value)))
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func jsonSize(value any) int {
	data, _ := json.Marshal(value)
	return len(data) + 1
}
