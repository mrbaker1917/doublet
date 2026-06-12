package game

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Dictionary stores valid words for fast membership checks.
type Dictionary map[string]struct{}

func LoadDictionary(path string) (Dictionary, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadDictionaryFromReader(f)
}

func LoadDictionaryFromReader(r io.Reader) (Dictionary, error) {
	dict := make(Dictionary)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		w := Normalize(scanner.Text())
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

func LoadDictionaryForFlags(dictPath, lexicon string) (Dictionary, error) {
	if Normalize(dictPath) != "" {
		return LoadDictionary(dictPath)
	}

	switch Normalize(lexicon) {
	case "small":
		return LoadDictionary("words.txt")
	case "common":
		return LoadDictionary("words-common.txt")
	case "large":
		return LoadDictionary("words-large.txt")
	default:
		return nil, fmt.Errorf("unknown lexicon %q; use small, common, or large", lexicon)
	}
}

func LoadWebDictionaries(dictPath, lexicon string) (common Dictionary, expert Dictionary, err error) {
	common, err = LoadDictionary("words-common.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("load common dictionary: %w", err)
	}

	expert, err = LoadDictionaryForFlags(dictPath, lexicon)
	if err != nil {
		return nil, nil, fmt.Errorf("load expert dictionary: %w", err)
	}

	return common, expert, nil
}

func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func IsWord(dict Dictionary, word string) bool {
	_, ok := dict[word]
	return ok
}
