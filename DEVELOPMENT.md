# Koji - Development Plan

## Philosophy

1. Don't die (safety first)
2. Don't be annoying (reliable behavior)
3. Be charming (personality and reactions)

---

## Phase 0: Personality Prototype (No Hardware)

**Goal:** Validate the personality feels alive before building anything physical.

### Tasks
- [x] Design LLM system prompt for Koji's personality
- [x] Build CLI simulator that accepts fake sensor input
- [ ] Define JSON schema for sensor input and action output
- [x] Implement emotional state machine (mood persistence, decay, transitions)
- [x] Implement action vocabulary (constrained outputs)
- [x] Test various scenarios (loud noise, music, familiar face, stranger)
- [x] Implement variation engine (weighted random, micro-behaviors, mood echoes)
- [ ] Tune personality until it feels right
- [ ] Benchmark local LLM options (Llama 3.2 1B vs Phi-3-mini vs Llama 3.2 3B)
- [ ] Design cloud filter prompt ("is this novel enough?")

### Tech
- Go CLI application
- **Local LLM:** Ollama + Llama 3.2 1B (Q4_K_M) for fast iteration
- **Cloud LLM:** GCP Vertex AI (Gemini) for complex decisions during testing
- JSON for all I/O
- Bubble Tea for interactive TUI (optional)

### Architecture
```
Sensor Event → Emotional State Machine → Variation Engine (action selection)
                     ↓                          ↓
              Mood Echoes ←──────────── Records mood changes
                                              ↓
                                    Action + Modifier + Micro-behavior

Optional LLM path (for complex/novel situations):
Sensor Event → Emotional State Machine → Local LLM (action selection)
                                      ↓
                            Cloud Filter ("is this novel?")
                                      ↓
                            Cloud LLM (if needed)
```

### Variation Engine (Implemented)
Instead of asking an LLM to pick actions, the variation engine uses:
- **Weighted random selection**: Each mood has actions with probability weights
- **Action modifiers**: Same action varies by intensity (slow/fast/frantic/gentle)
- **Micro-behaviors**: Idle animations (ear twitches, sighs, weight shifts)
- **Mood echoes**: Past moods bleed into current behavior (still jumpy after being scared)

### Deliverable
A terminal app where you can type sensor events and get believable pet reactions. Should validate:
1. Emotional state transitions feel natural
2. Variation engine produces lifelike, non-repetitive behavior
3. Action vocabulary covers needed behaviors
4. Mood echoes create realistic recovery from emotional states
5. (Optional) Local LLM responses are fast enough (<500ms) for complex situations
6. (Optional) Cloud filter correctly identifies novel vs routine situations

---

## Phase 1: Safety Foundation

**Goal:** A robot that cannot hurt itself.

### Tasks
- [ ] Select chassis/base platform
- [ ] Integrate cliff sensors (IR pointing down)
- [ ] Integrate front/side obstacle sensors
- [ ] Implement bump switch failsafe
- [ ] Write safety controller that overrides all other behavior
- [ ] Test extensively — push it toward edges, obstacles

### Tech
- Raspberry Pi
- Go for sensor reading and motor control
- GPIO libraries

### Deliverable
A mobile base that stops at edges and avoids obstacles. No personality yet, just survival.

---

## Phase 2: Basic Movement & Expression

**Goal:** Koji can move and emote.

### Tasks
- [ ] Implement basic locomotion (forward, back, turn)
- [ ] Add expressive servos (tail, ears, head tilt)
- [ ] Define action vocabulary (wag, perk, tilt, crouch, etc.)
- [ ] Map action JSON from personality engine to physical movements
- [ ] Add speaker for simple sounds (whimper, happy chirp, alert bark)

### Tech
- Servo control via PWM
- Audio playback (aplay or similar)
- Go coordinating everything

### Deliverable
A robot that can physically express the actions from Phase 0.

---

