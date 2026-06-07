package game

import (
	"fmt"
	"strings"
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
	if requestedMax > 0 {
		return requestedMax, nil
	}

	switch difficulty {
	case "easy":
		return shortestChanges + 2, nil
	case "medium":
		return shortestChanges + 1, nil
	case "hard":
		return shortestChanges, nil
	case "custom":
		return 0, fmt.Errorf("custom difficulty requires -max to be set")
	default:
		return 0, fmt.Errorf("unsupported difficulty: %s", difficulty)
	}
}
