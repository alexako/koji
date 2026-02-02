// Package personality implements Koji's emotional state machine and
// behavior selection.
package personality

import (
	"time"
)

// Mood represents Koji's current emotional state.
type Mood string

const (
	MoodCurious    Mood = "curious"    // baseline state
	MoodExcited    Mood = "excited"    // new person, play time
	MoodStartled   Mood = "startled"   // sudden stimulus, brief
	MoodFrightened Mood = "frightened" // escalated fear
	MoodHappy      Mood = "happy"      // music, familiar faces
	MoodSleepy     Mood = "sleepy"     // quiet environment
	MoodCautious   Mood = "cautious"   // wary, recovering from fear
)

// Intensity represents how strongly a mood is felt (0.0 to 1.0).
type Intensity float64

const (
	IntensityLow    Intensity = 0.3
	IntensityMedium Intensity = 0.6
	IntensityHigh   Intensity = 0.9
)

// EmotionalState tracks Koji's current mood and how it changes over time.
type EmotionalState struct {
	CurrentMood Mood
	Intensity   Intensity
	EnteredAt   time.Time
	baseline    Mood // mood to decay toward
}

// NewEmotionalState creates a new emotional state starting at the baseline mood.
func NewEmotionalState() *EmotionalState {
	return &EmotionalState{
		CurrentMood: MoodCurious,
		Intensity:   IntensityMedium,
		EnteredAt:   time.Now(),
		baseline:    MoodCurious,
	}
}

// SetMood changes the current mood with the given intensity.
func (e *EmotionalState) SetMood(mood Mood, intensity Intensity) {
	e.CurrentMood = mood
	e.Intensity = intensity
	e.EnteredAt = time.Now()
}

// Duration returns how long we've been in the current mood.
func (e *EmotionalState) Duration() time.Duration {
	return time.Since(e.EnteredAt)
}

// IsBaseline returns true if we're at the baseline mood.
func (e *EmotionalState) IsBaseline() bool {
	return e.CurrentMood == e.baseline
}