## Phase 3: Sound Awareness

**Goal:** Koji reacts to audio environment.

### Tasks
- [ ] Integrate microphone
- [ ] Implement amplitude detection (volume levels)
- [ ] Implement basic frequency analysis (FFT)
- [ ] Classify: silence, speech, music, loud bang
- [ ] Wire sound events into personality engine
- [ ] Test reactions: hide from bang, bob to music, perk up at speech

### Tech
- USB or I2S microphone
- Go audio processing (or shell out to Python for DSP if needed)
- No cloud needed for basic classification

### Deliverable
Koji responds appropriately to sounds in the environment.

---

## Phase 4: Vision - Safety & Detection

**Goal:** Koji can see and avoid things, detect motion and faces.

### Tasks
- [ ] Integrate camera (Pi Camera or USB)
- [ ] Implement motion detection (simple frame differencing)
- [ ] Add face detection (local, TFLite or OpenCV)
- [ ] Supplement obstacle detection with vision
- [ ] Wire visual events into personality engine

### Tech
- OpenCV for motion detection
- TensorFlow Lite for face detection
- Go orchestration (may need CGo or subprocess for OpenCV)

### Deliverable
Koji notices movement and faces, reacts accordingly.

---

## Phase 5: Recognition & Memory

**Goal:** Koji knows who's who and remembers things.

### Tasks
- [x] Implement face recognition (embedding-based, not just detection)
- [x] Build face database (owner vs stranger vs known friends)
- [x] Add emotion detection for recognized faces
- [x] Create local web UI for face enrollment
- [ ] Add object recognition for interesting things
- [ ] Implement "allowlist" filter — only cloud-call for unknowns
- [ ] Add location memory (favorite spots, danger zones)
- [ ] Persist memory across restarts
- [ ] Integrate face recognition with personality engine

### Face Recognition Architecture (Implemented)
```
Camera Frame → Face Detection → Face Embedding (128/512-dim vector)
                                      ↓
                              Cosine Similarity vs FaceDB
                                      ↓
                    ┌─────────────────┴─────────────────┐
                    ↓                                   ↓
            Match (>0.6 similarity)            No Match (<0.6)
                    ↓                                   ↓
            Return Person + Emotion            Return "stranger"
```

### Owner Enrollment Flow
No training needed! Uses pre-trained face embedding model:
1. User visits `http://koji.local:8080`
2. Enters name, selects "Owner" relationship
3. Looks at camera for ~10 seconds (captures 5-10 samples)
4. System extracts embeddings and stores in local JSON database
5. Done - Koji now recognizes owner

### Tech
- Face detection: MediaPipe or OpenCV (runs on Pi)
- Face embeddings: InsightFace or dlib (one-time inference per face)
- Emotion detection: FER or DeepFace (happy/sad/angry/surprised/neutral)
- Local storage: JSON file with embeddings
- Enrollment UI: Embedded web server at `http://koji.local:8080`

### Deliverable
Koji recognizes owner, gets excited. Wary of strangers. Reads owner's mood and reacts accordingly.

---

## Phase 6: Full Personality Integration

**Goal:** Everything working together seamlessly.

### Tasks
- [ ] Tune emotional state transitions
- [ ] Balance local vs cloud decision making
- [ ] Optimize response latency (should feel instant)
- [x] Add behavioral variety (same input, slightly different reactions) — variation engine
- [ ] Long-term mood (had a good day vs rough day)
- [ ] Power management and battery monitoring
- [ ] Optimize LLM inference (consider llama.cpp if Ollama too slow)
- [ ] Tune cloud filter to minimize unnecessary API calls
- [ ] Add offline fallback mode (no network = full local)

### LLM Tuning Targets
| Metric | Target |
|--------|--------|
| Local LLM response | <500ms |
| Cloud filter decision | <200ms |
| Cloud LLM response (when used) | <2s |
| Cloud call frequency | <10% of decisions |

