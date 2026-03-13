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

	// Test mode: override emotion for testing
	testEmotionOverride int
	testEmotionExpiry   time.Time
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
// Includes full state so the ESP32 can react immediately without a second request.
type EventResponse struct {
	Accepted     bool    `json:"accepted"`
	MoodChanged  bool    `json:"mood_changed"`
	Mood         string  `json:"mood"`
	Intensity    float64 `json:"intensity"`
	FaceEmotion  string  `json:"face_emotion"`
	EmotionIndex int     `json:"emotion_index"`
}

// TestEmotionResponse is the JSON response for /api/test/emotion.
type TestEmotionResponse struct {
	EmotionIndex int    `json:"emotion_index"`
	EmotionName  string `json:"emotion_name"`
	DurationSec  int    `json:"duration_sec"`
}

// Start begins serving the API.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/event", s.handleEvent)
	mux.HandleFunc("/api/test/emotion", s.handleTestEmotion)
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
	testOverride := s.testEmotionOverride
	testExpiry := s.testEmotionExpiry
	s.mu.RUnlock()

	faceEmotion := state.ToFaceEmotion()
	resp := StateResponse{
		Mood:         string(state.CurrentMood),
		Intensity:    float64(state.Intensity),
		DurationMs:   state.Duration().Milliseconds(),
		FaceEmotion:  string(faceEmotion),
		EmotionIndex: faceEmotion.Index(),
	}

	// Check for test emotion override
	if time.Now().Before(testExpiry) && testOverride >= 0 && testOverride < len(emotionNames) {
		resp.EmotionIndex = testOverride
		resp.FaceEmotion = emotionNames[testOverride]
		resp.Mood = "test"
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

	// Build response with full state for immediate reaction
	state := s.provider.GetState()
	faceEmotion := state.ToFaceEmotion()

	resp := EventResponse{
		Accepted:     true,
		MoodChanged:  moodChanged,
		Mood:         string(state.CurrentMood),
		Intensity:    float64(state.Intensity),
		FaceEmotion:  string(faceEmotion),
		EmotionIndex: faceEmotion.Index(),
	}

	log.Printf("Event received: %s (intensity=%.2f, source=%s) -> mood_changed=%v, emotion=%s",
		req.Event, ctx.Intensity, req.Source, moodChanged, faceEmotion)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth is a simple health check endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

var emotionNames = []string{
	"normal", "angry", "glee", "happy", "sad", "worried", "focused", "annoyed",
	"surprised", "skeptic", "frustrated", "unimpressed", "sleepy", "suspicious",
	"squint", "furious", "scared", "awe",
}

// handleTestEmotion allows directly setting emotion for testing.
// GET /api/test/emotion?index=5&duration=3
func (s *Server) handleTestEmotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params
	indexStr := r.URL.Query().Get("index")
	if indexStr == "" {
		http.Error(w, "index parameter required (0-17)", http.StatusBadRequest)
		return
	}

	var index int
	if err := json.Unmarshal([]byte(indexStr), &index); err != nil || index < 0 || index > 17 {
		http.Error(w, "index must be 0-17", http.StatusBadRequest)
		return
	}

	durationStr := r.URL.Query().Get("duration")
	duration := 3 // default 3 seconds
	if durationStr != "" {
		_ = json.Unmarshal([]byte(durationStr), &duration)
	}

	// Set the override
	s.mu.Lock()
	s.testEmotionOverride = index
	s.testEmotionExpiry = time.Now().Add(time.Duration(duration) * time.Second)
	s.mu.Unlock()

	resp := TestEmotionResponse{
		EmotionIndex: index,
		EmotionName:  emotionNames[index],
		DurationSec:  duration,
	}

	log.Printf("TEST: Setting emotion to %d (%s) for %d seconds", index, emotionNames[index], duration)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
