package personality

import (
	"testing"
	"time"
)

func TestProcessEvent_LoudNoiseStartlesCurious(t *testing.T) {
	state := NewEmotionalState()

	changed := state.ProcessEvent(NewEventContext(EventLoudNoise))

	if !changed {
		t.Error("expected mood to change")
	}
	if state.CurrentMood != MoodStartled {
		t.Errorf("expected startled, got %s", state.CurrentMood)
	}
}

func TestProcessEvent_LoudNoiseEscalatesStartledToFrightened(t *testing.T) {
	state := NewEmotionalState()
	state.SetMood(MoodStartled, IntensityMedium)

	changed := state.ProcessEvent(NewEventContext(EventLoudNoise))

	if !changed {
		t.Error("expected mood to change")
	}
	if state.CurrentMood != MoodFrightened {
		t.Errorf("expected frightened, got %s", state.CurrentMood)
	}
}

func TestProcessEvent_MusicCalmsDown(t *testing.T) {
	state := NewEmotionalState()
	state.SetMood(MoodFrightened, IntensityHigh)

	changed := state.ProcessEvent(NewEventContext(EventMusic))

	if !changed {
		t.Error("expected mood to change")
	}
	if state.CurrentMood != MoodCautious {
		t.Errorf("expected cautious, got %s", state.CurrentMood)
	}
}

func TestProcessEvent_FamiliarFaceMakesHappy(t *testing.T) {
	state := NewEmotionalState()

	changed := state.ProcessEvent(NewEventContext(EventFamiliarFace))

	if !changed {
		t.Error("expected mood to change")
	}
	if state.CurrentMood != MoodHappy {
		t.Errorf("expected happy, got %s", state.CurrentMood)
	}
}

func TestProcessEvent_UnknownFaceMakesCautious(t *testing.T) {
	state := NewEmotionalState()

	changed := state.ProcessEvent(NewEventContext(EventUnknownFace))

	if !changed {
		t.Error("expected mood to change")
	}
	if state.CurrentMood != MoodCautious {
		t.Errorf("expected cautious, got %s", state.CurrentMood)
	}
}

func TestProcessEvent_PettingCalmsFrightened(t *testing.T) {
	state := NewEmotionalState()
	state.SetMood(MoodFrightened, IntensityHigh)

	changed := state.ProcessEvent(NewEventContext(EventPetted))

	if !changed {
		t.Error("expected mood to change")
	}
	if state.CurrentMood != MoodCautious {
		t.Errorf("expected cautious, got %s", state.CurrentMood)
	}
}

func TestProcessEvent_UnknownEventNoChange(t *testing.T) {
	state := NewEmotionalState()
	originalMood := state.CurrentMood

	changed := state.ProcessEvent(NewEventContext("unknown_event"))

	if changed {
		t.Error("expected no mood change for unknown event")
	}
	if state.CurrentMood != originalMood {
		t.Errorf("mood changed unexpectedly from %s to %s", originalMood, state.CurrentMood)
	}
}

func TestProcessEvent_IntensityAffectsResult(t *testing.T) {
	state := NewEmotionalState()

	ctx := NewEventContext(EventLoudNoise).WithIntensity(0.9)
	state.ProcessEvent(ctx)

	if state.Intensity != IntensityHigh {
		t.Errorf("expected high intensity for loud event, got %f", state.Intensity)
	}
}

func TestDecay_FrightenedToCautious(t *testing.T) {
	state := NewEmotionalState()
	state.CurrentMood = MoodFrightened
	state.Intensity = IntensityHigh
	state.EnteredAt = time.Now().Add(-16 * time.Second) // past decay time

	changed := state.Decay()

	if !changed {
		t.Error("expected mood to decay")
	}
	if state.CurrentMood != MoodCautious {
		t.Errorf("expected cautious, got %s", state.CurrentMood)
	}
}

func TestDecay_NoDecayBeforeTime(t *testing.T) {
	state := NewEmotionalState()
	state.CurrentMood = MoodFrightened
	state.EnteredAt = time.Now() // just entered, shouldn't decay yet

	changed := state.Decay()

	if changed {
		t.Error("expected no decay before time threshold")
	}
	if state.CurrentMood != MoodFrightened {
		t.Errorf("mood changed unexpectedly to %s", state.CurrentMood)
	}
}

func TestDecay_BaselineNoDecay(t *testing.T) {
	state := NewEmotionalState()
	state.EnteredAt = time.Now().Add(-5 * time.Minute) // long time

	changed := state.Decay()

	if changed {
		t.Error("expected no decay at baseline")
	}
	if state.CurrentMood != MoodCurious {
		t.Errorf("baseline mood changed unexpectedly to %s", state.CurrentMood)
	}
}

func TestDecay_FullPathToBaseline(t *testing.T) {
	state := NewEmotionalState()
	state.SetMood(MoodFrightened, IntensityHigh)

	// Simulate time passing and decay steps
	// Frightened -> Cautious -> Curious
	path := []Mood{MoodFrightened, MoodCautious, MoodCurious}

	for i, expectedMood := range path {
		if state.CurrentMood != expectedMood {
			t.Errorf("step %d: expected %s, got %s", i, expectedMood, state.CurrentMood)
		}
		if state.IsBaseline() {
			break
		}
		// Force time to pass
		state.EnteredAt = time.Now().Add(-1 * time.Minute)
		state.Decay()
	}

	if state.CurrentMood != MoodCurious {
		t.Errorf("expected to reach curious baseline, got %s", state.CurrentMood)
	}
}

// Table-driven test for common scenarios
func TestProcessEvent_Scenarios(t *testing.T) {
	tests := []struct {
		name         string
		initialMood  Mood
		event        Event
		expectedMood Mood
		shouldChange bool
	}{
		{"curious + loud noise = startled", MoodCurious, EventLoudNoise, MoodStartled, true},
		{"sleepy + loud noise = frightened", MoodSleepy, EventLoudNoise, MoodFrightened, true},
		{"happy + familiar face = excited", MoodHappy, EventFamiliarFace, MoodExcited, true},
		{"curious + music = happy", MoodCurious, EventMusic, MoodHappy, true},
		{"happy + rhythm = excited", MoodHappy, EventRhythm, MoodExcited, true},
		{"frightened + petted = cautious", MoodFrightened, EventPetted, MoodCautious, true},
		{"sleepy + poked = startled", MoodSleepy, EventPoked, MoodStartled, true},
		{"curious + silence = sleepy", MoodCurious, EventSilence, MoodSleepy, true},
		// Unknown object curiosity
		{"curious + unknown object = excited", MoodCurious, EventUnknownObject, MoodExcited, true},
		{"sleepy + unknown object = curious", MoodSleepy, EventUnknownObject, MoodCurious, true},
		{"frightened + unknown object = cautious", MoodFrightened, EventUnknownObject, MoodCautious, true},
		{"cautious + unknown object = curious", MoodCautious, EventUnknownObject, MoodCurious, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewEmotionalState()
			state.SetMood(tt.initialMood, IntensityMedium)

			changed := state.ProcessEvent(NewEventContext(tt.event))

			if changed != tt.shouldChange {
				t.Errorf("expected changed=%v, got %v", tt.shouldChange, changed)
			}
			if state.CurrentMood != tt.expectedMood {
				t.Errorf("expected %s, got %s", tt.expectedMood, state.CurrentMood)
			}
		})
	}
}
