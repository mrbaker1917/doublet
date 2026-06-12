package game

import (
	"bufio"
	"crypto/rand"
	"embed"
	"math/big"
	"strings"
)

//go:embed suggestiondata/*.txt suggestiondata/common/*.txt
var suggestionFiles embed.FS

type suggestionPools struct {
	easy   [][2]string
	medium [][2]string
	hard   [][2]string
}

var (
	expertSuggestions suggestionPools
	commonSuggestions suggestionPools
)

func init() {
	var err error
	expertSuggestions, err = loadSuggestionPools("suggestiondata")
	if err != nil {
		panic("load expert suggestions: " + err.Error())
	}
	commonSuggestions, err = loadSuggestionPools("suggestiondata/common")
	if err != nil {
		panic("load common suggestions: " + err.Error())
	}
}

func loadSuggestionPools(dir string) (suggestionPools, error) {
	easy, err := loadSuggestionPairs(dir + "/easy.txt")
	if err != nil {
		return suggestionPools{}, err
	}
	medium, err := loadSuggestionPairs(dir + "/medium.txt")
	if err != nil {
		return suggestionPools{}, err
	}
	hard, err := loadSuggestionPairs(dir + "/hard.txt")
	if err != nil {
		return suggestionPools{}, err
	}
	return suggestionPools{easy: easy, medium: medium, hard: hard}, nil
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

func GetSuggestedDoublets(expert bool) ([2]string, [2]string, [2]string) {
	pools := commonSuggestions
	if expert {
		pools = expertSuggestions
	}
	return pools.easy[randIntn(len(pools.easy))],
		pools.medium[randIntn(len(pools.medium))],
		pools.hard[randIntn(len(pools.hard))]
}
