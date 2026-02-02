// Package vision handles face detection, recognition, and emotion analysis.
package vision

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Common errors.
var (
	ErrNoFaceDetected   = errors.New("no face detected in image")
	ErrMultipleFaces    = errors.New("multiple faces detected, expected one")
	ErrUnknownFace      = errors.New("face not recognized")
	ErrPersonExists     = errors.New("person already enrolled")
	ErrPersonNotFound   = errors.New("person not found")
	ErrInsufficientData = errors.New("insufficient enrollment data")
)

// Emotion represents a detected emotional state.
type Emotion string

const (
	EmotionNeutral   Emotion = "neutral"
	EmotionHappy     Emotion = "happy"
	EmotionSad       Emotion = "sad"
	EmotionAngry     Emotion = "angry"
	EmotionSurprised Emotion = "surprised"
	EmotionFearful   Emotion = "fearful"
	EmotionDisgusted Emotion = "disgusted"
)

// Relationship describes how Koji knows someone.
type Relationship string

const (
	RelationshipOwner    Relationship = "owner"    // primary owner
	RelationshipFamily   Relationship = "family"   // household members
	RelationshipFriend   Relationship = "friend"   // recognized visitors
	RelationshipStranger Relationship = "stranger" // unknown face
)

// Embedding is a face embedding vector.
// Typically 128 or 512 floats depending on the model.
type Embedding []float64

// Person represents a known individual.
type Person struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Relationship Relationship `json:"relationship"`
	Embeddings   []Embedding  `json:"embeddings"` // multiple for robustness
	EnrolledAt   time.Time    `json:"enrolled_at"`
	LastSeenAt   time.Time    `json:"last_seen_at"`
	SeenCount    int          `json:"seen_count"`
}

// FaceDetection represents a detected face in an image.
type FaceDetection struct {
	BoundingBox BoundingBox
	Confidence  float64
	Embedding   Embedding
	Emotion     Emotion
	EmotionConf float64 // confidence in emotion detection
}

// BoundingBox defines a rectangular region.
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RecognitionResult is returned when trying to identify a face.
type RecognitionResult struct {
	Person      *Person
	Confidence  float64 // 0.0 to 1.0, how sure we are
	Emotion     Emotion
	EmotionConf float64
	IsOwner     bool
}

// FaceDB stores known faces and handles recognition.
type FaceDB struct {
	mu       sync.RWMutex
	people   map[string]*Person
	dataPath string

	// Recognition thresholds
	matchThreshold float64 // cosine similarity threshold for match
	ownerThreshold float64 // stricter threshold for owner recognition
}

// NewFaceDB creates a new face database.
func NewFaceDB(dataPath string) (*FaceDB, error) {
	db := &FaceDB{
		people:         make(map[string]*Person),
		dataPath:       dataPath,
		matchThreshold: 0.6, // tune based on testing
		ownerThreshold: 0.7, // higher confidence for owner
	}

	// Try to load existing data
	if err := db.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading face database: %w", err)
	}

	return db, nil
}

