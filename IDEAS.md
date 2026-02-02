# Koji - Ideas & Brain Dump

A curious little robot dog with personality.

---

## Core Concept

An AI-powered robot pet that feels alive. Not a voice assistant in a dog costume — an actual pet with moods, reactions, and quirks.

---

## Personality Traits

- Curious and easily excited by new things
- Startled by loud noises (hides, peeks out cautiously)
- Loves music (bobs head, wags tail)
- Gets sleepy when quiet
- A little clumsy but enthusiastic — but not stupid (see "Safety vs Emotion Model")
- Recognizes familiar faces (excited to see owner)
- Wary of strangers at first

**The "Charming Clumsy" Rule:** Koji can bonk into something unexpected once (cute), but never the same thing twice (dumb). Real pets have spatial memory and survival instincts even when panicked.

---

## Sensor Ideas

### Sound
- [ ] Volume/amplitude detection (loud = scary)
- [ ] Frequency analysis (detect music vs speech vs bang)
- [ ] Wake word detection (local, on Pi)
- [ ] Rhythm detection for music response

### Vision
- [ ] Face detection (is someone there?)
- [ ] Face recognition (who is it?)
- [ ] Motion detection
- [ ] Object recognition (what's that new thing?)

### Safety (Non-negotiable)
- [ ] Cliff/edge detection (IR sensors pointing down)
- [ ] Front/side obstacle detection
- [ ] Bump sensors as failsafe

---

## Movement Ideas

- Tail wag (happy, excited)
- Ear perk (alert, curious)
- Head tilt (confused, listening)
- Happy spin
- Cautious crouch
- Excited bounce
- Sleepy curl
- Retreat/hide
- Peek out cautiously
- Head bob (music)

---

## Emotional State

Persistent mood that affects behavior:
- Baseline: curious, content
- Excited: new person, play time
- Startled: loud noise, fades over time
- Sleepy: quiet environment
- Happy: music, familiar faces
- Nervous: stranger, new environment

Mood should decay/transition naturally over time.

---

## Architecture Ideas

### Tiered Decision System

```
┌─────────────────────────────────────────────────────┐
│  LAYER 0: Safety (Always On, <10ms)                 │
│  ├── Cliff detection → STOP                         │
│  ├── Obstacle detection → AVOID                     │
│  ├── Bump sensor → REVERSE                          │
│  └── CANNOT be overridden by emotion or personality │
├─────────────────────────────────────────────────────┤
│  LAYER 1: Local Fast Path (<50ms)                   │
│  ├── Basic reactions (startle, perk up, etc.)       │
│  ├── Known patterns (familiar face, routine sounds) │
│  ├── Emotional state updates                        │
│  └── Simple movement decisions                      │
├─────────────────────────────────────────────────────┤
│  LAYER 2: Local LLM Filter                          │
│  ├── "Is this interesting enough for cloud?"        │
│  ├── Novelty detection                              │
│  ├── Ambiguity detection                            │
│  └── Most requests STOP here (cost/latency saving)  │
├─────────────────────────────────────────────────────┤
│  LAYER 3: Cloud API (GCP, when needed)              │
│  ├── Unknown object identification                  │
│  ├── Complex scene understanding                    │
│  ├── Nuanced personality decisions                  │
│  └── Only called for genuinely novel situations     │
└─────────────────────────────────────────────────────┘
```

### Data Flow

```
┌─────────────────────────────────────────────────────┐
│                      Koji                           │
├─────────────────────────────────────────────────────┤
│  Sensors (Input)                                    │
│  ├── Microphone → FFT/amplitude analysis            │
│  ├── Camera → OpenCV/TFLite                         │
│  └── IR/Ultrasonic → cliff/obstacle                 │
├─────────────────────────────────────────────────────┤
│  Edge Processing (Raspberry Pi)                     │
│  ├── Sound classification (local)                   │
│  ├── Face detection (local)                         │
│  ├── Object detection (local, TFLite)               │
│  ├── Safety logic (always local, no latency)        │
│  └── LLM filter → cloud only if novel/ambiguous     │
├─────────────────────────────────────────────────────┤
│  Cloud (only when needed)                           │
│  ├── Complex scene understanding                    │
│  ├── Unknown object identification                  │
│  └── LLM personality decisions                      │
├─────────────────────────────────────────────────────┤
│  Decision Engine                                    │
│  ├── Current emotional state                        │
│  ├── Sensor input                                   │
│  ├── Spatial memory (obstacle map)                  │
│  ├── Recent event memory                            │
│  └── → Action output (JSON)                         │
├─────────────────────────────────────────────────────┤
│  Actuators (Output)                                 │
│  ├── Motors (movement, wheels/legs)                 │
│  ├── Servos (tail, ears, head)                      │
│  └── Speaker (whimpers, happy sounds, barks)        │
└─────────────────────────────────────────────────────┘
```

---

## Safety vs Emotion Model

The key insight: **Safety is physics, emotion is behavior selection.**

### The Golden Rule

```
Emotional State  →  affects WHAT Koji wants to do
Safety Layer     →  constrains HOW Koji can do it
```

Safety NEVER turns off. Not when scared, not when excited, not ever. A panicked Koji still knows where the edges are.

### Charming Clumsiness vs Stupid Repetition

Real animals are clumsy but not idiots. Koji should be the same.

**Acceptable (charming):**
- Startled by loud noise, bonks into unexpected chair while fleeing
- Skids a little close to edge when panicking before safety catches it
- Bumps into new obstacle that wasn't there before
- Misjudges a gap once, learns from it

**Not acceptable (stupid):**
- Hits the same chair every time it gets scared
- Falls off known edges
- Repeatedly collides with static furniture
- No learning, no spatial memory

### How It Works

```
┌──────────────────────────────────────────────────────────────┐
│  PANIC EVENT (e.g., loud noise)                              │
├──────────────────────────────────────────────────────────────┤
│  1. Emotional state → TERRIFIED                              │
│  2. Behavior selection → FLEE (fast, away from sound)        │
│  3. Path planning query spatial memory:                      │
│     - Known obstacles? Route around them                     │
│     - Known edges? Stay away                                 │
│     - Unknown obstacle in path? Might bonk (first time OK)   │
│  4. Safety layer (always on):                                │
│     - Cliff detected? STOP, even mid-flee                    │
│     - Obstacle collision? STOP, update spatial memory        │
│  5. Execute movement with safety constraints                 │
└──────────────────────────────────────────────────────────────┘
```

### Spatial Memory Strategy

#### The Problem

Full SLAM (Simultaneous Localization and Mapping) is overkill for a pet robot. It's complex, needs good sensors (ideally lidar), and is computationally expensive. We're building a pet, not a warehouse robot.

#### Phased Approach

**Phase 1-2: Reactive + Short-Term Memory (Option 3)**

No global map. Just remember recent collisions and avoid repeating the exact same mistake.

```go
type RecentObstacle struct {
    Direction    Direction     // where we were heading
    MovementType MovementType  // forward, turning, fleeing, etc.
    Timestamp    time.Time
}

type ShortTermMemory struct {
    recentBumps []RecentObstacle  // last N collisions
    ttl         time.Duration     // forget after this long (e.g., 30s)
}

// "I just bumped something while fleeing forward-left. Don't do that again."
func (m *ShortTermMemory) ShouldAvoid(dir Direction, movement MovementType) bool {
    for _, bump := range m.recentBumps {
        if bump.Direction == dir && bump.MovementType == movement {
            if time.Since(bump.Timestamp) < m.ttl {
                return true
            }
        }
    }
    return false
}
```

- Pros: Dead simple, no localization needed
- Cons: No persistent map, won't remember obstacles across restarts
- Good enough for: "Don't immediately hit the same thing twice"

**Phase 5+: Landmark-Based Zones (Future)**

Once vision is working, graduate to zone-based memory using visual landmarks.

```go
type Zone struct {
    ID         string
    Landmarks  []string  // recognized visual features
    Hazards    []Hazard  // cliffs, obstacles encountered here
    SafePaths  []Direction
}

// "I'm near the desk (I can see the desk leg pattern). 
// Last time I was here, there was a cliff to my left."
```

- Pros: Doesn't need precise localization, robust to drift
- Cons: Needs decent vision processing
- Deferred to: Phase 5 (Recognition & Memory)

#### What We're NOT Doing

- Full SLAM with occupancy grids
- Precise odometry-based positioning
- Lidar mapping
- Any of that "real robotics" shit that would take 6 months

#### Summary

| Phase | Memory Type | Complexity | Capability |
|-------|-------------|------------|------------|
| 1-2 | Reactive + short-term | Low | Don't repeat immediate mistakes |
| 5+ | Landmark-based zones | Medium | Remember areas, not coordinates |
| Never | Full SLAM | High | Overkill for a pet |

### First-Time vs Repeat Collisions

```go
// With short-term memory approach
func handleCollision(dir Direction, movement MovementType) {
    if shortTermMemory.ShouldAvoid(dir, movement) {
        // We JUST hit something going this way - pathfinding bug
        log.Error("repeated collision in same direction - shouldn't happen")
    } else {
        // First time hitting this - acceptable, learn from it
        shortTermMemory.Add(RecentObstacle{
            Direction:    dir,
            MovementType: movement,
            Timestamp:    time.Now(),
        })
        playSound("surprised_bonk")  // cute
    }
}
```

### Edge Behavior When Panicked

Even in TERRIFIED state, cliff sensors override movement:

```go
// Safety layer - cannot be bypassed
func (s *SafetyController) CheckMovement(intended Movement) Movement {
    if s.CliffAhead() {
        // Doesn't matter how scared we are
        return Movement{Stop: true}
    }
    if s.ObstacleAhead() && spatialMemory.IsKnown(obstacle) {
        // We know this is here, route around
        return s.FindAlternatePath(intended)
    }
    return intended  // OK to proceed
}
```

The "skid close to edge" charm comes from:
- Brief deceleration time (physics, not software)
- Visual appearance of "oh shit almost went over"
- But safety ALWAYS catches it before actual fall

---

## Hardware Feasibility

### Size Constraints

**Pi 5 dimensions:** 85mm x 56mm (credit card sized), plus 20-30mm for cooling.

**Minimum realistic chassis size:** ~15-20cm long. Think "small dog toy" or "chunky hamster."

The Pi isn't the problem — it's everything else:
- Battery pack (LiPo for decent runtime)
- Motor driver board
- Servo controller
- Sensors (IR, ultrasonic, camera, mic)
- Speaker
- Wiring harness

**Reference points:**
| Robot | Size | Notes |
|-------|------|-------|
| Petoi Bittle | 20cm x 11cm x 11cm | Robot dog, good baseline |
| Anki Vector | 10cm x 6cm x 7cm | Tiny, but cloud-dependent (cheating) |

**Verdict:** Cute is doable. "Fits in palm" is not, unless we cut features.

### Compute Options

| Board | Pros | Cons |
|-------|------|------|
| **Pi 5 8GB** (baseline) | Familiar, good ecosystem, enough for small LLMs | Power hungry (5-7W), no ML accelerator, needs cooling |
| **Pi 5 + Coral USB** | ML acceleration for vision tasks | Extra power, USB bandwidth contention |
| **Jetson Orin Nano** | Built-in GPU, better ML perf/watt | Pricier, less GPIO-friendly, steeper learning curve |
| **ESP32 + Pi Zero 2W** | ESP32 for real-time safety, Pi for brain | Two boards, more complexity, save for v2 |
| **Orange Pi 5** | Cheaper, has NPU | Worse software support, community smaller |

**Decision:** Start with Pi 5 8GB. Add Coral USB if vision ML is too slow. Revisit Jetson if we hit a wall.

### Hardware Candidates

#### Compute
- Raspberry Pi 5 8GB (main brain)
- Coral USB Accelerator (ML boost for vision, add if needed)

#### Chassis
- Target size: 15-20cm body length
- Needs to be stable enough not to tip over when startled
- Reference: Petoi Bittle, Freenove Robot Dog, or custom 3D print

#### Sensors
- USB microphone or I2S MEMS mic
- Pi Camera v3 or USB webcam (wide angle preferred)
- IR distance sensors x3-4 (cliff detection)
- Ultrasonic sensors x2 (obstacle detection)
- Bump switches x2 (failsafe)

#### Actuators
- Servos for expressive parts (tail, ears, head)
- DC motors or servos for locomotion
- Small speaker (3W is plenty)

---

## Tech Stack

- **Go** — main application, sensor coordination, decision loop
- **TensorFlow Lite** — on-device ML (face detection, object detection)
- **OpenCV** — image processing
- **GCP** — cloud AI services (Vision API, Vertex AI for LLM)
- **Local LLM** — filter/simple decisions (see LLM Strategy below)
- **Ollama / llama.cpp** — local LLM inference runtime

---

## LLM Strategy

### The Core Problem

LLMs don't have emotions — they're stateless text predictors. But we can get emotionally coherent behavior by:

1. **External state machine** — Code tracks mood (curious, scared, happy). LLM picks actions consistent with that mood.
2. **System prompt engineering** — Give personality + current emotional context.
3. **Constrained outputs** — Don't free-form. Vocabulary of actions, pick from them.
4. **Code-driven mood transitions** — LLM doesn't decide mood changes, rules do.

### Personality vs Mood

**Personality** = constant traits, the "who is Koji" layer. Baked into system prompt.
**Mood** = temporary emotional state, changes based on events. Passed as context.

```
┌─────────────────────────────────────────────────────────────┐
│  PERSONALITY (constant)                                     │
│  - Curious by nature                                        │
│  - Easily excited                                           │
│  - A little clumsy but enthusiastic                         │
│  - Loves music                                              │
│  - Wary of strangers at first                               │
│  - NOT a helpful assistant, just a pet                      │
├─────────────────────────────────────────────────────────────┤
│  MOOD (variable)                                            │
│  - Current: frightened                                      │
│  - Intensity: 0.8                                           │
│  - Duration: 3 seconds                                      │
│  - Decaying toward: cautious → curious (baseline)           │
└─────────────────────────────────────────────────────────────┘
```

A frightened Koji is still curious by nature — he'll peek out to investigate after hiding, rather than cowering forever. Personality shapes *how* he experiences each mood.

### Two-Tier LLM Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  LOCAL LLM (Pi 5)                                           │
│  ├── Model: Llama 3.2 1B or Phi-3-mini (quantized Q4)       │
│  ├── Runtime: Ollama or llama.cpp                           │
│  ├── Speed: ~5-10 tokens/sec                                │
│  ├── Role: Filter + simple personality responses            │
│  └── Decisions:                                             │
│      - "Is this novel enough for cloud?" (yes/no)           │
│      - "What basic action fits this mood?" (from vocab)     │
│      - "Quick reaction to known stimulus"                   │
├─────────────────────────────────────────────────────────────┤
│  CLOUD LLM (GCP Vertex AI)                                  │
│  ├── Model: Gemini Pro or Claude (via API)                  │
│  ├── Latency: 500ms-2s (acceptable for non-urgent)          │
│  ├── Role: Complex/nuanced decisions                        │
│  └── Decisions:                                             │
│      - "What is this unknown object?"                       │
│      - "How should I react to this complex scene?"          │
│      - "Generate varied response for repeated situation"    │
└─────────────────────────────────────────────────────────────┘
```

### What Can Run Locally on Pi 5 8GB?

| Model | Size (Q4) | RAM Needed | Speed | Verdict |
|-------|-----------|------------|-------|---------|
| **TinyLlama 1.1B** | ~600MB | ~1-2GB | ~8-12 tok/s | Fast, quality meh |
| **Llama 3.2 1B** | ~700MB | ~2GB | ~5-10 tok/s | Best small option |
| **Phi-3-mini 3.8B** | ~2GB | ~4GB | ~3-5 tok/s | Smarter, slower |
| **Llama 3.2 3B** | ~1.8GB | ~3-4GB | ~2-4 tok/s | Good balance |
| **Gemma 2 2B** | ~1.5GB | ~3GB | ~4-6 tok/s | Decent alternative |
| **Mistral 7B** | ~4GB | ~6GB | <1 tok/s | Too slow |
| **Llama 3 8B** | ~4.5GB | ~7GB | <1 tok/s | Too slow |

**Recommendation:** Start with **Llama 3.2 1B Q4_K_M** for speed. If responses feel dumb, try **Phi-3-mini** or **Llama 3.2 3B** and see if latency is acceptable.

### Local LLM Inference Stack

| Option | Pros | Cons |
|--------|------|------|
| **Ollama** | Easy setup, good defaults, HTTP API | Slight overhead |
| **llama.cpp** | Fastest, C++, direct control | More setup, no API server by default |
| **ExecuTorch** | Meta's embedded runtime, optimized | Newer, less documented |

**Decision:** Use Ollama for Phase 0 prototyping (easy). Switch to llama.cpp if we need every last drop of performance.

### System Prompt Design

```
You are Koji, a small robot pet with a curious, excitable personality.

Current state:
- Mood: {FRIGHTENED}
- Mood intensity: {0.8}
- Time in mood: {3 seconds}
- Location: {corner, backed against wall}
- Recent events: {loud bang detected 3s ago}

Available actions: [cower, peek, whimper, stay_still, cautious_look, flee]

Given the sensor input below, choose ONE action from the list.
Respond with JSON: {"action": "<action>", "reason": "<brief reason>"}

Sensor input:
{silence, no motion detected, familiar room}
```

### When to Call Cloud

The local LLM filter decides. Simple heuristic:

```go
type CloudDecision struct {
    ShouldCall bool
    Reason     string
}

// Local LLM prompt for filter
filterPrompt := `
You are a filter deciding if a situation needs cloud AI processing.
Answer YES only if:
- Object/person is unrecognized AND interesting
- Situation is ambiguous or complex
- Novel combination of stimuli not seen before

Answer NO if:
- Routine event (familiar face, normal sounds)
- Simple reaction will suffice
- Safety-critical (must act immediately, no time for cloud)

Respond: {"call_cloud": true/false, "reason": "..."}
`
```

### Emotional State Machine (Code, Not LLM)

The LLM doesn't control mood transitions. Deterministic rules do:

```go
type Mood string
const (
    MoodCurious   Mood = "curious"    // baseline
    MoodExcited   Mood = "excited"
    MoodStartled  Mood = "startled"
    MoodFrightened Mood = "frightened"
    MoodHappy     Mood = "happy"
    MoodSleepy    Mood = "sleepy"
    MoodCautious  Mood = "cautious"
)

// Deterministic transitions
var transitions = map[Event]map[Mood]Mood{
    EventLoudNoise: {
        MoodCurious:  MoodStartled,
        MoodHappy:    MoodStartled,
        MoodSleepy:   MoodFrightened,  // worse when sleepy
        MoodStartled: MoodFrightened,  // escalate
    },
    EventFamiliarFace: {
        MoodCurious:   MoodHappy,
        MoodCautious:  MoodHappy,
        MoodFrightened: MoodCautious,  // calming, but still wary
    },
    // ... etc
}

// Mood decay over time (return to baseline)
func (m Mood) Decay(elapsed time.Duration) Mood {
    // Frightened → Cautious → Curious
    // Excited → Happy → Curious
    // etc.
}
```

The LLM's job is: "Given mood X and stimulus Y, pick action from list Z." That's it.

---

## Development Phases

See DEVELOPMENT.md for the full plan.

---

## Random Ideas / Future

- "Memory" of favorite spots in the house
- Learn owner's schedule (excited when they usually come home)
- Different personalities/breeds as config options
- Mobile app for "pet status" and settings
- Treat dispenser integration (the robot gets treats?)
- Companion robot friends (multi-robot awareness)

---

## Open Questions

- Wheels vs legs? (wheels = simpler, legs = cuter but complex)
- How to handle stairs? (probably just avoid them)
- Battery life considerations
- How loud is too loud for motors/servos?
- Wake word: "Koji" or something else?

---

## Name

**Koji** (麹) — Japanese, means "mold/ferment" used in sake and miso. 

Also just sounds cute as hell.

---

*Dump any new ideas below:*

