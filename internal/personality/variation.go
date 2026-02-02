package personality

import (
	"math/rand"
	"time"
)

// WeightedAction pairs an action with a probability weight.
type WeightedAction struct {
	Action Action
	Weight float64 // relative weight, doesn't need to sum to 1
}

// ActionModifier changes how an action is performed.
type ActionModifier string

const (
	ModifierSlow     ActionModifier = "slow"
	ModifierNormal   ActionModifier = "normal"
	ModifierFast     ActionModifier = "fast"
	ModifierFrantic  ActionModifier = "frantic"
	ModifierGentle   ActionModifier = "gentle"
	ModifierHesitant ActionModifier = "hesitant"
	ModifierEager    ActionModifier = "eager"
)

// ModifiedAction is an action with a modifier describing how to perform it.
type ModifiedAction struct {
	Action   Action
	Modifier ActionModifier
}

// MicroBehavior represents a small idle animation or twitch.
type MicroBehavior struct {
	Name     string
	Duration time.Duration
}

// Common micro-behaviors that can happen during idle moments.
var microBehaviors = map[Mood][]WeightedMicroBehavior{
	MoodCurious: {
		{MicroBehavior{"ear_twitch", 200 * time.Millisecond}, 3.0},
		{MicroBehavior{"look_around", 500 * time.Millisecond}, 2.0},
		{MicroBehavior{"sniff", 300 * time.Millisecond}, 2.0},
		{MicroBehavior{"weight_shift", 400 * time.Millisecond}, 1.5},
		{MicroBehavior{"tail_flick", 150 * time.Millisecond}, 1.0},
	},
	MoodHappy: {
		{MicroBehavior{"tail_wag_small", 300 * time.Millisecond}, 4.0},
		{MicroBehavior{"ear_perk", 200 * time.Millisecond}, 2.0},
		{MicroBehavior{"wiggle", 400 * time.Millisecond}, 2.0},
		{MicroBehavior{"happy_sigh", 500 * time.Millisecond}, 1.0},
	},
	MoodExcited: {
		{MicroBehavior{"bounce_small", 250 * time.Millisecond}, 4.0},
		{MicroBehavior{"tail_wag_fast", 200 * time.Millisecond}, 3.0},
		{MicroBehavior{"spin_partial", 400 * time.Millisecond}, 2.0},
		{MicroBehavior{"eager_lean", 300 * time.Millisecond}, 2.0},
	},
	MoodSleepy: {
		{MicroBehavior{"slow_blink", 800 * time.Millisecond}, 4.0},
		{MicroBehavior{"yawn_small", 600 * time.Millisecond}, 2.0},
		{MicroBehavior{"head_droop", 700 * time.Millisecond}, 2.0},
		{MicroBehavior{"sleepy_sigh", 500 * time.Millisecond}, 1.5},
		{MicroBehavior{"ear_droop", 300 * time.Millisecond}, 1.0},
	},
	MoodCautious: {
		{MicroBehavior{"ear_swivel", 250 * time.Millisecond}, 4.0},
		{MicroBehavior{"freeze_brief", 400 * time.Millisecond}, 2.0},
		{MicroBehavior{"low_crouch", 350 * time.Millisecond}, 2.0},
		{MicroBehavior{"nervous_glance", 300 * time.Millisecond}, 3.0},
		{MicroBehavior{"tail_tuck_partial", 200 * time.Millisecond}, 1.5},
	},
	MoodStartled: {
		{MicroBehavior{"flinch", 150 * time.Millisecond}, 4.0},
		{MicroBehavior{"ears_back_quick", 100 * time.Millisecond}, 3.0},
		{MicroBehavior{"gasp", 200 * time.Millisecond}, 2.0},
		{MicroBehavior{"freeze_tense", 300 * time.Millisecond}, 2.0},
	},
	MoodFrightened: {
		{MicroBehavior{"tremble", 400 * time.Millisecond}, 4.0},
		{MicroBehavior{"whimper_soft", 300 * time.Millisecond}, 3.0},
		{MicroBehavior{"shrink", 350 * time.Millisecond}, 2.0},
		{MicroBehavior{"eyes_dart", 250 * time.Millisecond}, 2.0},
		{MicroBehavior{"tail_between_legs", 200 * time.Millisecond}, 1.5},
	},
}

