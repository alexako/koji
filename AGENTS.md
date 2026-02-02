# AGENTS.md - Koji Development Guide

> Guidelines for AI agents and developers working on the Koji codebase.

## Project Overview

Koji is an AI-powered robot pet built primarily in **Go**, running on a Raspberry Pi. It uses local ML (TensorFlow Lite, OpenCV) for sensing and a local LLM for personality decisions.

**Core Philosophy:**
1. Don't die (safety first - always local, zero latency)
2. Don't be annoying (reliable, predictable behavior)
3. Be charming (personality and quirky reactions)

---

## Build & Run Commands

```bash
# Build the main binary
go build -o koji ./cmd/koji

# Run the application
./koji

# Run with verbose logging
./koji -v

# Build for Raspberry Pi (cross-compile from dev machine)
GOOS=linux GOARCH=arm64 go build -o koji ./cmd/koji
```

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
│   └── koji/           # Main entry point
│       └── main.go
├── internal/           # Private packages (not importable externally)
│   ├── personality/    # LLM integration, emotional state machine
│   ├── sensors/        # Camera, microphone, IR, ultrasonic
│   ├── actuators/      # Motors, servos, speaker
│   ├── safety/         # Safety controller (overrides all behavior)
│   └── config/         # Configuration loading
├── pkg/                # Public packages (if any)
├── configs/            # YAML/JSON configuration files
├── scripts/            # Build, deploy, dev scripts
├── testdata/           # Test fixtures
└── docs/               # Additional documentation
```

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

- **Never commit to main** - use feature branches
- **Branch naming**: `feature/add-cliff-sensors`, `fix/motor-timeout`, `refactor/sensor-interface`
- **Commit messages**: Use gitmoji + imperative mood
  - `:sparkles: Add emotional state machine`
  - `:bug: Fix cliff sensor false positives`
  - `:recycle: Refactor sensor interface for testability`

---

## Hardware Considerations

- **Raspberry Pi GPIO**: Use a library like `periph.io` for GPIO access
- **Cross-compile**: Build on dev machine, deploy to Pi
- **Sensor polling**: Keep it fast - safety sensors need <50ms response
- **Power management**: Monitor battery, graceful shutdown on low power

---

## Dependencies

Keep dependencies minimal. Prefer standard library when reasonable.

**Approved dependencies:**
- `periph.io` - GPIO/hardware access
- `gocv.io/x/gocv` - OpenCV bindings
- `github.com/charmbracelet/bubbletea` - TUI for debug interface
- `go.uber.org/zap` - structured logging

**For ML inference:** Shell out to Python or use TFLite Go bindings.

---

## JSON Schemas

All sensor input and action output uses JSON. Keep schemas in `configs/schemas/`.

```json
// Sensor event example
{
  "type": "sound",
  "subtype": "loud_noise",
  "amplitude": 0.85,
  "timestamp": "2026-02-01T10:30:00Z"
}

// Action output example  
{
  "actions": [
    {"type": "movement", "action": "stop"},
    {"type": "expression", "action": "crouch"},
    {"type": "sound", "action": "whimper"}
  ]
}
```
