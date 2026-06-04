package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	dictPath := flag.String("dict", "", "path to newline-separated dictionary (overrides -lexicon)")
	lexicon := flag.String("lexicon", "large", "dictionary preset: small or large")
	startFlag := flag.String("start", "", "starting word")
	endFlag := flag.String("end", "", "target word")
	difficultyFlag := flag.String("difficulty", "medium", "difficulty: easy, medium, hard, custom")
	maxFlag := flag.Int("max", 0, "maximum allowed letter changes")
	showOnly := flag.Bool("solve", false, "print shortest solution and exit")
	flag.Parse()

	dict, err := loadDictionaryForFlags(*dictPath, *lexicon)
	if err != nil {
		fmt.Printf("failed to load dictionary: %v\n", err)
		os.Exit(1)
	}

	start, end, difficulty, requestedMax, ok := gatherInputs(*startFlag, *endFlag, *difficultyFlag, *maxFlag)
	if !ok {
		os.Exit(1)
	}

	if err := validateInputs(dict, start, end); err != nil {
		fmt.Println("input error:", err)
		os.Exit(1)
	}
	if err := validateDifficulty(difficulty); err != nil {
		fmt.Println("input error:", err)
		os.Exit(1)
	}

	shortest, found := shortestPathBFS(dict, start, end, 0)
	if !found {
		fmt.Printf("no path found from %q to %q with current dictionary\n", start, end)
		os.Exit(1)
	}

	shortestChanges := len(shortest) - 1
	maxChanges, err := resolveMaxChanges(difficulty, requestedMax, shortestChanges)
	if err != nil {
		fmt.Println("input error:", err)
		os.Exit(1)
	}
	if shortestChanges > maxChanges {
		fmt.Printf("no path found from %q to %q in %d or fewer changes\n", start, end, maxChanges)
		os.Exit(1)
	}

	if *showOnly {
		fmt.Printf("difficulty: %s | max changes: %d\n", difficulty, maxChanges)
		fmt.Println("shortest path:")
		printPath(shortest)
		return
	}

	playGame(dict, start, end, maxChanges)
}

func gatherInputs(start, end, difficulty string, maxChanges int) (string, string, string, int, bool) {
	start = normalize(start)
	end = normalize(end)
	difficulty = normalizeDifficulty(difficulty)

	reader := bufio.NewReader(os.Stdin)
	if start == "" {
		fmt.Print("start word: ")
		s, _ := reader.ReadString('\n')
		start = normalize(s)
	}
	if end == "" {
		fmt.Print("target word: ")
		s, _ := reader.ReadString('\n')
		end = normalize(s)
	}
	if difficulty == "" {
		fmt.Print("difficulty (easy/medium/hard/custom): ")
		s, _ := reader.ReadString('\n')
		difficulty = normalizeDifficulty(s)
	}
	if difficulty == "custom" && maxChanges <= 0 {
		fmt.Print("max changes: ")
		s, _ := reader.ReadString('\n')
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n <= 0 {
			fmt.Println("max changes must be a positive integer")
			return "", "", "", 0, false
		}
		maxChanges = n
	}

	return start, end, difficulty, maxChanges, true
}

func validateInputs(dict Dictionary, start, end string) error {
	if start == "" || end == "" {
		return fmt.Errorf("start and target words are required")
	}
	if len(start) != len(end) {
		return fmt.Errorf("start and target must have same length")
	}
	if !isWord(dict, start) {
		return fmt.Errorf("start word %q is not in dictionary", start)
	}
	if !isWord(dict, end) {
		return fmt.Errorf("target word %q is not in dictionary", end)
	}
	return nil
}

func normalizeDifficulty(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validateDifficulty(difficulty string) error {
	switch difficulty {
	case "easy", "medium", "hard", "custom":
		return nil
	default:
		return fmt.Errorf("difficulty must be one of: easy, medium, hard, custom")
	}
}

func resolveMaxChanges(difficulty string, requestedMax, shortestChanges int) (int, error) {
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

func playGame(dict Dictionary, start, end string, maxChanges int) {
	fmt.Println("doublet challenge")
	fmt.Printf("turn %q into %q in at most %d changes\n", start, end, maxChanges)
	fmt.Println("rules: change exactly one letter each move and keep valid words")
	fmt.Println("commands: /quit")

	reader := bufio.NewReader(os.Stdin)
	current := start
	moves := 0

	for {
		if current == end {
			fmt.Printf("you solved it in %d changes\n", moves)
			return
		}
		if moves >= maxChanges {
			fmt.Printf("limit reached (%d changes).\n", maxChanges)
			fmt.Println("challenge failed. try again!")
			return
		}

		remaining := maxChanges - moves
		fmt.Printf("current: %s | remaining changes: %d\n", current, remaining)
		fmt.Print("next word: ")
		line, _ := reader.ReadString('\n')
		next := normalize(line)

		switch next {
		case "":
			fmt.Println("enter a word or command")
			continue
		case "/quit":
			fmt.Println("game ended")
			return
		}

		if len(next) != len(current) {
			fmt.Printf("word must be %d letters\n", len(current))
			continue
		}
		if !isWord(dict, next) {
			fmt.Println("not in dictionary")
			continue
		}
		if !oneLetterApart(current, next) {
			fmt.Println("must change exactly one letter")
			continue
		}

		current = next
		moves++
	}
}
