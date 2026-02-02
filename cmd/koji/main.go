package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alex/koji/internal/llm"
	"github.com/alex/koji/internal/personality"
)

type app struct {
	state        *personality.EmotionalState
	llmClient    *llm.Client
	engine       *llm.PersonalityEngine
	recentEvents []personality.Event
	useLLM       bool
}

func main() {
	// Flags
	ollamaURL := flag.String("ollama", "http://localhost:11434", "Ollama API URL")
	model := flag.String("model", "llama3.2:1b", "LLM model to use")
	noLLM := flag.Bool("no-llm", false, "Disable LLM, use only deterministic actions")
	flag.Parse()

	app := &app{
		state:        personality.NewEmotionalState(),
		recentEvents: make([]personality.Event, 0, 10),
		useLLM:       !*noLLM,
	}

	fmt.Println("=== Koji Emotional State Simulator ===")
	fmt.Println()

	// Try to connect to LLM if enabled
	if app.useLLM {
		app.llmClient = llm.NewClient(llm.Config{
			BaseURL: *ollamaURL,
			Model:   *model,
			Timeout: 30 * time.Second,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := app.llmClient.Ping(ctx)
		cancel()

		if err != nil {
			fmt.Printf("Warning: Cannot connect to Ollama at %s: %v\n", *ollamaURL, err)
			fmt.Println("Running in deterministic mode (no LLM).")
			fmt.Println("Start Ollama with: ollama serve")
			fmt.Println()
			app.useLLM = false
		} else {
			app.engine = llm.NewPersonalityEngine(app.llmClient)
			fmt.Printf("Connected to Ollama (model: %s)\n", *model)
			fmt.Println("LLM will select actions based on personality.")
			fmt.Println()
		}
	} else {
		fmt.Println("Running in deterministic mode (--no-llm)")
		fmt.Println()
	}

	app.printState()
	app.printHelp()

	// Start decay ticker
	decayTicker := time.NewTicker(1 * time.Second)
	defer decayTicker.Stop()

	// Channel for user input
	inputChan := make(chan string)
	go readInput(inputChan)

	for {
		select {
		case <-decayTicker.C:
			if app.state.Decay() {
				fmt.Printf("\n[decay] Mood decayed after %s\n", app.state.Duration().Round(time.Second))
				app.printState()
				fmt.Print("> ")
			}

		case input := <-inputChan:
			if input == "" {
				continue
			}
			app.handleInput(input)
		}
	}
}

func (a *app) handleInput(input string) {
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "quit", "exit", "q":
		fmt.Println("Bye!")
		os.Exit(0)
	case "help", "?":
		a.printHelp()
		return
	case "status", "s":
		a.printState()
		return
	case "actions", "a":
		a.printActions()
		return
	case "llm":
		a.toggleLLM()
		return
	}

	event := parseEvent(input)
	if event == "" {
		fmt.Printf("Unknown event: %s (type 'help' for options)\n", input)
		return
	}

	ctx := personality.NewEventContext(event)

	// Check for intensity modifier
	if strings.Contains(input, "!") || strings.Contains(input, "loud") {
		ctx = ctx.WithIntensity(0.9)
	} else if strings.Contains(input, "soft") || strings.Contains(input, "quiet") {
		ctx = ctx.WithIntensity(0.2)
	}

	// Track recent events
	a.recentEvents = append(a.recentEvents, event)
	if len(a.recentEvents) > 5 {
		a.recentEvents = a.recentEvents[1:]
	}

	oldMood := a.state.CurrentMood
	changed := a.state.ProcessEvent(ctx)

	if changed {
		fmt.Printf("\n[event] %s: %s -> %s\n", event, oldMood, a.state.CurrentMood)
	} else {
		fmt.Printf("\n[event] %s: no mood change (still %s)\n", event, a.state.CurrentMood)
	}

	a.printState()

	// Get action from LLM or fallback
	a.selectAndPrintAction(ctx)
}

func (a *app) selectAndPrintAction(eventCtx personality.EventContext) {
	if a.useLLM && a.engine != nil {
		fmt.Println("  [LLM thinking...]")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := llm.ActionRequest{
			EmotionalState: a.state,
			Event:          eventCtx,
			RecentEvents:   a.recentEvents,
		}

		resp := a.engine.SelectActionWithFallback(ctx, req)
		fmt.Printf("  Koji chooses: %s\n", resp.Action)
		fmt.Printf("  Reason: %s\n", resp.Reason)
	} else {
		// Deterministic fallback
		defaultAction := a.state.SuggestDefaultAction()
		fmt.Printf("  Koji chooses: %s (deterministic)\n", defaultAction.Movement)
		fmt.Printf("  Expression: %s, Sound: %s\n", defaultAction.Expression, defaultAction.Sound)
	}
	fmt.Println()
}

func (a *app) toggleLLM() {
	if a.llmClient == nil {
		fmt.Println("LLM not configured. Restart with Ollama running.")
		return
	}

	a.useLLM = !a.useLLM
	if a.useLLM {
		fmt.Println("LLM enabled")
	} else {
		fmt.Println("LLM disabled (deterministic mode)")
	}
}

func (a *app) printState() {
	fmt.Println()
	fmt.Printf("  Mood:      %s\n", a.state.CurrentMood)
	fmt.Printf("  Intensity: %.1f\n", a.state.Intensity)
	fmt.Printf("  Duration:  %s\n", a.state.Duration().Round(time.Second))
	fmt.Printf("  Baseline:  %v\n", a.state.IsBaseline())
	if a.useLLM {
		fmt.Printf("  LLM:       enabled\n")
	} else {
		fmt.Printf("  LLM:       disabled\n")
	}
	fmt.Println()
}

func (a *app) printActions() {
	actions := a.state.AvailableActions()
	defaultAction := a.state.SuggestDefaultAction()

	fmt.Printf("  Available actions: %v\n", actions)
	fmt.Printf("  Default action:    movement=%s, expression=%s, sound=%s\n",
		defaultAction.Movement, defaultAction.Expression, defaultAction.Sound)
	fmt.Println()
}

func (a *app) printHelp() {
	fmt.Println("Events:")
	fmt.Println("  loud, bang, noise    - loud noise")
	fmt.Println("  music, song          - music playing")
	fmt.Println("  rhythm, beat         - beat detected")
	fmt.Println("  face, familiar, owner - familiar face")
	fmt.Println("  stranger, unknown    - unknown face")
	fmt.Println("  motion, movement     - motion detected")
	fmt.Println("  object, thing, new   - unknown object spotted")
	fmt.Println("  pet, petted          - being petted")
	fmt.Println("  poke, poked          - being poked")
	fmt.Println("  silence, quiet       - silence")
	fmt.Println("  wait, time           - time passes")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  status, s            - show current state")
	fmt.Println("  actions, a           - show available actions")
	fmt.Println("  llm                  - toggle LLM on/off")
	fmt.Println("  help, ?              - show this help")
	fmt.Println("  quit, exit, q        - exit")
	fmt.Println()
	fmt.Println("Add '!' for high intensity (e.g., 'loud!' or 'bang!')")
	fmt.Println()
}

func readInput(ch chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if scanner.Scan() {
			ch <- scanner.Text()
		} else {
			close(ch)
			return
		}
	}
}

