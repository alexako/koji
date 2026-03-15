# AGENTS.md - Koji Development Guide

> Guidelines for AI agents and developers working on the Koji codebase.

## Project Overview

Koji is an AI-powered robot pet with a distributed architecture:
- **Brain Server** (Go) - Runs on a home server, handles emotional state and LLM decisions
- **Body** (Go) - Runs on Raspberry Pi, handles sensors and actuators
- **Face Display** (C++/Arduino) - Runs on ESP32 with GC9A01 round TFT, animated eyes

**Core Philosophy:**
1. Don't die (safety first - always local, zero latency)
2. Don't be annoying (reliable, predictable behavior)
3. Be charming (personality and quirky reactions)

---

## Architecture

```
┌─────────────────┐                    ┌─────────────────┐
│   Brain Server  │     HTTP/JSON      │     ESP32       │
│   (theserver)   │◄──────────────────►│  (face display) │
│   Port 8585     │   GET /api/state   │  192.168.1.194  │
│                 │   (poll 500ms)     └─────────────────┘
│  - Mood state   │     
│  - Event proc   │     HTTP/JSON      ┌─────────────────┐
│  - Decay loop   │◄──────────────────►│  Sensors (TBD)  │
│  - Face emotion │   POST /api/event  │  (desktop/Pi)   │
└─────────────────┘                    └─────────────────┘
        │
        │  Gitea Actions (auto-deploy on push to main)
        ▼
┌─────────────────┐
│   Docker        │
│   koji-brain    │
└─────────────────┘
```

**Current Deployment:**
- Brain server runs on `theserver` at `http://192.168.1.41:8585`
- ESP32 face display polls `/api/state` every 500ms
- CI/CD via Gitea Actions auto-deploys on push to main

**API Endpoints (Brain Server):**
- `GET /api/state` - Current mood, intensity, face emotion, emotion_index
- `POST /api/event` - Send sensor events, returns full state for immediate reaction
- `GET /health` - Health check

---

## Build & Run Commands

### Brain Server (runs on theserver)

```bash
# Build the brain server
go build -o brain ./cmd/brain

# Run the brain server
./brain -addr :8080

# Build for Linux server (cross-compile)
GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain
```

### CLI Simulator (for testing)

```bash
# Build the CLI simulator
go build -o koji ./cmd/koji

# Run with LLM support
./koji -ollama http://localhost:11434 -model phi3:mini

# Run without LLM (variation engine only)
./koji -no-llm
```

### ESP32 Face Display

```bash
# Build and upload (from esp32-face/ directory)
cd esp32-face
pio run -t upload

# Monitor serial output
pio device monitor
```

**Setup:**
1. Copy `src/config.h` to `src/config_local.h`
2. Edit WiFi credentials and brain server URL
3. Flash with PlatformIO

---

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test by name
go test -v -run TestEmotionalStateMachine ./internal/personality

# Run tests in a specific package
go test -v ./internal/sensors/...

# Run tests with race detection (slow but catches concurrency bugs)
go test -race ./...

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Linting & Formatting

```bash
# Format all Go code
gofmt -w .

# Better: use goimports (also organizes imports)
goimports -w .

# Run linter (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
golangci-lint run

# Fix auto-fixable lint issues
golangci-lint run --fix
```

---

## Project Structure

```
koji/
├── cmd/
│   ├── koji/           # CLI simulator for testing
│   │   └── main.go
│   └── brain/          # Brain server (runs on theserver)
│       └── main.go
├── internal/           # Private packages (not importable externally)
│   ├── api/            # HTTP API server
│   ├── brain/          # Brain orchestrator (decay loop, event handling)
│   ├── personality/    # Emotional state machine, mood transitions
│   │   ├── mood.go         # Mood types and EmotionalState
│   │   ├── emotions.go     # Mood-to-face-emotion mapping
│   │   ├── events.go       # Sensor event types
│   │   ├── transitions.go  # Event→mood transition rules
│   │   ├── actions.go      # Available actions per mood
│   │   └── variation.go    # Lifelike variation engine
│   ├── llm/            # LLM client and personality engine
│   ├── vision/         # Face recognition, enrollment
│   ├── sensors/        # Camera, microphone, IR, ultrasonic
│   ├── actuators/      # Motors, servos, speaker
│   ├── safety/         # Safety controller (overrides all behavior)
│   └── config/         # Configuration loading
├── esp32-face/         # ESP32 face display (PlatformIO project)
│   ├── platformio.ini
│   └── src/
│       ├── main.cpp        # Main loop, WiFi, brain API client
│       ├── config.h        # Default config (copy to config_local.h)
│       ├── Face.cpp/h      # Face rendering
│       ├── Eye.cpp/h       # Eye rendering
│       ├── FaceEmotions.hpp    # Emotion enum (18 emotions)
│       ├── FaceBehavior.cpp/h  # Emotion transitions
│       └── ...             # Animation helpers
├── configs/            # YAML/JSON configuration files
├── scripts/            # Build, deploy, dev scripts
└── docs/               # Additional documentation
```

