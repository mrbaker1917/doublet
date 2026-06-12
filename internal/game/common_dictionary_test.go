package game

import (
	"testing"
)

func TestCommonDictionaryIncludesReportedWords(t *testing.T) {
	dict, err := LoadDictionary("../../words-common.txt")
	if err != nil {
		t.Fatalf("load common dictionary: %v", err)
	}

	for _, word := range []string{"brood", "gloom", "broom", "gloat", "bloat", "soot"} {
		if !IsWord(dict, word) {
			t.Fatalf("common dictionary missing %q", word)
		}
	}
}

func TestCommonDictionaryExcludesObscureWords(t *testing.T) {
	dict, err := LoadDictionary("../../words-common.txt")
	if err != nil {
		t.Fatalf("load common dictionary: %v", err)
	}

	for _, word := range []string{"fiot", "flot", "frot"} {
		if IsWord(dict, word) {
			t.Fatalf("common dictionary should not include %q", word)
		}
	}
}
