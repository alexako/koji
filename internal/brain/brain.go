// Package brain orchestrates Koji's emotional state, event processing, and behavior.
package brain

import (
	"context"
	"log"
	"math/rand"
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
	lastEventAt  time.Time // tracks when we last got external stimulus

	// Configuration
	decayInterval time.Duration
	maxEvents     int
	idleEnabled   bool // enable puppy-like idle behavior
}

// Config holds configuration for the Brain.
type Config struct {
	DecayInterval time.Duration // How often to check for mood decay
	MaxEvents     int           // How many recent events to remember
	IdleEnabled   bool          // Enable puppy-like idle behavior (sleepy/curious cycling)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DecayInterval: 1 * time.Second,
		MaxEvents:     10,
		IdleEnabled:   true,
	}
}

// New creates a new Brain with the given configuration.
func New(cfg Config) *Brain {
	return &Brain{
		state:         personality.NewEmotionalState(),
		recentEvents:  make([]personality.Event, 0, cfg.MaxEvents),
		lastEventAt:   time.Now(),
		decayInterval: cfg.DecayInterval,
		maxEvents:     cfg.MaxEvents,
		idleEnabled:   cfg.IdleEnabled,
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

	// Update last event time (for idle behavior)
	b.lastEventAt = time.Now()

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

// Idle behavior constants - like a puppy dozing off then perking up
const (
	idleCheckInterval = 5 * time.Second  // how often to check for idle behavior
	idleMinQuietTime  = 30 * time.Second // minimum quiet time before getting sleepy
	idleSleepyChance  = 0.15             // chance per check to get sleepy when idle
	idlePerkUpChance  = 0.20             // chance per check to perk back up when sleepy
	idleMinSleepyTime = 10 * time.Second // minimum time before perking up
)

// Run starts the brain's main loop (decay timer, etc).
// Blocks until context is cancelled.
func (b *Brain) Run(ctx context.Context) error {
	decayTicker := time.NewTicker(b.decayInterval)
	defer decayTicker.Stop()

	idleTicker := time.NewTicker(idleCheckInterval)
	defer idleTicker.Stop()

	log.Printf("Brain started: mood=%s, decay_interval=%s",
		b.state.CurrentMood, b.decayInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Brain shutting down")
			return ctx.Err()

		case <-decayTicker.C:
			b.mu.Lock()
			if b.state.Decay() {
				log.Printf("Mood decayed to %s (intensity=%.2f)",
					b.state.CurrentMood, b.state.Intensity)
			}
			b.mu.Unlock()

		case <-idleTicker.C:
			if b.idleEnabled {
				b.checkIdleBehavior()
			}
		}
	}
}

// checkIdleBehavior implements puppy-like behavior: getting sleepy when idle,
// then randomly perking back up to curious.
func (b *Brain) checkIdleBehavior() {
	b.mu.Lock()
	defer b.mu.Unlock()

	timeSinceEvent := time.Since(b.lastEventAt)
	currentMood := b.state.CurrentMood

	switch currentMood {
	case personality.MoodCurious:
		// If it's been quiet and we're just curious, maybe get sleepy
		if timeSinceEvent > idleMinQuietTime && rand.Float64() < idleSleepyChance {
			b.state.SetMood(personality.MoodSleepy, personality.IntensityLow)
			log.Printf("Idle: getting sleepy (quiet for %s)", timeSinceEvent.Round(time.Second))
		}

	case personality.MoodSleepy:
		// If we've been sleepy for a bit, maybe perk back up
		if b.state.Duration() > idleMinSleepyTime && rand.Float64() < idlePerkUpChance {
			b.state.SetMood(personality.MoodCurious, personality.IntensityMedium)
			log.Printf("Idle: perking up! (was sleepy for %s)", b.state.Duration().Round(time.Second))
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