---

## Emotional State System

### Moods (internal)
7 moods with intensity (0.0-1.0):
- `curious` - baseline state
- `excited` - new person, play time
- `startled` - sudden stimulus, brief
- `frightened` - escalated fear
- `happy` - music, familiar faces
- `sleepy` - quiet environment
- `cautious` - wary, recovering from fear

### Face Emotions (ESP32)
18 emotions mapped from moods + intensity:
- Normal, Happy, Glee, Sad, Worried, Focused, Annoyed, Surprised
- Skeptic, Frustrated, Unimpressed, Sleepy, Suspicious, Squint
- Angry, Furious, Scared, Awe

### Event Types
```go
// Sound events
EventLoudNoise, EventMusic, EventSpeech, EventSilence, EventRhythm

// Vision events
EventFamiliarFace, EventUnknownFace, EventMotionDetected, EventUnknownObject

// Physical events
EventPetted, EventPoked, EventPickedUp

// Time-based events
EventTimePassedShort, EventTimePassedMedium, EventTimePassedLong
```

### Mood Decay
Moods decay in a cycle over time:
- Startled → Cautious (5s) → Curious (20s)
- Frightened → Cautious (15s) → Curious (20s)
- Excited → Happy (30s) → Curious (45s)
- Happy → Curious (45s)
- Curious → Sleepy (1 hour) → Curious (3 hours) → repeat

**Idle cycle:** When left alone, Koji will be curious for about an hour, then sleep for about 3 hours, then wake up curious again. Events interrupt this cycle and moods decay back to curious before the cycle resumes.

---

## Code Style Guidelines

### Naming Conventions

```go
// Packages: short, lowercase, no underscores
package sensors  // good
package ir_sensors  // bad

// Interfaces: describe behavior, often end in -er
type SensorReader interface { ... }
type Actuator interface { ... }

// Structs: noun, PascalCase
type EmotionalState struct { ... }
type CliffSensor struct { ... }

// Functions/Methods: verb or verb phrase, PascalCase for exported
func (s *CliffSensor) Read() (float64, error) { ... }
func parseConfig(path string) (*Config, error) { ... }

// Constants: PascalCase for exported, camelCase for unexported
const MaxSensorRetries = 3
const defaultTimeout = 5 * time.Second

// Avoid stuttering
sensors.NewSensor()     // bad - sensors.Sensor or sensors.New
sensors.New()           // good
sensors.NewCliff()      // good
```

### Import Organization

Group imports in this order, separated by blank lines:

```go
import (
    // Standard library
    "context"
    "fmt"
    "time"

    // Third-party packages
    "github.com/charmbracelet/bubbletea"
    "go.uber.org/zap"

    // Internal packages
    "github.com/alex/koji/internal/personality"
    "github.com/alex/koji/internal/sensors"
)
```

### Error Handling

```go
// Always handle errors explicitly - never ignore them
result, err := sensor.Read()
if err != nil {
    return fmt.Errorf("reading cliff sensor: %w", err)  // wrap with context
}

// Use sentinel errors for expected conditions
var ErrSensorTimeout = errors.New("sensor timeout")

// Check with errors.Is for wrapped errors
if errors.Is(err, ErrSensorTimeout) {
    // handle timeout specifically
}

// Safety-critical code: fail safe, not silent
if err != nil {
    s.emergencyStop()  // stop motors first, ask questions later
    return err
}
```

### Concurrency Patterns

```go
// Use context for cancellation and timeouts
func (s *Sensor) Read(ctx context.Context) (float64, error) {
    select {
    case <-ctx.Done():
        return 0, ctx.Err()
    case result := <-s.readChan:
        return result, nil
    }
}

// Protect shared state with mutexes
type EmotionalState struct {
    mu   sync.RWMutex
    mood Mood
}

func (e *EmotionalState) GetMood() Mood {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.mood
}

// Prefer channels for communication between goroutines
// Prefer mutexes for protecting shared data structures
```

### Comments and Documentation

```go
// Package personality implements Koji's emotional state machine and
// LLM-based decision making.
package personality

// EmotionalState tracks Koji's current mood and how it changes over time.
// Mood decays naturally toward a baseline "curious" state.
type EmotionalState struct { ... }

// UpdateMood adjusts the current mood based on a sensor event.
// It returns the new mood and any actions that should be triggered.
func (e *EmotionalState) UpdateMood(event SensorEvent) (Mood, []Action) { ... }
```

---

## Safety-Critical Code

The `internal/safety` package has special rules:

1. **No external dependencies** - only standard library
2. **No blocking operations** - must respond in <10ms
3. **Always fail safe** - if in doubt, stop all motors
4. **100% test coverage required**
5. **No panics** - recover and stop motors instead