// WeightedMicroBehavior pairs a micro-behavior with a probability weight.
type WeightedMicroBehavior struct {
	Behavior MicroBehavior
	Weight   float64
}

// weightedMoodActions maps moods to weighted action choices.
// Higher weights mean more likely to be chosen.
var weightedMoodActions = map[Mood][]WeightedAction{
	MoodCurious: {
		{ActionExplore, 4.0},  // most likely - curious pets explore
		{ActionPerkEars, 3.0}, // listening
		{ActionTiltHead, 3.0}, // the classic curious head tilt
		{ActionApproach, 2.0}, // investigate
		{ActionStay, 1.0},     // sometimes just watch
		{ActionChirp, 1.5},    // occasional curious sounds
		{ActionSniff, 2.5},    // what's that smell?
	},
	MoodExcited: {
		{ActionBounce, 4.0},   // bouncy bouncy!
		{ActionWagTail, 4.0},  // tail going crazy
		{ActionSpin, 3.0},     // happy spins
		{ActionApproach, 3.0}, // gotta get closer!
		{ActionBark, 2.0},     // excited barks
		{ActionPerkEars, 2.0}, // alert and happy
		{ActionChirp, 2.5},    // happy sounds
		{ActionExplore, 1.5},  // too excited to focus
	},
	MoodHappy: {
		{ActionWagTail, 5.0},  // happy tail
		{ActionNuzzle, 3.0},   // affectionate
		{ActionPurr, 3.0},     // content sounds
		{ActionStay, 2.5},     // content to chill
		{ActionHeadBob, 2.0},  // vibing
		{ActionChirp, 2.0},    // happy chirps
		{ActionApproach, 1.5}, // want to be near
		{ActionExplore, 1.0},  // might wander happily
	},
	MoodStartled: {
		{ActionFreeze, 5.0},      // deer in headlights
		{ActionPerkEars, 4.0},    // what was that?!
		{ActionCrouch, 3.0},      // defensive
		{ActionRetreat, 2.5},     // back away
		{ActionWhimper, 2.0},     // scared sound
		{ActionFlee, 1.5},        // might bolt
		{ActionFlattenEars, 2.0}, // scared ears
	},
	MoodFrightened: {
		{ActionFlee, 5.0},        // NOPE
		{ActionCrouch, 4.0},      // make self small
		{ActionWhimper, 4.0},     // scared sounds
		{ActionFlattenEars, 3.5}, // terrified ears
		{ActionRetreat, 3.0},     // backing away
		{ActionPeek, 2.0},        // is it gone?
		{ActionFreeze, 1.5},      // too scared to move
	},
	MoodCautious: {
		{ActionPeek, 4.0},        // careful observation
		{ActionPerkEars, 4.0},    // listening carefully
		{ActionFreeze, 3.0},      // holding still
		{ActionStay, 3.0},        // staying put
		{ActionRetreat, 2.5},     // might back off
		{ActionGrowl, 2.0},       // warning sound
		{ActionFlattenEars, 1.5}, // wary
		{ActionWhimper, 1.0},     // nervous
	},
	MoodSleepy: {
		{ActionCurl, 5.0}, // curl up for nap
		{ActionYawn, 4.0}, // so sleepy
		{ActionStay, 3.5}, // too tired to move
		{ActionPurr, 2.0}, // sleepy purrs
	},
}

// MoodEcho represents lingering effects from a previous mood.
type MoodEcho struct {
	FromMood  Mood
	Strength  float64 // 0.0 to 1.0, how much it affects current behavior
	StartedAt time.Time
}

