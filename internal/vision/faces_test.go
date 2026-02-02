package vision

import (
	"math"
	"path/filepath"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Embedding
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        Embedding{1, 0, 0},
			b:        Embedding{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        Embedding{1, 0, 0},
			b:        Embedding{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        Embedding{1, 0, 0},
			b:        Embedding{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        Embedding{1, 2, 3},
			b:        Embedding{1, 2, 3.1},
			expected: 0.9998, // approximately
		},
		{
			name:     "different lengths returns 0",
			a:        Embedding{1, 2},
			b:        Embedding{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.expected) > 0.01 {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFaceDB_EnrollAndRecognize(t *testing.T) {
	// Use temp file for database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "faces.json")

	db, err := NewFaceDB(dbPath)
	if err != nil {
		t.Fatalf("NewFaceDB() error = %v", err)
	}

	// Create some fake embeddings (128-dimensional)
	makeEmbedding := func(seed float64) Embedding {
		emb := make(Embedding, 128)
		for i := range emb {
			emb[i] = seed + float64(i)*0.01
		}
		return emb
	}

	ownerEmbeddings := []Embedding{
		makeEmbedding(1.0),
		makeEmbedding(1.01),
		makeEmbedding(1.02),
		makeEmbedding(0.99),
		makeEmbedding(1.005),
	}

	// Enroll owner
	owner, err := db.EnrollOwner("Alice", ownerEmbeddings)
	if err != nil {
		t.Fatalf("EnrollOwner() error = %v", err)
	}

	if owner.Name != "Alice" {
		t.Errorf("owner.Name = %v, want Alice", owner.Name)
	}

	if owner.Relationship != RelationshipOwner {
		t.Errorf("owner.Relationship = %v, want owner", owner.Relationship)
	}

	// Should recognize owner with similar embedding
	result := db.Recognize(makeEmbedding(1.0), EmotionHappy, 0.9)
	if result.Person == nil {
		t.Fatal("expected to recognize owner")
	}
	if !result.IsOwner {
		t.Error("expected IsOwner to be true")
	}
	if result.Person.Name != "Alice" {
		t.Errorf("recognized name = %v, want Alice", result.Person.Name)
	}

	// Should not recognize very different embedding
	// Make a truly different embedding (negative values, different pattern)
	strangerEmbedding := make(Embedding, 128)
	for i := range strangerEmbedding {
		strangerEmbedding[i] = -float64(i) * 0.1 // completely different pattern
	}
	result = db.Recognize(strangerEmbedding, EmotionNeutral, 0.5)
	if result.Person != nil {
		t.Errorf("expected not to recognize stranger, got %v with confidence %v", result.Person.Name, result.Confidence)
	}
}

func TestFaceDB_HasOwner(t *testing.T) {
	db, _ := NewFaceDB("")

	if db.HasOwner() {
		t.Error("expected no owner initially")
	}

	embeddings := make([]Embedding, 5)
	for i := range embeddings {
		embeddings[i] = make(Embedding, 128)
	}

	_, err := db.EnrollOwner("Test", embeddings)
	if err != nil {
		t.Fatalf("EnrollOwner() error = %v", err)
	}

	if !db.HasOwner() {
		t.Error("expected to have owner after enrollment")
	}

	// Should not allow second owner
	_, err = db.EnrollOwner("Test2", embeddings)
	if err == nil {
		t.Error("expected error when enrolling second owner")
	}
}

func TestFaceDB_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "faces.json")

	// Create and populate database
	db1, _ := NewFaceDB(dbPath)
	embeddings := make([]Embedding, 5)
	for i := range embeddings {
		embeddings[i] = make(Embedding, 128)
		for j := range embeddings[i] {
			embeddings[i][j] = float64(i + j)
		}
	}
	db1.EnrollOwner("Persisted", embeddings)

	// Load from disk
	db2, err := NewFaceDB(dbPath)
	if err != nil {
		t.Fatalf("loading database: %v", err)
	}

	owner := db2.GetOwner()
	if owner == nil {
		t.Fatal("expected to find owner in loaded database")
	}
	if owner.Name != "Persisted" {
		t.Errorf("owner.Name = %v, want Persisted", owner.Name)
	}
}

func TestFaceDB_InsufficientData(t *testing.T) {
	db, _ := NewFaceDB("")

	// Try to enroll with too few embeddings
	_, err := db.Enroll("Test", RelationshipFriend, []Embedding{{1, 2, 3}})
	if err != ErrInsufficientData {
		t.Errorf("expected ErrInsufficientData, got %v", err)
	}
}

func TestFaceDB_RemovePerson(t *testing.T) {
	db, _ := NewFaceDB("")

	embeddings := make([]Embedding, 5)
	for i := range embeddings {
		embeddings[i] = make(Embedding, 128)
	}

	person, _ := db.Enroll("ToRemove", RelationshipFriend, embeddings)

	if len(db.ListPeople()) != 1 {
		t.Error("expected 1 person")
	}

	err := db.RemovePerson(person.ID)
	if err != nil {
		t.Errorf("RemovePerson() error = %v", err)
	}

	if len(db.ListPeople()) != 0 {
		t.Error("expected 0 people after removal")
	}

	// Remove non-existent should error
	err = db.RemovePerson("nonexistent")
	if err != ErrPersonNotFound {
		t.Errorf("expected ErrPersonNotFound, got %v", err)
	}
}

func TestFaceDB_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nonexistent", "faces.json")

	// Should not error on non-existent file
	db, err := NewFaceDB(dbPath)
	if err != nil {
		t.Fatalf("NewFaceDB() should not error on missing file: %v", err)
	}

	if len(db.ListPeople()) != 0 {
		t.Error("expected empty database")
	}
}

func TestEmotions(t *testing.T) {
	// Just verify the emotion constants are defined properly
	emotions := []Emotion{
		EmotionNeutral,
		EmotionHappy,
		EmotionSad,
		EmotionAngry,
		EmotionSurprised,
		EmotionFearful,
		EmotionDisgusted,
	}

	for _, e := range emotions {
		if e == "" {
			t.Error("emotion should not be empty")
		}
	}
}

func TestRelationships(t *testing.T) {
	relationships := []Relationship{
		RelationshipOwner,
		RelationshipFamily,
		RelationshipFriend,
		RelationshipStranger,
	}

	for _, r := range relationships {
		if r == "" {
			t.Error("relationship should not be empty")
		}
	}
}

func TestFaceDB_BestSimilarity(t *testing.T) {
	db, _ := NewFaceDB("")

	refs := []Embedding{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}

	// Query that matches first reference
	query := Embedding{1, 0.1, 0}
	best := db.bestSimilarity(query, refs)

	// Should be close to 1 (matching first reference)
	if best < 0.9 {
		t.Errorf("bestSimilarity() = %v, expected > 0.9", best)
	}
}
