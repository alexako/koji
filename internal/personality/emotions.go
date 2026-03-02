package personality

// FaceEmotion represents the emotions supported by the ESP32 face display.
// These map to the eEmotions enum in FaceEmotions.hpp
type FaceEmotion string

const (
	FaceNormal      FaceEmotion = "normal"
	FaceAngry       FaceEmotion = "angry"
	FaceGlee        FaceEmotion = "glee"
	FaceHappy       FaceEmotion = "happy"
	FaceSad         FaceEmotion = "sad"
	FaceWorried     FaceEmotion = "worried"
	FaceFocused     FaceEmotion = "focused"
	FaceAnnoyed     FaceEmotion = "annoyed"
	FaceSurprised   FaceEmotion = "surprised"
	FaceSkeptic     FaceEmotion = "skeptic"
	FaceFrustrated  FaceEmotion = "frustrated"
	FaceUnimpressed FaceEmotion = "unimpressed"
	FaceSleepy      FaceEmotion = "sleepy"
	FaceSuspicious  FaceEmotion = "suspicious"
	FaceSquint      FaceEmotion = "squint"
	FaceFurious     FaceEmotion = "furious"
	FaceScared      FaceEmotion = "scared"
	FaceAwe         FaceEmotion = "awe"
)

// moodToFaceEmotion maps Koji's internal moods to ESP32 face emotions.
// Some moods map to different emotions based on intensity.
var moodToFaceEmotion = map[Mood]map[Intensity]FaceEmotion{
	MoodCurious: {
		IntensityLow:    FaceNormal,
		IntensityMedium: FaceNormal,
		IntensityHigh:   FaceFocused,
	},
	MoodExcited: {
		IntensityLow:    FaceHappy,
		IntensityMedium: FaceGlee,
		IntensityHigh:   FaceAwe,
	},
	MoodHappy: {
		IntensityLow:    FaceHappy,
		IntensityMedium: FaceHappy,
		IntensityHigh:   FaceGlee,
	},
	MoodStartled: {
		IntensityLow:    FaceSurprised,
		IntensityMedium: FaceSurprised,
		IntensityHigh:   FaceScared,
	},
	MoodFrightened: {
		IntensityLow:    FaceWorried,
		IntensityMedium: FaceScared,
		IntensityHigh:   FaceFurious, // wide-eyed terror
	},
	MoodCautious: {
		IntensityLow:    FaceSkeptic,
		IntensityMedium: FaceSuspicious,
		IntensityHigh:   FaceSquint,
	},
	MoodSleepy: {
		IntensityLow:    FaceUnimpressed,
		IntensityMedium: FaceSleepy,
		IntensityHigh:   FaceSleepy,
	},
}

// ToFaceEmotion converts the current emotional state to an ESP32 face emotion.
func (e *EmotionalState) ToFaceEmotion() FaceEmotion {
	intensityMap, ok := moodToFaceEmotion[e.CurrentMood]
	if !ok {
		return FaceNormal
	}

	// Find the closest intensity bracket
	var emotion FaceEmotion
	switch {
	case e.Intensity <= 0.4:
		emotion = intensityMap[IntensityLow]
	case e.Intensity <= 0.7:
		emotion = intensityMap[IntensityMedium]
	default:
		emotion = intensityMap[IntensityHigh]
	}

	if emotion == "" {
		return FaceNormal
	}
	return emotion
}

// FaceEmotionIndex returns the numeric index for the ESP32 eEmotions enum.
// This matches the order in FaceEmotions.hpp
func (fe FaceEmotion) Index() int {
	switch fe {
	case FaceNormal:
		return 0
	case FaceAngry:
		return 1
	case FaceGlee:
		return 2
	case FaceHappy:
		return 3
	case FaceSad:
		return 4
	case FaceWorried:
		return 5
	case FaceFocused:
		return 6
	case FaceAnnoyed:
		return 7
	case FaceSurprised:
		return 8
	case FaceSkeptic:
		return 9
	case FaceFrustrated:
		return 10
	case FaceUnimpressed:
		return 11
	case FaceSleepy:
		return 12
	case FaceSuspicious:
		return 13
	case FaceSquint:
		return 14
	case FaceFurious:
		return 15
	case FaceScared:
		return 16
	case FaceAwe:
		return 17
	default:
		return 0
	}
}