func parseEvent(input string) personality.Event {
	// Strip intensity modifiers for matching
	input = strings.ReplaceAll(input, "!", "")
	input = strings.TrimSpace(input)

	switch {
	case contains(input, "loud", "bang", "noise", "crash"):
		return personality.EventLoudNoise
	case contains(input, "music", "song"):
		return personality.EventMusic
	case contains(input, "rhythm", "beat", "bop"):
		return personality.EventRhythm
	case contains(input, "familiar", "owner", "friend"):
		return personality.EventFamiliarFace
	case contains(input, "stranger", "unknown face", "who"):
		return personality.EventUnknownFace
	case contains(input, "face"): // generic face = familiar
		return personality.EventFamiliarFace
	case contains(input, "motion", "movement", "moving"):
		return personality.EventMotionDetected
	case contains(input, "object", "thing", "new", "whats that"):
		return personality.EventUnknownObject
	case contains(input, "pet", "petted", "stroke"):
		return personality.EventPetted
	case contains(input, "poke", "poked", "tap"):
		return personality.EventPoked
	case contains(input, "silence", "quiet", "nothing"):
		return personality.EventSilence
	case contains(input, "wait", "time", "pass"):
		return personality.EventTimePassedLong
	default:
		return ""
	}
}

func contains(input string, options ...string) bool {
	for _, opt := range options {
		if strings.Contains(input, opt) {
			return true
		}
	}
	return false
}
