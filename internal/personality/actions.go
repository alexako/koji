package personality

// Action represents something Koji can physically do.
type Action string

const (
	// Movement actions
	ActionStay     Action = "stay"     // don't move
	ActionExplore  Action = "explore"  // wander around
	ActionFlee     Action = "flee"     // run away from stimulus
	ActionApproach Action = "approach" // move toward stimulus
	ActionRetreat  Action = "retreat"  // back away slowly
	ActionFreeze   Action = "freeze"   // stop and assess

	// Expressive actions
	ActionWagTail     Action = "wag_tail"     // happy tail wag
	ActionPerkEars    Action = "perk_ears"    // alert, listening
	ActionFlattenEars Action = "flatten_ears" // scared, submissive
	ActionTiltHead    Action = "tilt_head"    // curious, confused
	ActionCrouch      Action = "crouch"       // scared, defensive
	ActionBounce      Action = "bounce"       // excited hop
	ActionSpin        Action = "spin"         // happy spin
	ActionCurl        Action = "curl"         // sleepy curl up
	ActionPeek        Action = "peek"         // cautious look around
	ActionNuzzle      Action = "nuzzle"       // affectionate

	// Sound actions
	ActionWhimper Action = "whimper"  // scared sound
	ActionChirp   Action = "chirp"    // happy sound
	ActionBark    Action = "bark"     // alert/excited sound
	ActionGrowl   Action = "growl"    // warning sound
	ActionYawn    Action = "yawn"     // sleepy sound
	ActionPurr    Action = "purr"     // content sound
	ActionHeadBob Action = "head_bob" // bobbing to music
)

// ActionSet is a collection of actions that can be performed together.
type ActionSet struct {
	Movement   Action
	Expression Action
	Sound      Action
}

// moodActions maps moods to their available/typical action sets.
// This is what gets passed to the LLM as the vocabulary to choose from.
var moodActions = map[Mood][]Action{
	MoodCurious: {
		ActionExplore, ActionApproach, ActionStay,
		ActionPerkEars, ActionTiltHead,
		ActionChirp,
	},
	MoodExcited: {
		ActionApproach, ActionExplore, ActionSpin,
		ActionWagTail, ActionBounce, ActionPerkEars,
		ActionChirp, ActionBark,
	},
	MoodHappy: {
		ActionStay, ActionApproach, ActionExplore,
		ActionWagTail, ActionNuzzle, ActionHeadBob,
		ActionChirp, ActionPurr,
	},
	MoodStartled: {
		ActionFreeze, ActionRetreat, ActionFlee,
		ActionPerkEars, ActionCrouch,
		ActionWhimper,
	},
	MoodFrightened: {
		ActionFlee, ActionRetreat, ActionFreeze,
		ActionFlattenEars, ActionCrouch, ActionPeek,
		ActionWhimper,
	},
	MoodCautious: {
		ActionFreeze, ActionRetreat, ActionStay, ActionPeek,
		ActionPerkEars, ActionFlattenEars,
		ActionGrowl, ActionWhimper,
	},
	MoodSleepy: {
		ActionStay, ActionCurl,
		ActionYawn,
		ActionPurr,
	},
}

// AvailableActions returns the actions appropriate for the current mood.
func (e *EmotionalState) AvailableActions() []Action {
	actions, ok := moodActions[e.CurrentMood]
	if !ok {
		// Fallback to curious actions
		return moodActions[MoodCurious]
	}
	return actions
}

// SuggestDefaultAction returns a reasonable default action for the current mood.
// Used when LLM isn't available or for immediate reactions.
func (e *EmotionalState) SuggestDefaultAction() ActionSet {
	switch e.CurrentMood {
	case MoodCurious:
		return ActionSet{ActionExplore, ActionPerkEars, ActionChirp}
	case MoodExcited:
		return ActionSet{ActionApproach, ActionWagTail, ActionChirp}
	case MoodHappy:
		return ActionSet{ActionStay, ActionWagTail, ActionPurr}
	case MoodStartled:
		return ActionSet{ActionFreeze, ActionPerkEars, ActionWhimper}
	case MoodFrightened:
		return ActionSet{ActionFlee, ActionFlattenEars, ActionWhimper}
	case MoodCautious:
		return ActionSet{ActionFreeze, ActionPerkEars, ActionGrowl}
	case MoodSleepy:
		return ActionSet{ActionCurl, ActionCurl, ActionYawn}
	default:
		return ActionSet{ActionStay, ActionPerkEars, ActionChirp}
	}
}
