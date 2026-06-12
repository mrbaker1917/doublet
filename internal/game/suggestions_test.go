package game

import (
	"testing"
)

func TestSuggestionPairsAreValid(t *testing.T) {
	dict, err := LoadDictionary("../../words-large.txt")
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}

	check := func(name string, pairs [][2]string, minLen, maxLen int) {
		t.Helper()
		if len(pairs) == 0 {
			t.Fatalf("%s: no pairs loaded", name)
		}
		for _, pair := range pairs {
			start, end := pair[0], pair[1]
			if len(start) < minLen || len(start) > maxLen {
				t.Fatalf("%s pair %q->%q: unexpected start length", name, start, end)
			}
			if len(end) < minLen || len(end) > maxLen {
				t.Fatalf("%s pair %q->%q: unexpected end length", name, start, end)
			}
			if !IsWord(dict, start) || !IsWord(dict, end) {
				t.Fatalf("%s pair %q->%q: word missing from dictionary", name, start, end)
			}
			if _, ok := ShortestPathBFS(dict, start, end, 0); !ok {
				t.Fatalf("%s pair %q->%q: no path", name, start, end)
			}
		}
	}

	check("easy", easyDoublets, 3, 3)
	check("medium", mediumDoublets, 4, 4)
	check("hard", hardDoublets, 5, 5)
}

func TestSuggestionPoolSize(t *testing.T) {
	if len(easyDoublets) < 50 {
		t.Fatalf("easy pool too small: %d", len(easyDoublets))
	}
	if len(mediumDoublets) < 40 {
		t.Fatalf("medium pool too small: %d", len(mediumDoublets))
	}
	if len(hardDoublets) < 20 {
		t.Fatalf("hard pool too small: %d", len(hardDoublets))
	}
}