### Deliverable
A complete, coherent robot pet personality.

---

## Phase 7: Polish & QoL

**Goal:** Ready for daily use.

### Tasks
- [ ] Charging dock awareness (go home when tired)
- [ ] Mobile app for status/config (optional)
- [ ] OTA updates
- [ ] Logging and diagnostics
- [ ] Quiet hours mode (sleepy at night)

---

## Hardware Architecture

### Two-Board Design
```
┌─────────────────────────────────────────────────────────────┐
│  "High Brain" - Raspberry Pi 5                             │
│  - Camera + face recognition                               │
│  - Microphone + audio classification                       │
│  - Personality engine + variation engine                   │
│  - LLM (optional, for complex situations)                  │
│  - Web UI for enrollment                                   │
└──────────────────────┬──────────────────────────────────────┘
                       │ UART Serial (simple text commands)
                       │ e.g., "servo:tail:90:fast\n"
┌──────────────────────▼──────────────────────────────────────┐
│  "Low Brain" - ESP32                                        │
│  - Cliff sensors (analog, real-time)                       │
│  - Ultrasonic sensors                                      │
│  - Bump switches                                           │
│  - Servo control (PWM)                                     │
│  - Motor control                                           │
│  - SAFETY OVERRIDES (runs even if Pi crashes)              │
└─────────────────────────────────────────────────────────────┘
```

**Why two boards?**
- Safety isolation: ESP32 stops motors at cliff edge even if Pi is frozen
- Real-time control: Microcontrollers handle tight timing loops
- Cleaner separation: High-level decisions vs low-level actuation
- Simpler wiring: All sensors to one board, one cable to Pi
- Hot-swap brains: Upgrade Pi to Jetson later without rewiring

---

## Hardware Shopping List

### High Brain (Perception + Personality)
| Item | Purpose | Est. Price | Notes |
|------|---------|------------|-------|
| Raspberry Pi 5 4GB | Main compute | $55 | 4GB enough now that GPIO moved to ESP32 |
| MicroSD card 64GB | Storage | $12 | Fast A2 card, for OS + models |
| Pi Camera v3 | Vision | $25 | Wide angle version preferred |
| USB microphone | Audio input | $10 | Or I2S MEMS for cleaner signal |
| USB speaker/DAC | Audio output | $10 | Small powered speaker |
| **Subtotal** | | **~$112** | |

### Low Brain (Sensors + Actuators)
| Item | Purpose | Est. Price | Notes |
|------|---------|------------|-------|
| ESP32-WROOM-32E module | Sensor/motor controller | $5 | Classic ESP32, best library support |
| ESP32 dev board | Development | $8 | ESP32-DevKitC or similar, has USB |
| IR distance sensors x4 | Cliff detection | $12 | Sharp GP2Y0A21YK (analog) |
| Ultrasonic sensors x2 | Obstacle detection | $4 | HC-SR04 |
| Bump switches x2 | Failsafe | $2 | Microswitch, normally-closed |
| Servo motors x5 | Tail, ears, head tilt | $15 | SG90 (plastic) or MG90S (metal gear) |
| DC motors x2 + driver | Locomotion | $15 | N20 gear motors + TB6612 driver |
| PCA9685 PWM driver | Servo control | $5 | 16-channel, frees up ESP32 pins |
| Level shifter | Pi-ESP32 comms | $2 | 3.3V Pi to 3.3V ESP32 (may not need) |
| **Subtotal** | | **~$68** | |

### Power
| Item | Purpose | Est. Price | Notes |
|------|---------|------------|-------|
| 2S LiPo battery (7.4V) | Main power | $20 | 2000-3000mAh, 18650 cells work too |
| BMS board | Battery protection | $5 | 2S balance charging |
| 5V 3A buck converter | Pi power | $5 | From 7.4V LiPo |
| 5V regulator for servos | Servo power | $3 | Separate to avoid brownouts |
| Power switch | On/off | $2 | |
| **Subtotal** | | **~$35** | |

