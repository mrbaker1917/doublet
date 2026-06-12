package main

import (
	"crypto/rand"
	"doublet/internal/game"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type bucket struct {
	wordLen  int
	minDist  int
	maxDist  int
	target   int
	maxUses  int
	outFile  string
	seedFile string
}

func main() {
	dictPath := flag.String("dict", "words-large.txt", "dictionary file")
	outDir := flag.String("out", "internal/game/suggestiondata", "output directory for suggestion lists")
	pool := flag.String("pool", "", "optional subdirectory under out (e.g. common)")
	flag.Parse()

	if *pool != "" {
		*outDir = filepath.Join(*outDir, *pool)
	}

	blocked := loadBlocked(filepath.Join(*outDir, "blocked.txt"))
	if len(blocked) == 0 {
		blocked = loadBlocked(filepath.Join(filepath.Dir(*outDir), "blocked.txt"))
	}

	dict, err := game.LoadDictionary(*dictPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load dictionary: %v\n", err)
		os.Exit(1)
	}

	buckets := []bucket{
		{wordLen: 3, minDist: 2, maxDist: 5, target: 200, maxUses: 6, outFile: "easy.txt", seedFile: "easy.seeds"},
		{wordLen: 4, minDist: 3, maxDist: 7, target: 150, maxUses: 5, outFile: "medium.txt", seedFile: "medium.seeds"},
		{wordLen: 5, minDist: 5, maxDist: 12, target: 80, maxUses: 4, outFile: "hard.txt", seedFile: "hard.seeds"},
	}
	if *pool == "common" {
		buckets = []bucket{
			{wordLen: 3, minDist: 2, maxDist: 5, target: 200, maxUses: 6, outFile: "easy.txt", seedFile: "easy.seeds"},
			{wordLen: 4, minDist: 3, maxDist: 7, target: 150, maxUses: 5, outFile: "medium.txt", seedFile: "medium.seeds"},
			{wordLen: 5, minDist: 3, maxDist: 7, target: 50, maxUses: 4, outFile: "hard.txt", seedFile: "hard.seeds"},
		}
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	for _, b := range buckets {
		seeds := loadSeeds(filepath.Join(*outDir, b.seedFile))
		if len(seeds) == 0 {
			seeds = loadSeeds(filepath.Join(filepath.Dir(*outDir), b.seedFile))
		}
		pairs := collectPairs(dict, b, seeds, blocked, *pool == "common")
		outPath := filepath.Join(*outDir, b.outFile)
		if err := writePairs(outPath, pairs); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %d pairs to %s\n", len(pairs), outPath)
	}
}

func loadBlocked(path string) map[string]struct{} {
	blocked := make(map[string]struct{})
	data, err := os.ReadFile(path)
	if err != nil {
		return blocked
	}
	for line := range strings.Lines(string(data)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		word := game.Normalize(line)
		if word != "" {
			blocked[word] = struct{}{}
		}
	}
	return blocked
}

func loadSeeds(path string) [][2]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var seeds [][2]string
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
			seeds = append(seeds, [2]string{start, end})
		}
	}
	return seeds
}

func collectPairs(dict game.Dictionary, b bucket, seeds [][2]string, blocked map[string]struct{}, relaxed bool) [][2]string {
	words := candidateWords(dict, b.wordLen, seeds, blocked, relaxed)
	if len(words) < 2 {
		return nil
	}

	fmt.Fprintf(os.Stderr, "%s: %d candidate words\n", b.outFile, len(words))

	seen := make(map[string]struct{})
	startUses := make(map[string]int)
	endUses := make(map[string]int)
	var pairs [][2]string

	add := func(start, end string) bool {
		if start == end {
			return false
		}
		if isBlocked(start, blocked) || isBlocked(end, blocked) {
			return false
		}
		if !game.IsPlayableWord(start) || !game.IsPlayableWord(end) {
			return false
		}
		key := pairKey(start, end)
		if _, ok := seen[key]; ok {
			return false
		}
		if startUses[start] >= b.maxUses || endUses[end] >= b.maxUses {
			return false
		}
		path, ok := game.ShortestPathBFS(dict, start, end, 0)
		if !ok {
			return false
		}
		dist := len(path) - 1
		if dist < b.minDist || dist > b.maxDist {
			return false
		}
		seen[key] = struct{}{}
		startUses[start]++
		endUses[end]++
		pairs = append(pairs, [2]string{start, end})
		return true
	}

	for _, seed := range seeds {
		if len(pairs) >= b.target {
			break
		}
		add(seed[0], seed[1])
	}

	maxAttempts := b.target * 500
	for len(pairs) < b.target && maxAttempts > 0 {
		maxAttempts--
		start := words[randIntn(len(words))]
		end := words[randIntn(len(words))]
		add(start, end)
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] == pairs[j][0] {
			return pairs[i][1] < pairs[j][1]
		}
		return pairs[i][0] < pairs[j][0]
	})
	return pairs
}

func minNeighbors(wordLen int, relaxed bool) int {
	if relaxed {
		switch wordLen {
		case 3:
			return 8
		case 4:
			return 6
		default:
			return 4
		}
	}
	switch wordLen {
	case 3:
		return 12
	case 4:
		return 10
	default:
		return 8
	}
}

func candidateWords(dict game.Dictionary, wordLen int, seeds [][2]string, blocked map[string]struct{}, relaxed bool) []string {
	candidates := make(map[string]struct{})

	addWord := func(word string) {
		if len(word) != wordLen {
			return
		}
		candidates[word] = struct{}{}
	}

	for _, pair := range seeds {
		for _, word := range pair {
			addWord(word)
		}
		path, ok := game.ShortestPathBFS(dict, pair[0], pair[1], 0)
		if !ok {
			continue
		}
		for _, word := range path {
			addWord(word)
		}
	}

	words := make([]string, 0, len(candidates))
	for word := range candidates {
		if !game.IsPlayableWord(word) || isBlocked(word, blocked) {
			continue
		}
		if len(game.Neighbors(dict, word)) < minNeighbors(wordLen, relaxed) {
			continue
		}
		words = append(words, word)
	}
	sort.Strings(words)
	return words
}

func isBlocked(word string, blocked map[string]struct{}) bool {
	_, ok := blocked[word]
	return ok
}

func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

func randIntn(n int) int {
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(val.Int64())
}

func writePairs(path string, pairs [][2]string) error {
	var b strings.Builder
	b.WriteString("# Generated by: go run ./cmd/seedpairs\n")
	b.WriteString("# Format: start,target\n")
	for _, pair := range pairs {
		b.WriteString(pair[0])
		b.WriteString(",")
		b.WriteString(pair[1])
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
