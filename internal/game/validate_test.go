package game

import "testing"

func TestResolveMaxChangesPresetDifficultiesIgnoreRequestedMax(t *testing.T) {
	shortest := 5

	tests := []struct {
		difficulty string
		requested  int
		want       int
	}{
		{"easy", 0, 8},
		{"easy", 1_000_000, 8},
		{"medium", 999, 7},
		{"hard", 42, 5},
	}

	for _, tc := range tests {
		got, err := ResolveMaxChanges(tc.difficulty, tc.requested, shortest)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.difficulty, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %d, want %d", tc.difficulty, got, tc.want)
		}
	}
}

func TestScaledSlack(t *testing.T) {
	tests := []struct {
		shortest     int
		easy, medium int
	}{
		{1, 2, 1},
		{2, 4, 3},
		{3, 5, 4},
		{4, 6, 5},
		{5, 8, 7},
		{6, 9, 8},
	}

	for _, tc := range tests {
		easy, err := ResolveMaxChanges("easy", 0, tc.shortest)
		if err != nil || easy != tc.easy {
			t.Fatalf("easy shortest=%d: got %d err=%v want %d", tc.shortest, easy, err, tc.easy)
		}

		medium, err := ResolveMaxChanges("medium", 0, tc.shortest)
		if err != nil || medium != tc.medium {
			t.Fatalf("medium shortest=%d: got %d err=%v want %d", tc.shortest, medium, err, tc.medium)
		}

		hard, err := ResolveMaxChanges("hard", 0, tc.shortest)
		if err != nil || hard != tc.shortest {
			t.Fatalf("hard shortest=%d: got %d err=%v want %d", tc.shortest, hard, err, tc.shortest)
		}
	}
}

func TestResolveMaxChangesCustomRequiresMax(t *testing.T) {
	_, err := ResolveMaxChanges("custom", 0, 5)
	if err == nil {
		t.Fatal("expected error when custom max is missing")
	}
}

func TestResolveMaxChangesCustomRejectsBelowShortest(t *testing.T) {
	_, err := ResolveMaxChanges("custom", 4, 5)
	if err == nil {
		t.Fatal("expected error when custom max is below shortest path")
	}
}

func TestResolveMaxChangesCustomRejectsExcessiveMax(t *testing.T) {
	_, err := ResolveMaxChanges("custom", 1_000_000, 5)
	if err == nil {
		t.Fatal("expected error when custom max exceeds cap")
	}
}

func TestResolveMaxChangesCustomAcceptsValidMax(t *testing.T) {
	got, err := ResolveMaxChanges("custom", 8, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 8 {
		t.Fatalf("got %d, want 8", got)
	}
}

func TestCustomMaxChangesCap(t *testing.T) {
	if got := CustomMaxChangesCap(5); got != 15 {
		t.Fatalf("CustomMaxChangesCap(5) = %d, want 15", got)
	}
	if got := CustomMaxChangesCap(95); got != MaxCustomChanges {
		t.Fatalf("CustomMaxChangesCap(95) = %d, want %d", got, MaxCustomChanges)
	}
}
