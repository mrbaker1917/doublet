package game

import "testing"

func TestIsPlayableWord(t *testing.T) {
	if !IsPlayableWord("cat") {
		t.Fatal("cat should be playable")
	}
	if IsPlayableWord("quiz") {
		t.Fatal("quiz should not be playable")
	}
	if IsPlayableWord("gray") {
		t.Fatal("gray should not be playable")
	}
}
