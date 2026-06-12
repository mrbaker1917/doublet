package game

import (
	"bufio"
	"crypto/rand"
	"embed"
	"math/big"
	"strings"
)

//go:embed suggestiondata/*.txt
var suggestionFiles embed.FS

var (
	easyDoublets   [][2]string
	mediumDoublets [][2]string
	hardDoublets   [][2]string
)

func init() {
	var err error
	easyDoublets, err = loadSuggestionPairs("suggestiondata/easy.txt")
	if err != nil {
		panic("load easy suggestions: " + err.Error())
	}
	mediumDoublets, err = loadSuggestionPairs("suggestiondata/medium.txt")
	if err != nil {
		panic("load medium suggestions: " + err.Error())
	}
	hardDoublets, err = loadSuggestionPairs("suggestiondata/hard.txt")
	if err != nil {
		panic("load hard suggestions: " + err.Error())
	}
}

func loadSuggestionPairs(path string) ([][2]string, error) {
	data, err := suggestionFiles.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pairs [][2]string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		start, end, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		start = Normalize(start)
		end = Normalize(end)
		if start == "" || end == "" || start == end {
			continue
		}
		pairs = append(pairs, [2]string{start, end})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, errEmptySuggestions
	}
	return pairs, nil
}

func randIntn(n int) int {
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(val.Int64())
}

func GetSuggestedDoublets() ([2]string, [2]string, [2]string) {
	return easyDoublets[randIntn(len(easyDoublets))],
		mediumDoublets[randIntn(len(mediumDoublets))],
		hardDoublets[randIntn(len(hardDoublets))]
}