// echoEffects defines how past moods bleed into current behavior.
// Key is past mood, value maps to current mood modifications.
var echoEffects = map[Mood]struct {
	DecayTime time.Duration
	Effects   map[Mood][]WeightedAction // additional actions that might trigger
}{
	MoodFrightened: {
		DecayTime: 45 * time.Second, // stays jumpy for a while
		Effects: map[Mood][]WeightedAction{
			MoodCurious: {
				{ActionPeek, 2.0},        // still peeking nervously
				{ActionFlattenEars, 1.5}, // ears still back sometimes
				{ActionFreeze, 1.0},      // occasional freeze
			},
			MoodCautious: {
				{ActionWhimper, 1.5}, // still whimpering
				{ActionCrouch, 1.0},  // staying low
			},
			MoodHappy: {
				{ActionPeek, 1.0}, // checking that everything's really okay
			},
		},
	},
	MoodStartled: {
		DecayTime: 20 * time.Second,
		Effects: map[Mood][]WeightedAction{
			MoodCurious: {
				{ActionPerkEars, 2.0}, // extra alert
				{ActionFreeze, 1.0},   // brief freezes
			},
			MoodCautious: {
				{ActionFlinch, 1.5}, // still flinchy (pseudo-action for variation)
			},
		},
	},
	MoodExcited: {
		DecayTime: 30 * time.Second,
		Effects: map[Mood][]WeightedAction{
			MoodHappy: {
				{ActionBounce, 2.0},  // still a bit bouncy
				{ActionWagTail, 1.5}, // tail still going
			},
			MoodCurious: {
				{ActionBounce, 1.0}, // occasional excited bounce
			},
		},
	},
	MoodHappy: {
		DecayTime: 60 * time.Second,
		Effects: map[Mood][]WeightedAction{
			MoodCurious: {
				{ActionWagTail, 1.5}, // still a bit waggy
				{ActionChirp, 1.0},   // occasional happy sound
			},
		},
	},
}

// VariationEngine adds lifelike variation to Koji's behavior.
type VariationEngine struct {
	rng         *rand.Rand
	moodHistory []MoodEcho
	maxHistory  int
}

// NewVariationEngine creates a new variation engine.
func NewVariationEngine() *VariationEngine {
	return &VariationEngine{
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		moodHistory: make([]MoodEcho, 0, 8),
		maxHistory:  8,
	}
}

// RecordMoodChange records a mood transition for echo effects.
func (v *VariationEngine) RecordMoodChange(fromMood Mood) {
	echo := MoodEcho{
		FromMood:  fromMood,
		Strength:  1.0,
		StartedAt: time.Now(),
	}

	// Add to history, evicting oldest if needed
	if len(v.moodHistory) >= v.maxHistory {
		v.moodHistory = v.moodHistory[1:]
	}
	v.moodHistory = append(v.moodHistory, echo)
}

// GetActiveEchoes returns mood echoes that are still affecting behavior.
func (v *VariationEngine) GetActiveEchoes() []MoodEcho {
	now := time.Now()
	active := make([]MoodEcho, 0)

	for _, echo := range v.moodHistory {
		effect, ok := echoEffects[echo.FromMood]
		if !ok {
			continue
		}

		elapsed := now.Sub(echo.StartedAt)
		if elapsed >= effect.DecayTime {
			continue // echo has faded
		}

		// Calculate remaining strength (linear decay)
		strength := 1.0 - (float64(elapsed) / float64(effect.DecayTime))
		active = append(active, MoodEcho{
			FromMood:  echo.FromMood,
			Strength:  strength,
			StartedAt: echo.StartedAt,
		})
	}

	return active
}

