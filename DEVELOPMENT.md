# Koji - Development Plan

## Philosophy

1. Don't die (safety first)
2. Don't be annoying (reliable behavior)
3. Be charming (personality and reactions)

---

## Phase 0: Personality Prototype (No Hardware)

**Goal:** Validate the personality feels alive before building anything physical.

### Tasks
- [ ] Design LLM system prompt for Koji's personality
- [ ] Build CLI simulator that accepts fake sensor input
- [ ] Define JSON schema for sensor input and action output
- [ ] Implement emotional state machine (mood persistence, decay, transitions)
- [ ] Implement action vocabulary (constrained outputs)
- [ ] Test various scenarios (loud noise, music, familiar face, stranger)
- [ ] Tune personality until it feels right
- [ ] Benchmark local LLM options (Llama 3.2 1B vs Phi-3-mini vs Llama 3.2 3B)
- [ ] Design cloud filter prompt ("is this novel enough?")

### Tech
- Go CLI application
- **Local LLM:** Ollama + Llama 3.2 1B (Q4_K_M) for fast iteration
- **Cloud LLM:** GCP Vertex AI (Gemini) for complex decisions during testing
- JSON for all I/O
- Bubble Tea for interactive TUI (optional)

### LLM Architecture to Validate
```
Sensor Event → Emotional State Machine → Local LLM (action selection)
                                      ↓
                            Cloud Filter ("is this novel?")
                                      ↓
                            Cloud LLM (if needed)
```

### Deliverable
A terminal app where you can type sensor events and get believable pet reactions. Should validate:
1. Emotional state transitions feel natural
2. Local LLM responses are fast enough (<500ms)
3. Cloud filter correctly identifies novel vs routine situations
4. Action vocabulary covers needed behaviors

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
- [ ] Implement face recognition (not just detection)
- [ ] Build face database (owner vs stranger vs known friends)
- [ ] Add object recognition for interesting things
- [ ] Implement "allowlist" filter — only cloud-call for unknowns
- [ ] Add location memory (favorite spots, danger zones)
- [ ] Persist memory across restarts

### Tech
- Face embeddings (dlib, or cloud API)
- Local storage (SQLite or JSON)
- GCP Vision API for unknown objects

### Deliverable
Koji recognizes owner, gets excited. Wary of strangers. Remembers objects.

---

## Phase 6: Full Personality Integration

**Goal:** Everything working together seamlessly.

### Tasks
- [ ] Tune emotional state transitions
- [ ] Balance local vs cloud decision making
- [ ] Optimize response latency (should feel instant)
- [ ] Add behavioral variety (same input, slightly different reactions)
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

## Hardware Shopping List (Draft)

### Compute
| Item | Purpose | Notes |
|------|---------|-------|
| Raspberry Pi 5 8GB | Main brain | 8GB required for local LLM |
| Coral USB Accelerator | ML boost | Optional, helps with TFLite vision |
| MicroSD card (64GB+) | Storage | Fast one, need space for LLM models |
| NVMe SSD (optional) | Faster storage | Pi 5 has PCIe, helps LLM load times |

### Sensors
| Item | Purpose | Notes |
|------|---------|-------|
| Pi Camera v2/v3 or USB webcam | Vision | Wide angle helpful |
| USB microphone or I2S MEMS | Audio | I2S is cleaner |
| IR distance sensors (x3-4) | Cliff detection | Sharp GP2Y0A21YK or similar |
| Ultrasonic sensor (x2) | Obstacle detection | HC-SR04 |
| Bump switches (x2) | Failsafe | Microswitch type |

### Actuators
| Item | Purpose | Notes |
|------|---------|-------|
| Servo motors (x3-5) | Tail, ears, head | SG90 or MG90S |
| DC motors + driver | Locomotion | Depends on chassis |
| Small speaker | Sounds | 3W is plenty |

### Chassis
| Option | Pros | Cons |
|--------|------|------|
| Buy kit (e.g., PiDog, Freenove) | Fast start | Less custom |
| 3D print custom | Full control | Need printer, design skills |
| Hack a toy | Cheap, fun | Unpredictable |

### Power
| Item | Purpose | Notes |
|------|---------|-------|
| LiPo battery | Main power | 5V, 3A+ output |
| Battery management board | Charging, protection | |
| Power bank (simpler option) | Easier but bulkier | |

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02 | Pi 5 8GB as baseline compute | Need 8GB for local LLM (Llama 3.2 1B/3B). Can add Coral for vision. |
| 2026-02 | Two-tier LLM architecture | Local LLM for speed + filter, cloud for complex/novel situations. Cost and latency optimization. |
| 2026-02 | Ollama for initial LLM runtime | Easy setup, good for prototyping. Switch to llama.cpp if perf needed. |
| 2026-02 | Emotional state in code, not LLM | Deterministic mood transitions are more reliable. LLM just picks actions. |
| 2026-02 | Target chassis size 15-20cm | Minimum viable size for all components. "Chunky hamster" aesthetic. |

---

## Resources

### Hardware
- [Raspberry Pi GPIO Pinout](https://pinout.xyz/)
- [Raspberry Pi 5 Specs](https://www.raspberrypi.com/products/raspberry-pi-5/)
- [Coral USB Accelerator](https://coral.ai/products/accelerator/)

### ML & Vision
- [TensorFlow Lite for Pi](https://www.tensorflow.org/lite/guide/python)
- [GCP Vision API](https://cloud.google.com/vision)
- [GCP Vertex AI](https://cloud.google.com/vertex-ai)

### Local LLM
- [Ollama](https://ollama.ai/) — easy local LLM runtime
- [llama.cpp](https://github.com/ggerganov/llama.cpp) — fast CPU inference
- [Llama 3.2 Models](https://llama.meta.com/) — 1B and 3B variants
- [Phi-3](https://azure.microsoft.com/en-us/blog/introducing-phi-3/) — Microsoft small models
- [GGUF Model Format](https://github.com/ggerganov/ggml) — quantized model format

### Development
- [Bubble Tea (Go TUI)](https://github.com/charmbracelet/bubbletea) — for debug/control interface
- [periph.io](https://periph.io/) — Go GPIO library

