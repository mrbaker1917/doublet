package game

import (
	"testing"
)

func TestExpertSuggestionPairsAreValid(t *testing.T) {
	dict, err := LoadDictionary("../../words-large.txt")
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	validateSuggestionPools(t, dict, expertSuggestions, "expert")
}

func TestCommonSuggestionPairsAreValid(t *testing.T) {
	dict, err := LoadDictionary("../../words-common.txt")
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	validateSuggestionPools(t, dict, commonSuggestions, "common")
}

func validateSuggestionPools(t *testing.T, dict Dictionary, pools suggestionPools, name string) {
	t.Helper()
	check := func(level string, pairs [][2]string, minLen, maxLen int) {
		t.Helper()
		if len(pairs) == 0 {
			t.Fatalf("%s %s: no pairs loaded", name, level)
		}
		for _, pair := range pairs {
			start, end := pair[0], pair[1]
			if len(start) < minLen || len(start) > maxLen {
				t.Fatalf("%s %s pair %q->%q: unexpected start length", name, level, start, end)
			}
			if len(end) < minLen || len(end) > maxLen {
				t.Fatalf("%s %s pair %q->%q: unexpected end length", name, level, start, end)
			}
			if !IsWord(dict, start) || !IsWord(dict, end) {
				t.Fatalf("%s %s pair %q->%q: word missing from dictionary", name, level, start, end)
			}
			if _, ok := ShortestPathBFS(dict, start, end, 0); !ok {
				t.Fatalf("%s %s pair %q->%q: no path", name, level, start, end)
			}
		}
	}

	check("easy", pools.easy, 3, 3)
	check("medium", pools.medium, 4, 4)
	check("hard", pools.hard, 5, 5)
}

func TestSuggestionPoolSize(t *testing.T) {
	if len(expertSuggestions.easy) < 50 {
		t.Fatalf("expert easy pool too small: %d", len(expertSuggestions.easy))
	}
	if len(expertSuggestions.medium) < 40 {
		t.Fatalf("expert medium pool too small: %d", len(expertSuggestions.medium))
	}
	if len(expertSuggestions.hard) < 20 {
		t.Fatalf("expert hard pool too small: %d", len(expertSuggestions.hard))
	}
	if len(commonSuggestions.easy) < 50 {
		t.Fatalf("common easy pool too small: %d", len(commonSuggestions.easy))
	}
	if len(commonSuggestions.medium) < 40 {
		t.Fatalf("common medium pool too small: %d", len(commonSuggestions.medium))
	}
	if len(commonSuggestions.hard) < 8 {
		t.Fatalf("common hard pool too small: %d", len(commonSuggestions.hard))
	}
}
