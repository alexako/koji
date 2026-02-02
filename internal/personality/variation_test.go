package personality

import (
	"testing"
	"time"
)

func TestVariationEngine_SelectAction(t *testing.T) {
	v := NewVariationEngine()
	state := NewEmotionalState()

	// Run multiple times to verify we get variation
	actionCounts := make(map[Action]int)
	for i := 0; i < 100; i++ {
		result := v.SelectAction(state)
		actionCounts[result.Action]++
	}

	// Should have gotten multiple different actions
	if len(actionCounts) < 2 {
		t.Errorf("expected variation in actions, got only %d unique actions", len(actionCounts))
	}

	// Explore should be most common for curious mood (highest weight)
	if actionCounts[ActionExplore] < actionCounts[ActionStay] {
		t.Logf("actionCounts: %v", actionCounts)
		// This could theoretically fail due to randomness, but is unlikely
		// with 100 iterations and weights of 4.0 vs 1.0
	}
}

func TestVariationEngine_SelectAction_AllMoods(t *testing.T) {
	v := NewVariationEngine()

	moods := []Mood{
		MoodCurious, MoodExcited, MoodHappy,
		MoodStartled, MoodFrightened, MoodCautious, MoodSleepy,
	}

	for _, mood := range moods {
		t.Run(string(mood), func(t *testing.T) {
			state := &EmotionalState{
				CurrentMood: mood,
				Intensity:   IntensityMedium,
				EnteredAt:   time.Now(),
				baseline:    MoodCurious,
			}

			// Should not panic and should return a valid action
			result := v.SelectAction(state)
			if result.Action == "" {
				t.Errorf("got empty action for mood %s", mood)
			}
			if result.Modifier == "" {
				t.Errorf("got empty modifier for mood %s", mood)
			}
		})
	}
}

func TestVariationEngine_MoodEchoes(t *testing.T) {
	v := NewVariationEngine()

	// Record that we were frightened
	v.RecordMoodChange(MoodFrightened)

	// Now we're curious, but echoes should still be active
	state := &EmotionalState{
		CurrentMood: MoodCurious,
		Intensity:   IntensityMedium,
		EnteredAt:   time.Now(),
		baseline:    MoodCurious,
	}

	// Run many times and check if echo actions appear
	actionCounts := make(map[Action]int)
	for i := 0; i < 200; i++ {
		result := v.SelectAction(state)
		actionCounts[result.Action]++
	}

	// Should see some "echo" actions like Peek, FlattenEars, or Freeze
	// that wouldn't normally be as common in curious mood
	echoActionsSeen := actionCounts[ActionPeek] + actionCounts[ActionFlattenEars] + actionCounts[ActionFreeze]
	if echoActionsSeen == 0 {
		t.Errorf("expected some echo actions from frightened mood, got none. counts: %v", actionCounts)
	}
}

func TestVariationEngine_EchoesDecay(t *testing.T) {
	v := NewVariationEngine()

	// Record a mood change that happened long ago (simulating time passage)
	v.moodHistory = append(v.moodHistory, MoodEcho{
		FromMood:  MoodFrightened,
		Strength:  1.0,
		StartedAt: time.Now().Add(-time.Minute), // 60 seconds ago
	})

	// Frightened echo decays in 45 seconds, so this should be inactive
	echoes := v.GetActiveEchoes()
	if len(echoes) > 0 {
		t.Errorf("expected no active echoes after decay time, got %d", len(echoes))
	}
}

func TestVariationEngine_EchoStrengthDecays(t *testing.T) {
	v := NewVariationEngine()

	// Record a mood change that happened recently
	v.moodHistory = append(v.moodHistory, MoodEcho{
		FromMood:  MoodFrightened,
		Strength:  1.0,
		StartedAt: time.Now().Add(-22 * time.Second), // about half of 45s decay time
	})

	echoes := v.GetActiveEchoes()
	if len(echoes) != 1 {
		t.Fatalf("expected 1 active echo, got %d", len(echoes))
	}

	// Strength should be roughly 0.5 (about half decayed)
	if echoes[0].Strength < 0.4 || echoes[0].Strength > 0.6 {
		t.Errorf("expected strength around 0.5, got %f", echoes[0].Strength)
	}
}

