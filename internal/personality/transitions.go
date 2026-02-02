package personality

import "time"

// MoodTransition defines what mood results from an event given the current mood.
type MoodTransition struct {
	NewMood   Mood
	Intensity Intensity
}

// transitionTable maps events to mood transitions based on current mood.
// If an event/mood combo isn't in here, mood doesn't change.
var transitionTable = map[Event]map[Mood]MoodTransition{
	// Loud noise startles, escalates if already on edge
	EventLoudNoise: {
		MoodCurious:    {MoodStartled, IntensityHigh},
		MoodHappy:      {MoodStartled, IntensityMedium},
		MoodSleepy:     {MoodFrightened, IntensityHigh}, // rude awakening
		MoodStartled:   {MoodFrightened, IntensityHigh}, // escalate
		MoodCautious:   {MoodFrightened, IntensityHigh}, // already wary
		MoodExcited:    {MoodStartled, IntensityMedium}, // excitement interrupted
		MoodFrightened: {MoodFrightened, IntensityHigh}, // stay scared
	},

	// Music makes happy, helps recover from fear
	EventMusic: {
		MoodCurious:    {MoodHappy, IntensityMedium},
		MoodSleepy:     {MoodCurious, IntensityLow},     // gentle wake
		MoodCautious:   {MoodCurious, IntensityMedium},  // calming
		MoodStartled:   {MoodCautious, IntensityMedium}, // helps, but still wary
		MoodFrightened: {MoodCautious, IntensityMedium}, // calming effect
		MoodHappy:      {MoodHappy, IntensityHigh},      // more happy!
		MoodExcited:    {MoodHappy, IntensityHigh},      // good vibes
	},

	// Rhythm detected - time to bop
	EventRhythm: {
		MoodCurious: {MoodHappy, IntensityMedium},
		MoodHappy:   {MoodExcited, IntensityHigh}, // let's dance!
		MoodExcited: {MoodExcited, IntensityHigh}, // keep dancing
	},

	// Familiar face is comforting
	EventFamiliarFace: {
		MoodCurious:    {MoodHappy, IntensityMedium},
		MoodCautious:   {MoodHappy, IntensityMedium},
		MoodFrightened: {MoodCautious, IntensityMedium}, // calming but still shaken
		MoodStartled:   {MoodCautious, IntensityLow},    // oh it's just you
		MoodSleepy:     {MoodHappy, IntensityLow},       // sleepy but pleased
		MoodHappy:      {MoodExcited, IntensityHigh},    // yay you're here!
		MoodExcited:    {MoodExcited, IntensityHigh},    // still excited
	},

	// Unknown face - who dis?
	EventUnknownFace: {
		MoodCurious:    {MoodCautious, IntensityMedium},
		MoodHappy:      {MoodCautious, IntensityLow},    // hmm, who's this
		MoodSleepy:     {MoodCautious, IntensityMedium}, // alert now
		MoodCautious:   {MoodCautious, IntensityHigh},   // stay wary
		MoodFrightened: {MoodFrightened, IntensityHigh}, // stranger danger!
		MoodStartled:   {MoodFrightened, IntensityHigh}, // bad combo
		MoodExcited:    {MoodCautious, IntensityMedium}, // wait who are you
	},

	// Motion detected - something's happening
	EventMotionDetected: {
		MoodCurious: {MoodExcited, IntensityMedium}, // ooh what's that
		MoodSleepy:  {MoodCurious, IntensityLow},    // hmm?
		MoodHappy:   {MoodExcited, IntensityMedium}, // something to check out!
	},

	// Unknown object - what's that thing?
	EventUnknownObject: {
		MoodCurious:    {MoodExcited, IntensityHigh},   // ooh what IS that!
		MoodHappy:      {MoodExcited, IntensityHigh},   // new thing to check out!
		MoodSleepy:     {MoodCurious, IntensityMedium}, // hmm? *perks up*
		MoodExcited:    {MoodExcited, IntensityHigh},   // another thing!!
		MoodCautious:   {MoodCurious, IntensityMedium}, // interesting... but careful
		MoodStartled:   {MoodCautious, IntensityHigh},  // is that what scared me?
		MoodFrightened: {MoodCautious, IntensityHigh},  // what is it? *peeks*
	},

	// Being petted is nice
	EventPetted: {
		MoodCurious:    {MoodHappy, IntensityMedium},
		MoodCautious:   {MoodHappy, IntensityMedium},  // oh that's nice
		MoodFrightened: {MoodCautious, IntensityLow},  // calming down
		MoodStartled:   {MoodCautious, IntensityLow},  // oh it's okay
		MoodHappy:      {MoodHappy, IntensityHigh},    // more pets!
		MoodExcited:    {MoodHappy, IntensityHigh},    // aww yes
		MoodSleepy:     {MoodSleepy, IntensityMedium}, // mmm sleepy pets
	},

	// Being poked is annoying
	EventPoked: {
		MoodCurious:  {MoodStartled, IntensityMedium},
		MoodSleepy:   {MoodStartled, IntensityHigh},   // rude!
		MoodHappy:    {MoodCurious, IntensityMedium},  // hey what
		MoodCautious: {MoodStartled, IntensityMedium}, // don't!
	},

	// Silence over time makes sleepy
	EventSilence: {
		MoodCurious:  {MoodSleepy, IntensityLow},
		MoodHappy:    {MoodCurious, IntensityLow},  // winding down
		MoodCautious: {MoodCurious, IntensityLow},  // things seem okay
		MoodExcited:  {MoodHappy, IntensityMedium}, // calming down
	},

	// Time passing with nothing happening
	EventTimePassedLong: {
		MoodCurious:  {MoodSleepy, IntensityMedium},
		MoodCautious: {MoodCurious, IntensityMedium}, // coast is clear
		MoodHappy:    {MoodCurious, IntensityMedium}, // back to baseline
		MoodExcited:  {MoodHappy, IntensityMedium},   // calming down
	},
}

