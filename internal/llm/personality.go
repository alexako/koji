package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alex/koji/internal/personality"
)

// PersonalityEngine uses an LLM to select actions based on Koji's personality.
type PersonalityEngine struct {
	client *Client
}

// NewPersonalityEngine creates a new personality engine with the given LLM client.
func NewPersonalityEngine(client *Client) *PersonalityEngine {
	return &PersonalityEngine{client: client}
}

// ActionRequest contains all context needed for the LLM to pick an action.
type ActionRequest struct {
	EmotionalState *personality.EmotionalState
	Event          personality.EventContext
	RecentEvents   []personality.Event // last few events for context
}

// ActionResponse is what we expect back from the LLM.
type ActionResponse struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

const systemPrompt = `You are Koji, a small robot pet with a curious, excitable personality.

Personality traits:
- Curious by nature, easily excited by new things
- A little clumsy but enthusiastic
- Loves music, bobs head and wags tail
- Startled by loud noises, hides then peeks out cautiously
- Wary of strangers at first, but warms up quickly
- Gets sleepy when quiet for too long
- Affectionate with familiar people

You are NOT a helpful assistant. You are a pet. You don't answer questions or provide information. You react to your environment like an animal would.

IMPORTANT: You must respond with ONLY valid JSON in this exact format:
{"action": "<action_from_list>", "reason": "<brief 5-10 word reason>"}

Do not include any other text, explanation, or markdown. Just the JSON object.`

// buildPrompt constructs the full prompt for action selection.
func (e *PersonalityEngine) buildPrompt(req ActionRequest) string {
	var sb strings.Builder

	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n")

	// Current emotional state
	sb.WriteString("Current state:\n")
	sb.WriteString(fmt.Sprintf("- Mood: %s\n", req.EmotionalState.CurrentMood))
	sb.WriteString(fmt.Sprintf("- Intensity: %.1f (0=mild, 1=intense)\n", req.EmotionalState.Intensity))
	sb.WriteString(fmt.Sprintf("- Time in mood: %s\n", req.EmotionalState.Duration().Round(time.Second)))

	// Recent events for context
	if len(req.RecentEvents) > 0 {
		sb.WriteString(fmt.Sprintf("- Recent events: %v\n", req.RecentEvents))
	}

	sb.WriteString("\n")

	// Available actions for this mood
	actions := req.EmotionalState.AvailableActions()
	actionStrs := make([]string, len(actions))
	for i, a := range actions {
		actionStrs[i] = string(a)
	}
	sb.WriteString(fmt.Sprintf("Available actions: [%s]\n\n", strings.Join(actionStrs, ", ")))

	// Current event
	sb.WriteString(fmt.Sprintf("Event just detected: %s", req.Event.Event))
	if req.Event.Intensity > 0.7 {
		sb.WriteString(" (intense)")
	} else if req.Event.Intensity < 0.3 {
		sb.WriteString(" (mild)")
	}
	if req.Event.Source != "" {
		sb.WriteString(fmt.Sprintf(" from %s", req.Event.Source))
	}
	sb.WriteString("\n\n")

	sb.WriteString("Choose ONE action from the list. Respond with JSON only.")

	return sb.String()
}

// SelectAction asks the LLM to pick an action given the current context.
func (e *PersonalityEngine) SelectAction(ctx context.Context, req ActionRequest) (*ActionResponse, error) {
	prompt := e.buildPrompt(req)

	response, err := e.client.GenerateJSON(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generating response: %w", err)
	}

	// Parse the JSON response
	var actionResp ActionResponse
	if err := json.Unmarshal([]byte(response), &actionResp); err != nil {
		// Try to extract JSON if there's extra text
		cleaned := extractJSON(response)
		if err := json.Unmarshal([]byte(cleaned), &actionResp); err != nil {
			return nil, fmt.Errorf("parsing response %q: %w", response, err)
		}
	}

	// Validate the action is in the available set
	available := req.EmotionalState.AvailableActions()
	valid := false
	for _, a := range available {
		if string(a) == actionResp.Action {
			valid = true
			break
		}
	}

	if !valid {
		// Fall back to default action for this mood
		defaultAction := req.EmotionalState.SuggestDefaultAction()
		return &ActionResponse{
			Action: string(defaultAction.Movement),
			Reason: "fallback - LLM chose invalid action",
		}, nil
	}

	return &actionResp, nil
}

// extractJSON tries to find a JSON object in a string that might have extra text.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// SelectActionWithFallback tries LLM first, falls back to defaults on error.
func (e *PersonalityEngine) SelectActionWithFallback(ctx context.Context, req ActionRequest) ActionResponse {
	resp, err := e.SelectAction(ctx, req)
	if err != nil {
		// LLM failed, use deterministic fallback
		defaultAction := req.EmotionalState.SuggestDefaultAction()
		return ActionResponse{
			Action: string(defaultAction.Movement),
			Reason: fmt.Sprintf("fallback: %v", err),
		}
	}
	return *resp
}
