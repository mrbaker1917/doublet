package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dictionary stores valid words for fast membership checks.
type Dictionary map[string]struct{}

func loadDictionary(path string) (Dictionary, error) {

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dict := make(Dictionary)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		w := normalize(scanner.Text())
		if w == "" {
			continue
		}
		dict[w] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(dict) == 0 {
		return nil, errors.New("dictionary is empty")
	}
	return dict, nil
}

func loadDictionaryForFlags(dictPath, lexicon string) (Dictionary, error) {
	if normalize(dictPath) != "" {
		return loadDictionary(dictPath)
	}

	switch normalize(lexicon) {
	case "small":
		return loadDictionary("words.txt")
	case "large":
		return loadDictionary("words-large.txt")
	default:
		return nil, fmt.Errorf("unknown lexicon %q; use small or large", lexicon)
	}
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isWord(dict Dictionary, word string) bool {
	_, ok := dict[word]
	return ok
}

func oneLetterApart(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	diff := 0
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			diff++
			if diff > 1 {
				return false
			}
		}
	}
	return diff == 1
}

func neighbors(dict Dictionary, word string) []string {
	out := make([]string, 0, 16)
	bytes := []byte(word)
	for i := 0; i < len(bytes); i++ {
		orig := bytes[i]
		for c := byte('a'); c <= byte('z'); c++ {
			if c == orig {
				continue
			}
			bytes[i] = c
			cand := string(bytes)
			if isWord(dict, cand) {
				out = append(out, cand)
			}
		}
		bytes[i] = orig
	}
	return out
}

// shortestPathBFS finds a word ladder with at most maxChanges transitions.
func shortestPathBFS(dict Dictionary, start, end string, maxChanges int) ([]string, bool) {
	if start == end {
		return []string{start}, true
	}
	if maxChanges < 0 {
		return nil, false
	}
	if len(start) != len(end) {
		return nil, false
	}
	unlimited := maxChanges == 0

	type node struct {
		word  string
		steps int
	}

	queue := []node{{word: start, steps: 0}}
	visited := map[string]bool{start: true}
	prev := map[string]string{}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if !unlimited && cur.steps >= maxChanges {
			continue
		}

		for _, nxt := range neighbors(dict, cur.word) {
			if visited[nxt] {
				continue
			}
			visited[nxt] = true
			prev[nxt] = cur.word
			if nxt == end {
				path := []string{end}
				for path[len(path)-1] != start {
					path = append(path, prev[path[len(path)-1]])
				}
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path, true
			}
			queue = append(queue, node{word: nxt, steps: cur.steps + 1})
		}
	}

	return nil, false
}

func printPath(path []string) {
	if len(path) == 0 {
		return
	}
	fmt.Println(strings.Join(path, " -> "))
	fmt.Printf("changes used: %d\n", len(path)-1)
}