// Enroll adds a new person to the database.
// Requires at least 3 embeddings for robustness.
func (db *FaceDB) Enroll(name string, relationship Relationship, embeddings []Embedding) (*Person, error) {
	if len(embeddings) < 3 {
		return nil, ErrInsufficientData
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if already enrolled
	for _, p := range db.people {
		if p.Name == name {
			return nil, ErrPersonExists
		}
	}

	person := &Person{
		ID:           generateID(),
		Name:         name,
		Relationship: relationship,
		Embeddings:   embeddings,
		EnrolledAt:   time.Now(),
		LastSeenAt:   time.Now(),
		SeenCount:    0,
	}

	db.people[person.ID] = person

	if err := db.save(); err != nil {
		delete(db.people, person.ID)
		return nil, fmt.Errorf("saving database: %w", err)
	}

	return person, nil
}

// EnrollOwner is a convenience method for enrolling the primary owner.
func (db *FaceDB) EnrollOwner(name string, embeddings []Embedding) (*Person, error) {
	// Check if owner already exists
	db.mu.RLock()
	for _, p := range db.people {
		if p.Relationship == RelationshipOwner {
			db.mu.RUnlock()
			return nil, fmt.Errorf("owner already enrolled: %s", p.Name)
		}
	}
	db.mu.RUnlock()

	return db.Enroll(name, RelationshipOwner, embeddings)
}

// Recognize tries to identify a face from its embedding.
func (db *FaceDB) Recognize(embedding Embedding, emotion Emotion, emotionConf float64) *RecognitionResult {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var bestMatch *Person
	var bestSimilarity float64

	for _, person := range db.people {
		similarity := db.bestSimilarity(embedding, person.Embeddings)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = person
		}
	}

	// Check if we have a confident enough match
	threshold := db.matchThreshold
	if bestMatch != nil && bestMatch.Relationship == RelationshipOwner {
		threshold = db.ownerThreshold
	}

	if bestSimilarity < threshold {
		// Unknown face
		return &RecognitionResult{
			Person:      nil,
			Confidence:  bestSimilarity,
			Emotion:     emotion,
			EmotionConf: emotionConf,
			IsOwner:     false,
		}
	}

	// Update last seen (in a goroutine to not block)
	go db.recordSighting(bestMatch.ID)

	return &RecognitionResult{
		Person:      bestMatch,
		Confidence:  bestSimilarity,
		Emotion:     emotion,
		EmotionConf: emotionConf,
		IsOwner:     bestMatch.Relationship == RelationshipOwner,
	}
}

// GetOwner returns the enrolled owner, if any.
func (db *FaceDB) GetOwner() *Person {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, p := range db.people {
		if p.Relationship == RelationshipOwner {
			return p
		}
	}
	return nil
}

// GetPerson returns a person by ID.
func (db *FaceDB) GetPerson(id string) *Person {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.people[id]
}

// ListPeople returns all enrolled people.
func (db *FaceDB) ListPeople() []*Person {
	db.mu.RLock()
	defer db.mu.RUnlock()

	people := make([]*Person, 0, len(db.people))
	for _, p := range db.people {
		people = append(people, p)
	}
	return people
}

// RemovePerson removes a person from the database.
func (db *FaceDB) RemovePerson(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.people[id]; !ok {
		return ErrPersonNotFound
	}

	delete(db.people, id)
	return db.save()
}

// HasOwner returns true if an owner has been enrolled.
func (db *FaceDB) HasOwner() bool {
	return db.GetOwner() != nil
}

// bestSimilarity finds the highest cosine similarity between an embedding
// and a set of reference embeddings.
func (db *FaceDB) bestSimilarity(query Embedding, references []Embedding) float64 {
	var best float64
	for _, ref := range references {
		sim := cosineSimilarity(query, ref)
		if sim > best {
			best = sim
		}
	}
	return best
}

// recordSighting updates the last seen time and count for a person.
func (db *FaceDB) recordSighting(id string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if p, ok := db.people[id]; ok {
		p.LastSeenAt = time.Now()
		p.SeenCount++
		_ = db.save() // best effort
	}
}

// save persists the database to disk.
func (db *FaceDB) save() error {
	if db.dataPath == "" {
		return nil // in-memory only
	}

	// Ensure directory exists
	dir := filepath.Dir(db.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(db.people, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(db.dataPath, data, 0644)
}

// load reads the database from disk.
func (db *FaceDB) load() error {
	if db.dataPath == "" {
		return nil
	}

	data, err := os.ReadFile(db.dataPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &db.people)
}

// cosineSimilarity computes the cosine similarity between two embeddings.
// Returns a value between -1 and 1, where 1 means identical.
func cosineSimilarity(a, b Embedding) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// generateID creates a simple unique ID.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