```go
// Safety controller ALWAYS has priority
type SafetyController struct { ... }

// MustStop returns true if any safety condition is violated.
// This function must be non-blocking and infallible.
func (s *SafetyController) MustStop() bool { ... }
```

---

## Testing Conventions

```go
// Table-driven tests are the Go standard
func TestEmotionalState_UpdateMood(t *testing.T) {
    tests := []struct {
        name     string
        initial  Mood
        event    SensorEvent
        expected Mood
    }{
        {"loud noise startles", MoodCurious, EventLoudNoise, MoodStartled},
        {"music makes happy", MoodCurious, EventMusicDetected, MoodHappy},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            state := NewEmotionalState(tt.initial)
            got, _ := state.UpdateMood(tt.event)
            if got != tt.expected {
                t.Errorf("got %v, want %v", got, tt.expected)
            }
        })
    }
}

// Use testdata/ directory for fixtures
// Files in testdata/ are ignored by go build
```

---

## Git Workflow

- **Commit messages**: Use gitmoji + imperative mood
  - `:sparkles: Add emotional state machine`
  - `:bug: Fix cliff sensor false positives`
  - `:recycle: Refactor sensor interface for testability`

---

## Hardware Considerations

### ESP32 Face Display
- **Board**: Seeed XIAO ESP32-S3 (8MB PSRAM, 8MB Flash)
- **Display**: GC9A01 240x240 round TFT
- **Wiring (XIAO ESP32-S3)**:
  | Display Pin | XIAO Pin | GPIO |
  |-------------|----------|------|
  | SDA (MOSI)  | D9       | 8    |
  | SCL (SCLK)  | D8       | 7    |
  | DC          | D1       | 2    |
  | CS          | D2       | 3    |
  | RST         | D3       | 4    |
  | VCC         | 3V3      | -    |
  | GND         | GND      | -    |
- **Double buffering**: Enabled via PSRAM - no more flicker!
- **Build command**: `pio run -e xiao_esp32s3 -t upload`

### Legacy ESP32 WROOM (deprecated)
- **Wiring**: MOSI=23, SCLK=18, CS=5, DC=2, RST=4
- **Known issue**: Screen flicker due to insufficient RAM for framebuffer (has 114KB, needs 115KB)
- **Build command**: `pio run -e esp32dev -t upload`

### Raspberry Pi
- **GPIO**: Use a library like `periph.io` for GPIO access
- **Cross-compile**: Build on dev machine, deploy to Pi
- **Sensor polling**: Keep it fast - safety sensors need <50ms response
- **Power management**: Monitor battery, graceful shutdown on low power

---

## Dependencies

Keep dependencies minimal. Prefer standard library when reasonable.

**Go (Brain/Body):**
- `periph.io` - GPIO/hardware access
- `gocv.io/x/gocv` - OpenCV bindings
- `github.com/charmbracelet/bubbletea` - TUI for debug interface
- `go.uber.org/zap` - structured logging

**ESP32 (Face Display):**
- `TFT_eSPI` - Display driver
- `ArduinoJson` - JSON parsing for API responses

**For ML inference:** Shell out to Python or use TFLite Go bindings.

---

## API Reference

### GET /api/state
Returns current emotional state.

```json
{
  "mood": "startled",
  "intensity": 0.9,
  "duration_ms": 1234,
  "face_emotion": "scared",
  "emotion_index": 16,
  "action": "freeze",
  "action_age_ms": 500
}
```

### POST /api/event
Send a sensor event to the brain.

**Request:**
```json
{
  "event": "loud_noise",
  "intensity": 0.9,
  "source": "microphone",
  "metadata": {}
}
```

**Response:**
```json
{
  "accepted": true,
  "mood_changed": true,
  "mood": "startled",
  "intensity": 0.9,
  "face_emotion": "scared",
  "emotion_index": 16
}
```

### GET /health
Returns `ok` if server is running.

---

## CI/CD (Gitea Actions)

The brain server auto-deploys on every push to main:

1. Push to `main` branch triggers workflow
2. Gitea Actions runner (on theserver) builds Docker image
3. Container restarts with new code

**Workflow file:** `.gitea/workflows/deploy.yml`

**Manual event testing:**
```bash
# Scare Koji
curl -X POST http://192.168.1.41:8585/api/event \
  -H "Content-Type: application/json" \
  -d '{"event": "loud_noise", "intensity": 0.9}'

# Make Koji happy
curl -X POST http://192.168.1.41:8585/api/event \
  -H "Content-Type: application/json" \
  -d '{"event": "music", "intensity": 0.8}'

# Pet Koji
curl -X POST http://192.168.1.41:8585/api/event \
  -H "Content-Type: application/json" \
  -d '{"event": "petted", "intensity": 0.7}'
```