### Chassis & Misc
| Item | Purpose | Est. Price | Notes |
|------|---------|------------|-------|
| Chassis/frame | Structure | $20-50 | 3D print, kit, or hacked toy |
| Wheels x2 + caster | Mobility | $8 | Depends on chassis |
| Wires, connectors | Assembly | $10 | JST, Dupont, etc. |
| Standoffs, screws | Mounting | $5 | M2.5 and M3 |
| **Subtotal** | | **~$45-75** | |

### Total Estimate
| Category | Price |
|----------|-------|
| High Brain | ~$112 |
| Low Brain | ~$68 |
| Power | ~$35 |
| Chassis (3D printed) | ~$15 (filament) |
| Misc (wires, screws) | ~$15 |
| **Total** | **~$245** |

### Optional Upgrades
| Item | Purpose | Est. Price | Notes |
|------|---------|------------|-------|
| Coral USB Accelerator | ML acceleration | $60 | If face detection is too slow |
| NVMe SSD + adapter | Faster storage | $30 | If LLM loading is too slow |
| Pi 5 8GB (instead of 4GB) | More RAM | +$20 | If running larger LLM models |
| Better servos (MG996R) | More torque | +$20 | If robot is heavier |

### ESP32 Pin Allocation (Draft)
```
Analog (ADC1 - can use with WiFi):
  GPIO32 - Cliff sensor front-left
  GPIO33 - Cliff sensor front-right
  GPIO34 - Cliff sensor rear-left
  GPIO35 - Cliff sensor rear-right

Digital:
  GPIO16 - Ultrasonic 1 trigger
  GPIO17 - Ultrasonic 1 echo
  GPIO18 - Ultrasonic 2 trigger
  GPIO19 - Ultrasonic 2 echo
  GPIO21 - Bump switch left (INPUT_PULLUP)
  GPIO22 - Bump switch right (INPUT_PULLUP)

I2C (for PCA9685 PWM driver):
  GPIO21 - SDA
  GPIO22 - SCL

Motor control:
  GPIO25 - Motor A PWM
  GPIO26 - Motor A direction
  GPIO27 - Motor B PWM
  GPIO14 - Motor B direction

UART (to Pi):
  GPIO1  - TX (to Pi RX)
  GPIO3  - RX (from Pi TX)
```

### Chassis (3D Printed)
Custom 3D printed chassis is the plan. Parts to design/print:

| Part | Notes |
|------|-------|
| Main body shell | Houses Pi, ESP32, battery |
| Sensor mounts | Adjustable angle for cliff/ultrasonic sensors |
| Ear mechanisms | Servo-driven, expressive |
| Tail mount + linkage | Wagging mechanism |
| Head/face plate | Camera + "eyes", tilt servo mount |
| Wheel hubs | Interface with motors |
| Cable management | Internal routing, strain relief |

