package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alex/koji/internal/personality"
)

func main() {
	state := personality.NewEmotionalState()

	fmt.Println("=== Koji Emotional State Simulator ===")
	fmt.Println()
	printState(state)
	printHelp()

	// Start decay ticker
	decayTicker := time.NewTicker(1 * time.Second)
	defer decayTicker.Stop()

	// Channel for user input
	inputChan := make(chan string)
	go readInput(inputChan)

	for {
		select {
		case <-decayTicker.C:
			if state.Decay() {
				fmt.Printf("\n[decay] Mood decayed after %s\n", state.Duration().Round(time.Second))
				printState(state)
				fmt.Print("> ")
			}

		case input := <-inputChan:
			input = strings.TrimSpace(strings.ToLower(input))

			if input == "quit" || input == "exit" || input == "q" {
				fmt.Println("Bye!")
				return
			}

			if input == "help" || input == "?" {
				printHelp()
				continue
			}

			if input == "status" || input == "s" {
				printState(state)
				continue
			}

			if input == "actions" || input == "a" {
				printActions(state)
				continue
			}

			event := parseEvent(input)
			if event == "" {
				fmt.Printf("Unknown event: %s (type 'help' for options)\n", input)
				continue
			}

			ctx := personality.NewEventContext(event)

			// Check for intensity modifier
			if strings.Contains(input, "!") || strings.Contains(input, "loud") {
				ctx = ctx.WithIntensity(0.9)
			} else if strings.Contains(input, "soft") || strings.Contains(input, "quiet") {
				ctx = ctx.WithIntensity(0.2)
			}

			oldMood := state.CurrentMood
			changed := state.ProcessEvent(ctx)

			if changed {
				fmt.Printf("\n[event] %s: %s → %s\n", event, oldMood, state.CurrentMood)
			} else {
				fmt.Printf("\n[event] %s: no mood change (still %s)\n", event, state.CurrentMood)
			}

			printState(state)
			printActions(state)
		}
	}
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

func printState(state *personality.EmotionalState) {
	fmt.Println()
	fmt.Printf("  Mood:      %s\n", state.CurrentMood)
	fmt.Printf("  Intensity: %.1f\n", state.Intensity)
	fmt.Printf("  Duration:  %s\n", state.Duration().Round(time.Second))
	fmt.Printf("  Baseline:  %v\n", state.IsBaseline())
	fmt.Println()
}

func printActions(state *personality.EmotionalState) {
	actions := state.AvailableActions()
	defaultAction := state.SuggestDefaultAction()

	fmt.Printf("  Available actions: %v\n", actions)
	fmt.Printf("  Default action:    movement=%s, expression=%s, sound=%s\n",
		defaultAction.Movement, defaultAction.Expression, defaultAction.Sound)
	fmt.Println()
}

func printHelp() {
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
	fmt.Println("  help, ?              - show this help")
	fmt.Println("  quit, exit, q        - exit")
	fmt.Println()
	fmt.Println("Add '!' for high intensity (e.g., 'loud!' or 'bang!')")
	fmt.Println()
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
