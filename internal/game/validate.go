package game

import (
	"fmt"
	"strings"
)

const (
	// MaxCustomChanges is the absolute ceiling for custom difficulty.
	MaxCustomChanges = 100
	// MaxCustomExtra is how far above the shortest path custom max may go.
	MaxCustomExtra = 10
)

func ValidateInputs(dict Dictionary, start, end string) error {
	if start == "" || end == "" {
		return fmt.Errorf("start and target words are required")
	}
	if len(start) != len(end) {
		return fmt.Errorf("start and target must have same length")
	}
	if !IsWord(dict, start) {
		return fmt.Errorf("ERROR: start word %q is NOT in dictionary", start)
	}
	if !IsWord(dict, end) {
		return fmt.Errorf("ERROR: target word %q is NOT in dictionary", end)
	}
	return nil
}

func NormalizeDifficulty(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func ValidateDifficulty(difficulty string) error {
	switch difficulty {
	case "easy", "medium", "hard", "custom":
		return nil
	default:
		return fmt.Errorf("difficulty must be one of: easy, medium, hard, custom")
	}
}

func ResolveMaxChanges(difficulty string, requestedMax, shortestChanges int) (int, error) {
	switch difficulty {
	case "easy":
		return shortestChanges + easySlack(shortestChanges), nil
	case "medium":
		return shortestChanges + mediumSlack(shortestChanges), nil
	case "hard":
		return shortestChanges, nil
	case "custom":
		if requestedMax <= 0 {
			return 0, fmt.Errorf("custom difficulty requires max to be set")
		}
		if requestedMax < shortestChanges {
			return 0, fmt.Errorf("max changes must be at least %d for this pair", shortestChanges)
		}

		cap := shortestChanges + MaxCustomExtra
		if cap > MaxCustomChanges {
			cap = MaxCustomChanges
		}
		if requestedMax > cap {
			return 0, fmt.Errorf("max changes cannot exceed %d for custom difficulty", cap)
		}

		return requestedMax, nil
	default:
		return 0, fmt.Errorf("unsupported difficulty: %s", difficulty)
	}
}

func CustomMaxChangesCap(shortestChanges int) int {
	cap := shortestChanges + MaxCustomExtra
	if cap > MaxCustomChanges {
		return MaxCustomChanges
	}
	return cap
}

// easySlack scales extra moves for easy difficulty by puzzle length.
func easySlack(shortestChanges int) int {
	switch {
	case shortestChanges <= 1:
		return 1
	case shortestChanges <= 4:
		return 2
	default:
		return 3
	}
}

// mediumSlack scales extra moves for medium difficulty by puzzle length.
func mediumSlack(shortestChanges int) int {
	switch {
	case shortestChanges <= 1:
		return 0
	case shortestChanges <= 4:
		return 1
	default:
		return 2
	}
}
