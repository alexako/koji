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
	variation    *personality.VariationEngine
	llmClient    *llm.Client
	engine       *llm.PersonalityEngine
	recentEvents []personality.Event
	useLLM       bool
}

func main() {
	// Flags
	ollamaURL := flag.String("ollama", "http://localhost:11434", "Ollama API URL")
	model := flag.String("model", "phi3:mini", "LLM model to use")
	noLLM := flag.Bool("no-llm", false, "Disable LLM, use only deterministic actions")
	flag.Parse()

	app := &app{
		state:        personality.NewEmotionalState(),
		variation:    personality.NewVariationEngine(),
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
			// Check if the model is available
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			found, available, err := app.llmClient.CheckModel(ctx)
			cancel()

			if err != nil {
				fmt.Printf("Warning: Could not check models: %v\n", err)
				fmt.Println("Continuing anyway...")
			} else if !found {
				fmt.Printf("Warning: Model '%s' not found.\n", *model)
				fmt.Printf("Available models: %v\n", available)
				fmt.Printf("Install with: ollama pull %s\n", *model)
				fmt.Println()
				fmt.Println("Running with variation engine (weighted random + mood echoes)")
				fmt.Println()
				app.useLLM = false
			}

			if app.useLLM {
				app.engine = llm.NewPersonalityEngine(app.llmClient)
				fmt.Printf("Connected to Ollama (model: %s)\n", *model)
				fmt.Println("LLM will select actions based on personality.")
				fmt.Println()
			}
		}
	} else {
		fmt.Println("Running with variation engine (weighted random + mood echoes)")
		fmt.Println()
	}

	app.printState()
	app.printHelp()

	// Start decay ticker
	decayTicker := time.NewTicker(1 * time.Second)
	defer decayTicker.Stop()

	// Micro-behavior ticker (idle animations)
	microTicker := time.NewTicker(3 * time.Second)
	defer microTicker.Stop()

	// Channel for user input
	inputChan := make(chan string)
	go readInput(inputChan)

	for {
		select {
		case <-decayTicker.C:
			if app.state.Decay() {
				// Record the implicit mood change for echoes
				fmt.Printf("\n[decay] Mood decayed after %s\n", app.state.Duration().Round(time.Second))
				app.printState()
				fmt.Print("> ")
			}

		case <-microTicker.C:
			// Occasional idle micro-behaviors make Koji feel alive
			if !app.useLLM {
				micro := app.variation.SelectMicroBehavior(app.state.CurrentMood)
				if micro != nil {
					fmt.Printf("\n[idle] *%s*\n", micro.Name)
					fmt.Print("> ")
				}
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
		// Record the mood change for echo effects
		a.variation.RecordMoodChange(oldMood)
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
		// Use variation engine for lifelike behavior
		action := a.variation.SelectAction(a.state)
		fmt.Printf("  Koji chooses: %s (%s)\n", action.Action, action.Modifier)

		// Show any active mood echoes affecting behavior
		echoes := a.variation.GetActiveEchoes()
		if len(echoes) > 0 {
			for _, echo := range echoes {
				fmt.Printf("  [echo] still affected by %s (%.0f%% strength)\n",
					echo.FromMood, echo.Strength*100)
			}
		}

		// Maybe do a micro-behavior too
		micro := a.variation.SelectMicroBehavior(a.state.CurrentMood)
		if micro != nil {
			fmt.Printf("  [micro] %s (%dms)\n", micro.Name, micro.Duration.Milliseconds())
		}
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

	// Show active mood echoes
	echoes := a.variation.GetActiveEchoes()
	if len(echoes) > 0 {
		fmt.Printf("  Echoes:    ")
		for i, echo := range echoes {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s (%.0f%%)", echo.FromMood, echo.Strength*100)
		}
		fmt.Println()
	}

	if a.useLLM {
		fmt.Printf("  Mode:      LLM\n")
	} else {
		fmt.Printf("  Mode:      variation engine\n")
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
	fmt.Println("  status, s            - show current state (includes mood echoes)")
	fmt.Println("  actions, a           - show available actions")
	fmt.Println("  llm                  - toggle LLM on/off (variation engine is default)")
	fmt.Println("  help, ?              - show this help")
	fmt.Println("  quit, exit, q        - exit")
	fmt.Println()
	fmt.Println("Add '!' for high intensity (e.g., 'loud!' or 'bang!')")
	fmt.Println()
	fmt.Println("Koji will show idle micro-behaviors every few seconds.")
	fmt.Println("Past moods leave 'echoes' that affect current behavior.")
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
