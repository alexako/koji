// Package api provides HTTP endpoints for external devices (displays, sensors).
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alex/koji/internal/personality"
)

// EventHandler processes incoming sensor events.
type EventHandler interface {
	HandleEvent(ctx personality.EventContext) bool
}

// StateProvider gives access to Koji's current emotional state.
type StateProvider interface {
	GetState() *personality.EmotionalState
	GetRecentAction() string
}

// Server provides HTTP API for external devices.
type Server struct {
	addr         string
	provider     StateProvider
	eventHandler EventHandler

	mu           sync.RWMutex
	lastAction   string
	lastActionAt time.Time
}

// NewServer creates a new API server.
func NewServer(addr string, provider StateProvider, eventHandler EventHandler) *Server {
	return &Server{
		addr:         addr,
		provider:     provider,
		eventHandler: eventHandler,
	}
}

// SetLastAction records the most recent action taken.
func (s *Server) SetLastAction(action string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAction = action
	s.lastActionAt = time.Now()
}

// StateResponse is the JSON response for /api/state.
type StateResponse struct {
	Mood         string  `json:"mood"`
	Intensity    float64 `json:"intensity"`
	DurationMs   int64   `json:"duration_ms"`
	FaceEmotion  string  `json:"face_emotion"`
	EmotionIndex int     `json:"emotion_index"`
	Action       string  `json:"action,omitempty"`
	ActionAge    int64   `json:"action_age_ms,omitempty"`
}

// EventRequest is the JSON body for POST /api/event.
type EventRequest struct {
	Event     string            `json:"event"`
	Intensity float64           `json:"intensity,omitempty"`
	Source    string            `json:"source,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// EventResponse is the JSON response for POST /api/event.
type EventResponse struct {
	Accepted    bool   `json:"accepted"`
	MoodChanged bool   `json:"mood_changed"`
	NewMood     string `json:"new_mood,omitempty"`
}

// Start begins serving the API.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/event", s.handleEvent)
	mux.HandleFunc("/health", s.handleHealth)

	server := &http.Server{
		Addr:    s.addr,
		Handler: corsMiddleware(mux),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("API server starting on %s", s.addr)
	return server.ListenAndServe()
}

// handleState returns current emotional state.
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := s.provider.GetState()
	if state == nil {
		http.Error(w, "state not available", http.StatusServiceUnavailable)
		return
	}

	s.mu.RLock()
	action := s.lastAction
	actionAt := s.lastActionAt
	s.mu.RUnlock()

	faceEmotion := state.ToFaceEmotion()
	resp := StateResponse{
		Mood:         string(state.CurrentMood),
		Intensity:    float64(state.Intensity),
		DurationMs:   state.Duration().Milliseconds(),
		FaceEmotion:  string(faceEmotion),
		EmotionIndex: faceEmotion.Index(),
	}

	// Include action if recent (within 5 seconds)
	if action != "" && time.Since(actionAt) < 5*time.Second {
		resp.Action = action
		resp.ActionAge = time.Since(actionAt).Milliseconds()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleEvent receives sensor events from the Pi.
func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Event == "" {
		http.Error(w, "event field is required", http.StatusBadRequest)
		return
	}

	// Build event context
	ctx := personality.EventContext{
		Event:     personality.Event(req.Event),
		Intensity: req.Intensity,
		Source:    req.Source,
		Metadata:  req.Metadata,
	}

	// Default intensity if not provided
	if ctx.Intensity == 0 {
		ctx.Intensity = 0.5
	}
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]string)
	}

	// Process the event
	moodChanged := false
	if s.eventHandler != nil {
		moodChanged = s.eventHandler.HandleEvent(ctx)
	}

	resp := EventResponse{
		Accepted:    true,
		MoodChanged: moodChanged,
	}

	if moodChanged && s.provider != nil {
		state := s.provider.GetState()
		if state != nil {
			resp.NewMood = string(state.CurrentMood)
		}
	}

	log.Printf("Event received: %s (intensity=%.2f, source=%s) -> mood_changed=%v",
		req.Event, ctx.Intensity, req.Source, moodChanged)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth is a simple health check endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// corsMiddleware adds CORS headers for ESP32 or browser access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
