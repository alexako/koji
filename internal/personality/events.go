package personality

// Event represents something that happened in the environment.
type Event string

const (
	// Sound events
	EventLoudNoise Event = "loud_noise"
	EventMusic     Event = "music"
	EventSpeech    Event = "speech"
	EventSilence   Event = "silence"
	EventRhythm    Event = "rhythm" // beat detected

	// Vision events
	EventFamiliarFace   Event = "familiar_face"
	EventUnknownFace    Event = "unknown_face"
	EventMotionDetected Event = "motion_detected"
	EventNoMotion       Event = "no_motion"
	EventUnknownObject  Event = "unknown_object"

	// Physical events
	EventPetted   Event = "petted"    // touch sensor triggered gently
	EventPoked    Event = "poked"     // touch sensor triggered sharply
	EventPickedUp Event = "picked_up" // accelerometer detects lift

	// Time-based events
	EventTimePassedShort  Event = "time_passed_short"  // ~10s of nothing
	EventTimePassedMedium Event = "time_passed_medium" // ~30s of nothing
	EventTimePassedLong   Event = "time_passed_long"   // ~2min of nothing
)

// EventContext provides additional information about an event.
type EventContext struct {
	Event     Event
	Intensity float64           // 0.0 to 1.0, how strong/loud/fast
	Source    string            // where it came from (if known)
	Metadata  map[string]string // additional info
}

// NewEventContext creates a new event context with default values.
func NewEventContext(event Event) EventContext {
	return EventContext{
		Event:     event,
		Intensity: 0.5,
		Metadata:  make(map[string]string),
	}
}

// WithIntensity sets the intensity and returns the context for chaining.
func (ec EventContext) WithIntensity(intensity float64) EventContext {
	ec.Intensity = intensity
	return ec
}

// WithSource sets the source and returns the context for chaining.
func (ec EventContext) WithSource(source string) EventContext {
	ec.Source = source
	return ec
}