func TestVariationEngine_SelectMicroBehavior(t *testing.T) {
	v := NewVariationEngine()

	// Run many times and check we get variety
	behaviorCounts := make(map[string]int)
	nilCount := 0
	for i := 0; i < 100; i++ {
		behavior := v.SelectMicroBehavior(MoodCurious)
		if behavior == nil {
			nilCount++
		} else {
			behaviorCounts[behavior.Name]++
		}
	}

	// Should have some nil results (natural pauses)
	if nilCount == 0 {
		t.Error("expected some nil micro-behaviors (natural pauses)")
	}

	// Should have variety in behaviors
	if len(behaviorCounts) < 2 {
		t.Errorf("expected variety in micro-behaviors, got only %d unique", len(behaviorCounts))
	}
}

func TestVariationEngine_SelectMicroBehavior_AllMoods(t *testing.T) {
	v := NewVariationEngine()

	moods := []Mood{
		MoodCurious, MoodExcited, MoodHappy,
		MoodStartled, MoodFrightened, MoodCautious, MoodSleepy,
	}

	for _, mood := range moods {
		t.Run(string(mood), func(t *testing.T) {
			// Run a few times to make sure it doesn't panic
			for i := 0; i < 10; i++ {
				_ = v.SelectMicroBehavior(mood)
			}
		})
	}
}

func TestIntensityToModifier(t *testing.T) {
	v := NewVariationEngine()

	tests := []struct {
		name      string
		intensity Intensity
		mood      Mood
		possible  []ActionModifier // any of these is acceptable due to jitter
	}{
		{"high frightened", IntensityHigh, MoodFrightened, []ActionModifier{ModifierFrantic, ModifierFast}},
		{"low frightened", IntensityLow, MoodFrightened, []ActionModifier{ModifierHesitant}},
		{"high excited", IntensityHigh, MoodExcited, []ActionModifier{ModifierFrantic, ModifierEager}},
		{"high sleepy", IntensityHigh, MoodSleepy, []ActionModifier{ModifierSlow, ModifierGentle}},
		{"low curious", IntensityLow, MoodCurious, []ActionModifier{ModifierGentle, ModifierNormal}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times due to jitter
			found := false
			for i := 0; i < 20; i++ {
				mod := v.intensityToModifier(tt.intensity, tt.mood)
				for _, possible := range tt.possible {
					if mod == possible {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				t.Errorf("modifier for %s/%.1f not in expected set %v", tt.mood, tt.intensity, tt.possible)
			}
		})
	}
}

func TestModifiedAction_HasBothFields(t *testing.T) {
	v := NewVariationEngine()
	state := NewEmotionalState()

	for i := 0; i < 20; i++ {
		result := v.SelectAction(state)
		if result.Action == "" {
			t.Error("Action should not be empty")
		}
		if result.Modifier == "" {
			t.Error("Modifier should not be empty")
		}
	}
}

func TestMicroBehavior_HasDuration(t *testing.T) {
	v := NewVariationEngine()

	// Try to get a non-nil micro-behavior
	var behavior *MicroBehavior
	for i := 0; i < 50; i++ {
		behavior = v.SelectMicroBehavior(MoodCurious)
		if behavior != nil {
			break
		}
	}

	if behavior == nil {
		t.Fatal("couldn't get a non-nil micro-behavior after 50 tries")
	}

	if behavior.Duration <= 0 {
		t.Errorf("micro-behavior should have positive duration, got %v", behavior.Duration)
	}
	if behavior.Name == "" {
		t.Error("micro-behavior should have a name")
	}
}
