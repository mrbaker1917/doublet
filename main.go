package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
)

func randIntn(n int) int {
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(val.Int64())
}

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

	// Suggest doublets at three difficulty levels if user hasn't specified start/end
	if *startFlag == "" && *endFlag == "" {
		fmt.Println("\n=== WELCOME TO DOUBLET ===")
		fmt.Println("Here are some doublets to try:")
		easy, medium, hard := getSuggestedDoublets()
		fmt.Printf("          %-10s %s\n", "start", "target")
		fmt.Printf("          %-10s %s\n", "-----", "------")
		fmt.Printf("  Easy:   %-10q → %q\n", easy[0], easy[1])
		fmt.Printf("  Medium: %-10q → %q\n", medium[0], medium[1])
		fmt.Printf("  Hard:   %-10q → %q\n\n", hard[0], hard[1])
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

	reader := bufio.NewReader(os.Stdin)
	for {
		playGame(dict, start, end, maxChanges, shortest, reader)
		fmt.Print("\nPlay again with new words? (y/n): ")
		ans, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) != "y" {
			fmt.Println("Thanks for playing Doublet! Goodbye!")
			return
		}
		// pick new words for next round
		if *startFlag == "" && *endFlag == "" {
			fmt.Println("\n=== WELCOME TO DOUBLET ===")
			fmt.Println("Here are some doublets to try:")
			easy, medium, hard := getSuggestedDoublets()
			fmt.Printf("          %-10s %s\n", "start", "target")
			fmt.Printf("          %-10s %s\n", "-----", "------")
			fmt.Printf("  Easy:   %-10q → %q\n", easy[0], easy[1])
			fmt.Printf("  Medium: %-10q → %q\n", medium[0], medium[1])
			fmt.Printf("  Hard:   %-10q → %q\n\n", hard[0], hard[1])
		}
		newStart, newEnd, newDifficulty, newMax, ok := gatherInputs("", "", *difficultyFlag, *maxFlag)
		if !ok {
			return
		}
		if err := validateInputs(dict, newStart, newEnd); err != nil {
			fmt.Println("input error:", err)
			return
		}
		shortest, found = shortestPathBFS(dict, newStart, newEnd, 0)
		if !found {
			fmt.Printf("no path found from %q to %q with current dictionary\n", newStart, newEnd)
			return
		}
		newMax, err = resolveMaxChanges(newDifficulty, newMax, len(shortest)-1)
		if err != nil {
			fmt.Println("input error:", err)
			return
		}
		start, end, maxChanges = newStart, newEnd, newMax
	}
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
		return fmt.Errorf("ERROR: start word %q is NOT in dictionary", start)
	}
	if !isWord(dict, end) {
		return fmt.Errorf("ERROR: target word %q is NOT in dictionary", end)
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

var mediumDoublets = [][2]string{
	{"cold", "warm"},
	{"hand", "foot"},
	{"head", "tail"},
	{"more", "less"},
	{"dark", "dawn"},
	{"four", "five"},
	{"hate", "love"},
	{"fire", "hide"},
	{"ring", "song"},
	{"swim", "flew"},
	{"wine", "beer"},
	{"work", "play"},
	{"left", "mine"},
	{"hunt", "fish"},
	{"word", "game"},
}

var hardDoublets = [][2]string{
	{"stone", "money"},
	{"witch", "bride"},
	{"black", "white"},
	{"blood", "sweat"},
	{"bread", "toast"},
	{"floor", "glass"},
	{"night", "light"},
	{"grass", "green"},
	{"chain", "break"},
	{"sleep", "dream"},
}

var easyDoublets = [][2]string{
	{"cat", "dog"},
	{"hit", "hot"},
	{"bat", "cat"},
	{"rat", "bat"},
	{"hat", "cat"},
	{"bit", "bat"},
	{"pit", "pat"},
	{"pat", "cat"},
	{"sit", "sat"},
	{"sat", "cat"},
	{"mat", "hat"},
	{"fat", "cat"},
	{"bed", "bad"},
	{"red", "bed"},
	{"fed", "bed"},
	{"led", "bed"},
	{"men", "pen"},
	{"hen", "pen"},
	{"ten", "pen"},
	{"den", "pen"},
	{"big", "bag"},
	{"dig", "dog"},
	{"fog", "dog"},
	{"log", "dog"},
	{"cot", "cat"},
	{"cut", "cat"},
	{"cup", "cap"},
	{"cap", "cat"},
	{"car", "cat"},
	{"can", "cat"},
}

func getSuggestedDoublets() ([2]string, [2]string, [2]string) {
	return easyDoublets[randIntn(len(easyDoublets))],
		mediumDoublets[randIntn(len(mediumDoublets))],
		hardDoublets[randIntn(len(hardDoublets))]
}

// playGame runs one round.
func playGame(dict Dictionary, start, end string, maxChanges int, solution []string, reader *bufio.Reader) {
	fmt.Println("\nHere is your Doublet Challenge:")
	fmt.Printf("turn %q into %q in at most %d changes\n", start, end, maxChanges)
	fmt.Println("rules: change exactly one letter each move and keep valid words")
	fmt.Println("commands: `/restart`, `/quit`")

	current := start
	moves := 0

	for {
		if current == end {
			fmt.Printf("\nCONGRATULATIONS! You solved in %d/%d changes!\n", moves, maxChanges)
			return
		}
		if moves >= maxChanges {
			fmt.Printf("\nNO MOVES LEFT — the target was %q. better luck next time!\n", end)
			fmt.Println("ONE VALID PATH WAS:")
			printPath(solution)
			return
		}

		remaining := maxChanges - moves
		fmt.Printf("current: %s | target: %s | remaining changes: %d\n", current, strings.ToUpper(end), remaining)
		fmt.Print("next word: ")
		line, _ := reader.ReadString('\n')
		next := normalize(line)

		switch next {
		case "":
			fmt.Println("Enter a word or type `/restart` to start over, `/quit` to exit")
			continue
		case "/restart":
			fmt.Println("Restarting round...")
			current = start
			moves = 0
			fmt.Printf("turn %q into %q in at most %d changes\n", start, end, maxChanges)
			continue
		case "/quit":
			fmt.Println("Thanks for playing Doublet! Goodbye!")
			os.Exit(0)
		}

		if len(next) != len(current) {
			fmt.Printf("word must be %d letters\n", len(current))
			continue
		}
		if !isWord(dict, next) {
			fmt.Printf("ERROR: %q is NOT in dictionary\n", next)
			continue
		}
		if !oneLetterApart(current, next) {
			fmt.Println("ERROR: You must change exactly, only one letter")
			continue
		}

		current = next
		moves++
	}
}