// decayPaths defines how moods decay toward baseline over time.
// Each mood decays to the next mood in its path, eventually reaching curious.
var decayPaths = map[Mood]Mood{
	MoodFrightened: MoodCautious,
	MoodCautious:   MoodCurious,
	MoodStartled:   MoodCautious,
	MoodExcited:    MoodHappy,
	MoodHappy:      MoodCurious,
	MoodSleepy:     MoodCurious,
	MoodCurious:    MoodCurious, // baseline, no decay
}

// decayTimes defines how long before a mood decays to the next state.
var decayTimes = map[Mood]time.Duration{
	MoodFrightened: 15 * time.Second,
	MoodStartled:   5 * time.Second,
	MoodCautious:   20 * time.Second,
	MoodExcited:    30 * time.Second,
	MoodHappy:      45 * time.Second,
	MoodSleepy:     60 * time.Second,
}

// ProcessEvent updates the emotional state based on an incoming event.
// Returns true if the mood changed.
func (e *EmotionalState) ProcessEvent(ctx EventContext) bool {
	eventTransitions, ok := transitionTable[ctx.Event]
	if !ok {
		return false // unknown event, no change
	}

	transition, ok := eventTransitions[e.CurrentMood]
	if !ok {
		return false // no transition defined for this mood
	}

	// Scale intensity by event intensity
	newIntensity := transition.Intensity
	if ctx.Intensity > 0.7 {
		newIntensity = IntensityHigh
	} else if ctx.Intensity < 0.3 {
		newIntensity = IntensityLow
	}

	oldMood := e.CurrentMood
	e.SetMood(transition.NewMood, newIntensity)
	return oldMood != e.CurrentMood
}

// Decay checks if enough time has passed and decays the mood toward baseline.
// Returns true if the mood changed.
func (e *EmotionalState) Decay() bool {
	if e.IsBaseline() {
		return false
	}

	decayTime, ok := decayTimes[e.CurrentMood]
	if !ok {
		return false
	}

	if e.Duration() < decayTime {
		return false // not time yet
	}

	nextMood := decayPaths[e.CurrentMood]
	if nextMood == e.CurrentMood {
		return false // already at end of decay path
	}

	// Decay intensity slightly when transitioning
	newIntensity := e.Intensity - 0.2
	if newIntensity < IntensityLow {
		newIntensity = IntensityLow
	}

	e.SetMood(nextMood, newIntensity)
	return true
}