// SelectAction picks an action using weighted randomness, mood echoes, and intensity.
func (v *VariationEngine) SelectAction(state *EmotionalState) ModifiedAction {
	// Start with base weighted actions for current mood
	actions := v.getWeightedActions(state.CurrentMood)

	// Add echo effects from previous moods
	echoes := v.GetActiveEchoes()
	for _, echo := range echoes {
		effect, ok := echoEffects[echo.FromMood]
		if !ok {
			continue
		}
		echoActions, ok := effect.Effects[state.CurrentMood]
		if !ok {
			continue
		}
		// Add echo actions with reduced weight based on echo strength
		for _, wa := range echoActions {
			actions = append(actions, WeightedAction{
				Action: wa.Action,
				Weight: wa.Weight * echo.Strength,
			})
		}
	}

	// Pick an action using weighted random selection
	action := v.weightedRandomChoice(actions)

	// Determine modifier based on intensity
	modifier := v.intensityToModifier(state.Intensity, state.CurrentMood)

	return ModifiedAction{
		Action:   action,
		Modifier: modifier,
	}
}

// SelectMicroBehavior picks a random micro-behavior for the current mood.
// Returns nil if no micro-behavior should occur (also random).
func (v *VariationEngine) SelectMicroBehavior(mood Mood) *MicroBehavior {
	// 30% chance of no micro-behavior (natural pauses)
	if v.rng.Float64() < 0.3 {
		return nil
	}

	behaviors, ok := microBehaviors[mood]
	if !ok || len(behaviors) == 0 {
		return nil
	}

	// Weighted random selection
	var totalWeight float64
	for _, wb := range behaviors {
		totalWeight += wb.Weight
	}

	r := v.rng.Float64() * totalWeight
	var cumulative float64
	for _, wb := range behaviors {
		cumulative += wb.Weight
		if r <= cumulative {
			return &wb.Behavior
		}
	}

	return &behaviors[0].Behavior
}

// getWeightedActions returns the weighted actions for a mood.
func (v *VariationEngine) getWeightedActions(mood Mood) []WeightedAction {
	actions, ok := weightedMoodActions[mood]
	if !ok {
		return weightedMoodActions[MoodCurious] // fallback
	}
	// Return a copy to avoid modifying the original
	result := make([]WeightedAction, len(actions))
	copy(result, actions)
	return result
}

// weightedRandomChoice picks an action using weighted random selection.
func (v *VariationEngine) weightedRandomChoice(actions []WeightedAction) Action {
	if len(actions) == 0 {
		return ActionStay
	}

	var totalWeight float64
	for _, wa := range actions {
		totalWeight += wa.Weight
	}

	r := v.rng.Float64() * totalWeight
	var cumulative float64
	for _, wa := range actions {
		cumulative += wa.Weight
		if r <= cumulative {
			return wa.Action
		}
	}

	return actions[0].Action
}

// intensityToModifier maps emotional intensity to action modifiers.
func (v *VariationEngine) intensityToModifier(intensity Intensity, mood Mood) ActionModifier {
	// Add some randomness to the modifier selection
	jitter := (v.rng.Float64() - 0.5) * 0.2 // +/- 0.1
	adjustedIntensity := float64(intensity) + jitter

	// Mood-specific modifier mappings
	switch mood {
	case MoodFrightened, MoodStartled:
		if adjustedIntensity > 0.8 {
			return ModifierFrantic
		} else if adjustedIntensity > 0.5 {
			return ModifierFast
		}
		return ModifierHesitant

	case MoodExcited:
		if adjustedIntensity > 0.8 {
			return ModifierFrantic
		} else if adjustedIntensity > 0.5 {
			return ModifierEager
		}
		return ModifierFast

	case MoodSleepy:
		if adjustedIntensity > 0.7 {
			return ModifierSlow
		}
		return ModifierGentle

	case MoodCautious:
		if adjustedIntensity > 0.6 {
			return ModifierHesitant
		}
		return ModifierSlow

	case MoodHappy:
		if adjustedIntensity > 0.7 {
			return ModifierEager
		}
		return ModifierNormal

	default: // Curious and others
		if adjustedIntensity > 0.7 {
			return ModifierEager
		} else if adjustedIntensity < 0.3 {
			return ModifierGentle
		}
		return ModifierNormal
	}
}

// ActionFlinch is a pseudo-action used for echo effects.
const ActionFlinch Action = "flinch"

// ActionSniff is used for curious sniffing behavior.
const ActionSniff Action = "sniff"
