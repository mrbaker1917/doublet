package main

import (
	"bufio"
	"doublet/internal/game"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	source := flag.String("source", "words-large.txt", "full dictionary to filter")
	output := flag.String("output", "words-common.txt", "output common-word list")
	allowlist := flag.String("allowlist", "internal/game/wordlists/allowlist.txt", "known common English words")
	supplement := flag.String("supplement", "internal/game/wordlists/common-supplement.txt", "extra common words not in the allowlist")
	excluded := flag.String("excluded", "internal/game/suggestiondata/common-excluded.txt", "words to exclude from common")
	extra := flag.String("extra", "words.txt", "extra words to always include")
	seedsDir := flag.String("seeds", "internal/game/suggestiondata", "directory with *.seeds bridge pair files")
	flag.Parse()

	large, err := game.LoadDictionary(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load source: %v\n", err)
		os.Exit(1)
	}

	allowed, err := loadWordList(*allowlist)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load allowlist: %v\n", err)
		os.Exit(1)
	}

	supplementWords, err := loadWordList(*supplement)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load supplement: %v\n", err)
		os.Exit(1)
	}

	blocked, err := loadWordList(*excluded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load excluded: %v\n", err)
		os.Exit(1)
	}

	common := make(game.Dictionary)

	addWord := func(word string) {
		word = game.Normalize(word)
		if word == "" || isBlocked(word, blocked) {
			return
		}
		if !game.IsPlayableWord(word) || len(word) < 3 || len(word) > 5 {
			return
		}
		if !game.IsWord(large, word) {
			return
		}
		common[word] = struct{}{}
	}

	for word := range allowed {
		addWord(word)
	}

	for word := range supplementWords {
		addWord(word)
	}

	if *extra != "" {
		extraDict, err := game.LoadDictionary(*extra)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load extra: %v\n", err)
			os.Exit(1)
		}
		for word := range extraDict {
			addWord(word)
		}
	}

	playable := playableWords(large)
	seedPairs, err := loadSeedPairs(*seedsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load seeds: %v\n", err)
		os.Exit(1)
	}

	for _, pair := range seedPairs {
		path, ok := game.ShortestPathBFS(playable, pair[0], pair[1], 0)
		if !ok {
			continue
		}
		for _, word := range path {
			addWord(word)
		}
	}

	expandCommonNeighbors(common, playable, blocked, addWord)

	words := make([]string, 0, len(common))
	for word := range common {
		words = append(words, word)
	}
	sort.Strings(words)

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, word := range words {
		if _, err := w.WriteString(word + "\n"); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d common words to %s (from %d source words, %d allowlist, %d supplement, %d seed pairs)\n",
		len(words), *output, len(large), len(allowed), len(supplementWords), len(seedPairs))
}

// expandCommonNeighbors adds playable bridge words that connect to two or more words
// already in the common dictionary.
func expandCommonNeighbors(common, playable game.Dictionary, blocked map[string]struct{}, addWord func(string)) {
	const minCommonNeighbors = 2

	changed := true
	for changed {
		changed = false
		for word := range playable {
			if isBlocked(word, blocked) {
				continue
			}
			if _, ok := common[word]; ok {
				continue
			}

			neighbors := 0
			for _, next := range game.Neighbors(playable, word) {
				if _, ok := common[next]; ok {
					neighbors++
					if neighbors >= minCommonNeighbors {
						break
					}
				}
			}
			if neighbors < minCommonNeighbors {
				continue
			}

			before := len(common)
			addWord(word)
			if len(common) > before {
				changed = true
			}
		}
	}
}

func playableWords(large game.Dictionary) game.Dictionary {
	out := make(game.Dictionary)
	for word := range large {
		if len(word) < 3 || len(word) > 5 {
			continue
		}
		if !game.IsPlayableWord(word) {
			continue
		}
		out[word] = struct{}{}
	}
	return out
}

func loadWordList(path string) (map[string]struct{}, error) {
	words := make(map[string]struct{})
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return words, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := game.Normalize(scanner.Text())
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		words[word] = struct{}{}
	}
	return words, scanner.Err()
}

func loadSeedPairs(dir string) ([][2]string, error) {
	var pairs [][2]string
	for _, name := range []string{"easy.seeds", "medium.seeds", "hard.seeds"} {
		path := filepath.Join(dir, name)
		filePairs, err := readSeedFile(path)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, filePairs...)
	}
	return pairs, nil
}

func readSeedFile(path string) ([][2]string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var pairs [][2]string
	for line := range strings.Lines(string(data)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		start, end, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		start = game.Normalize(start)
		end = game.Normalize(end)
		if start != "" && end != "" && start != end {
			pairs = append(pairs, [2]string{start, end})
		}
	}
	return pairs, nil
}

func isBlocked(word string, blocked map[string]struct{}) bool {
	_, ok := blocked[word]
	return ok
}
