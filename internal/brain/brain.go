// Package brain orchestrates Koji's emotional state, event processing, and behavior.
package brain

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/alex/koji/internal/personality"
)

// Brain is the central orchestrator for Koji's emotional state and behavior.
type Brain struct {
	state *personality.EmotionalState

	mu           sync.RWMutex
	recentEvents []personality.Event
	lastAction   string
	// Configuration
	decayInterval time.Duration
	maxEvents     int
}

// Config holds configuration for the Brain.
type Config struct {
	DecayInterval time.Duration // How often to check for mood decay
	MaxEvents     int           // How many recent events to remember
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DecayInterval: 1 * time.Second,
		MaxEvents:     10,
	}
}

// New creates a new Brain with the given configuration.
func New(cfg Config) *Brain {
	return &Brain{
		state:         personality.NewEmotionalState(),
		recentEvents:  make([]personality.Event, 0, cfg.MaxEvents),
		decayInterval: cfg.DecayInterval,
		maxEvents:     cfg.MaxEvents,
	}
}

// GetState returns the current emotional state (implements StateProvider).
func (b *Brain) GetState() *personality.EmotionalState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// GetRecentAction returns the most recent action (implements StateProvider).
func (b *Brain) GetRecentAction() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastAction
}

// HandleEvent processes an incoming event (implements EventHandler).
func (b *Brain) HandleEvent(ctx personality.EventContext) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Track recent events
	b.recentEvents = append(b.recentEvents, ctx.Event)
	if len(b.recentEvents) > b.maxEvents {
		b.recentEvents = b.recentEvents[1:]
	}

	// Process the event through the state machine
	changed := b.state.ProcessEvent(ctx)

	if changed {
		log.Printf("Mood changed to %s (intensity=%.2f) due to %s",
			b.state.CurrentMood, b.state.Intensity, ctx.Event)
	}

	return changed
}

// SetAction records the last action taken.
func (b *Brain) SetAction(action string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastAction = action
}

// RecentEvents returns the recent event history.
func (b *Brain) RecentEvents() []personality.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	events := make([]personality.Event, len(b.recentEvents))
	copy(events, b.recentEvents)
	return events
}

// Run starts the brain's main loop (decay timer, etc).
// Blocks until context is cancelled.
func (b *Brain) Run(ctx context.Context) error {
	ticker := time.NewTicker(b.decayInterval)
	defer ticker.Stop()

	log.Printf("Brain started: mood=%s, decay_interval=%s",
		b.state.CurrentMood, b.decayInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Brain shutting down")
			return ctx.Err()

		case <-ticker.C:
			b.mu.Lock()
			if b.state.Decay() {
				log.Printf("Mood decayed to %s (intensity=%.2f)",
					b.state.CurrentMood, b.state.Intensity)
			}
			b.mu.Unlock()
		}
	}
}

// CurrentMood returns the current mood (convenience method).
func (b *Brain) CurrentMood() personality.Mood {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state.CurrentMood
}

// CurrentIntensity returns the current intensity (convenience method).
func (b *Brain) CurrentIntensity() personality.Intensity {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state.Intensity
}
