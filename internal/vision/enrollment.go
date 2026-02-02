package vision

import (
	"context"
	"fmt"
	"time"
)

// EnrollmentSession manages the face enrollment process.
type EnrollmentSession struct {
	detector     FaceDetector
	db           *FaceDB
	name         string
	relationship Relationship
	embeddings   []Embedding
	minSamples   int
	maxSamples   int
}

// FaceDetector is the interface for face detection backends.
// This will be implemented by the actual ML backend (OpenCV, MediaPipe, etc.)
type FaceDetector interface {
	// DetectFaces finds faces in an image and returns their details.
	DetectFaces(ctx context.Context, image []byte) ([]FaceDetection, error)

	// ExtractEmbedding gets the face embedding from a cropped face image.
	// Most detectors include this in DetectFaces, but some separate it.
	ExtractEmbedding(ctx context.Context, faceImage []byte) (Embedding, error)

	// DetectEmotion analyzes a face for emotional state.
	DetectEmotion(ctx context.Context, faceImage []byte) (Emotion, float64, error)
}

// EnrollmentStatus tracks progress of an enrollment session.
type EnrollmentStatus struct {
	SamplesCollected int
	SamplesNeeded    int
	IsComplete       bool
	Message          string
}

// NewEnrollmentSession creates a new enrollment session.
func NewEnrollmentSession(detector FaceDetector, db *FaceDB, name string, relationship Relationship) *EnrollmentSession {
	return &EnrollmentSession{
		detector:     detector,
		db:           db,
		name:         name,
		relationship: relationship,
		embeddings:   make([]Embedding, 0, 10),
		minSamples:   5,  // need at least 5 good samples
		maxSamples:   10, // stop after 10
	}
}

// AddFrame processes a camera frame and extracts face data if suitable.
// Returns the current enrollment status.
func (s *EnrollmentSession) AddFrame(ctx context.Context, imageData []byte) (*EnrollmentStatus, error) {
	faces, err := s.detector.DetectFaces(ctx, imageData)
	if err != nil {
		return nil, fmt.Errorf("detecting faces: %w", err)
	}

	status := &EnrollmentStatus{
		SamplesCollected: len(s.embeddings),
		SamplesNeeded:    s.minSamples,
		IsComplete:       false,
	}

	if len(faces) == 0 {
		status.Message = "No face detected. Please look at the camera."
		return status, nil
	}

	if len(faces) > 1 {
		status.Message = "Multiple faces detected. Please ensure only one person is visible."
		return status, nil
	}

	face := faces[0]

	// Check face quality
	if face.Confidence < 0.8 {
		status.Message = "Face not clear enough. Please move closer or improve lighting."
		return status, nil
	}

	// Check embedding is valid
	if len(face.Embedding) == 0 {
		status.Message = "Could not extract face features. Please try again."
		return status, nil
	}

	// Check this embedding is different enough from existing ones
	// (we want variety - different angles, expressions)
	if len(s.embeddings) > 0 && s.isTooSimilar(face.Embedding) {
		status.Message = "Got it! Now try a slightly different angle or expression."
		return status, nil
	}

	// Add the embedding
	s.embeddings = append(s.embeddings, face.Embedding)
	status.SamplesCollected = len(s.embeddings)

	if len(s.embeddings) >= s.maxSamples {
		status.IsComplete = true
		status.Message = "Enrollment complete!"
	} else if len(s.embeddings) >= s.minSamples {
		status.Message = fmt.Sprintf("Good! %d samples collected. You can finish or continue for better accuracy.", len(s.embeddings))
	} else {
		remaining := s.minSamples - len(s.embeddings)
		status.Message = fmt.Sprintf("Great! Need %d more samples. Try different angles.", remaining)
	}

	return status, nil
}

// isTooSimilar checks if an embedding is too similar to existing ones.
// We want diverse samples for robust recognition.
func (s *EnrollmentSession) isTooSimilar(embedding Embedding) bool {
	const similarityThreshold = 0.95 // very similar

	for _, existing := range s.embeddings {
		if cosineSimilarity(embedding, existing) > similarityThreshold {
			return true
		}
	}
	return false
}

// CanFinish returns true if we have enough samples to complete enrollment.
func (s *EnrollmentSession) CanFinish() bool {
	return len(s.embeddings) >= s.minSamples
}

// Finish completes the enrollment and saves to the database.
func (s *EnrollmentSession) Finish() (*Person, error) {
	if !s.CanFinish() {
		return nil, ErrInsufficientData
	}

	return s.db.Enroll(s.name, s.relationship, s.embeddings)
}

// Cancel aborts the enrollment session.
func (s *EnrollmentSession) Cancel() {
	s.embeddings = nil
}

// QuickEnroll is a helper for simple enrollment scenarios.
// It captures frames for the specified duration and enrolls the person.
func QuickEnroll(ctx context.Context, detector FaceDetector, db *FaceDB, name string, relationship Relationship, frameSource <-chan []byte, timeout time.Duration) (*Person, error) {
	session := NewEnrollmentSession(detector, db, name, relationship)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond) // 5 fps sampling
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if session.CanFinish() {
				return session.Finish()
			}
			return nil, fmt.Errorf("enrollment timeout: only got %d samples", len(session.embeddings))

		case frame, ok := <-frameSource:
			if !ok {
				if session.CanFinish() {
					return session.Finish()
				}
				return nil, fmt.Errorf("frame source closed: only got %d samples", len(session.embeddings))
			}

			status, err := session.AddFrame(ctx, frame)
			if err != nil {
				continue // skip bad frames
			}

			if status.IsComplete {
				return session.Finish()
			}

		case <-ticker.C:
			// Just a pacing mechanism
		}
	}
}