**Design considerations:**
- Easy access to Pi/ESP32 for debugging
- Ventilation for Pi (it runs warm)
- Battery compartment with access for charging
- Snap-fit or screws for assembly
- Target size: 15-20cm ("chunky hamster")

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02 | Pi 5 8GB as baseline compute | Need 8GB for local LLM (Llama 3.2 1B/3B). Can add Coral for vision. |
| 2026-02 | Two-tier LLM architecture | Local LLM for speed + filter, cloud for complex/novel situations. Cost and latency optimization. |
| 2026-02 | Ollama for initial LLM runtime | Easy setup, good for prototyping. Switch to llama.cpp if perf needed. |
| 2026-02 | Emotional state in code, not LLM | Deterministic mood transitions are more reliable. LLM just picks actions. |
| 2026-02 | Target chassis size 15-20cm | Minimum viable size for all components. "Chunky hamster" aesthetic. |
| 2026-02 | Reactive short-term memory for Phase 1-2 | No SLAM, no precise localization. Just "don't repeat the same bump." Simple and good enough. |
| 2026-02 | Defer landmark-based zones to Phase 5 | Need vision working first. Will use visual landmarks instead of coordinates. |
| 2026-02 | Personality in prompt, mood in context | Personality is constant (curious, excitable), mood is variable (frightened, happy). Separation of concerns. |
| 2026-02 | Variation engine over LLM for action selection | LLM is overkill for picking from a small action list. Weighted random + mood echoes feels more alive with zero latency. LLM reserved for truly complex situations. |
| 2026-02 | Embedding-based face recognition | No training needed - use pre-trained model for embeddings, cosine similarity for matching. Simple on-device enrollment via web UI. |
| 2026-02 | Local web UI for face enrollment | Simpler than a mobile app, no app store needed. Just visit http://koji.local:8080 and look at the camera. |
| 2026-02 | Two-board architecture (Pi + ESP32) | Separates high-level perception/personality from real-time sensor/motor control. Safety isolation, cleaner wiring, can upgrade Pi without rewiring. |
| 2026-02 | ESP32-WROOM-32E for low brain | Classic ESP32 has more ADC channels (18) and PWM (16) than newer variants. Best documented, most libraries. RISC-V variants (C3, C6) have fewer analog pins. |
| 2026-02 | Pi 5 4GB (not 8GB) for high brain | With GPIO offloaded to ESP32, 4GB is sufficient. LLM is optional/rare now that variation engine handles most decisions. Save $20. |
| 2026-02 | 3D printed chassis | Custom fit for all components, can iterate on design, proper creature aesthetic instead of robot kit look. |

---

## Resources

### Hardware
- [Raspberry Pi GPIO Pinout](https://pinout.xyz/)
- [Raspberry Pi 5 Specs](https://www.raspberrypi.com/products/raspberry-pi-5/)
- [Coral USB Accelerator](https://coral.ai/products/accelerator/)

### ML & Vision
- [TensorFlow Lite for Pi](https://www.tensorflow.org/lite/guide/python)
- [MediaPipe Face Detection](https://developers.google.com/mediapipe/solutions/vision/face_detector) — fast on-device face detection
- [InsightFace](https://github.com/deepinsight/insightface) — face recognition embeddings
- [DeepFace](https://github.com/serengil/deepface) — emotion detection + face recognition
- [dlib](http://dlib.net/) — classic face recognition library
- [GCP Vision API](https://cloud.google.com/vision)
- [GCP Vertex AI](https://cloud.google.com/vertex-ai)

### Local LLM
- [Ollama](https://ollama.ai/) — easy local LLM runtime
- [llama.cpp](https://github.com/ggerganov/llama.cpp) — fast CPU inference
- [Llama 3.2 Models](https://llama.meta.com/) — 1B and 3B variants
- [Phi-3](https://azure.microsoft.com/en-us/blog/introducing-phi-3/) — Microsoft small models
- [GGUF Model Format](https://github.com/ggerganov/ggml) — quantized model format

### ESP32 / Low Brain
- [ESP-IDF](https://docs.espressif.com/projects/esp-idf/en/latest/) — official ESP32 framework
- [Arduino-ESP32](https://github.com/espressif/arduino-esp32) — Arduino core for ESP32 (easier to start)
- [ESP32 Pinout Reference](https://randomnerdtutorials.com/esp32-pinout-reference-gpios/)
- [PCA9685 Library](https://github.com/adafruit/Adafruit-PWM-Servo-Driver-Library) — for servo control board

### Development
- [Bubble Tea (Go TUI)](https://github.com/charmbracelet/bubbletea) — for debug/control interface
- [periph.io](https://periph.io/) — Go GPIO library (Pi side)
- [tinygo](https://tinygo.org/) — Go for microcontrollers (potential ESP32 option)

