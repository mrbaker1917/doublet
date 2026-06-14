package game

import (
	"strings"
	"testing"
)

func testDict(t *testing.T, words ...string) Dictionary {
	t.Helper()
	dict, err := LoadDictionaryFromReader(strings.NewReader(strings.Join(words, "\n")))
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	return dict
}

func TestAnotherValidPathPrefersDifferentShortest(t *testing.T) {
	dict := testDict(t, "cat", "cot", "cog", "dog", "dot", "dat")
	player := []string{"cat", "cot", "dog"}

	alt, ok := AnotherValidPath(dict, "cat", "dog", player)
	if !ok {
		t.Fatal("expected alternate path")
	}
	if pathsEqual(alt, player) {
		t.Fatalf("alternate should differ from player path: %v", alt)
	}
	if alt[0] != "cat" || alt[len(alt)-1] != "dog" {
		t.Fatalf("unexpected alternate path: %v", alt)
	}
}

func TestAnotherValidPathReturnsFalseWhenOnlyOnePath(t *testing.T) {
	dict := testDict(t, "cat", "cot", "dog")
	player := []string{"cat", "cot", "dog"}

	alt, ok := AnotherValidPath(dict, "cat", "dog", player)
	if ok {
		t.Fatalf("expected no alternate path, got %v", alt)
	}
}
