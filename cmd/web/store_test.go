package main

import (
	"errors"
	"testing"
	"time"
)

func TestGameStoreEvictsExpiredGame(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Millisecond, 0)

	created, err := store.create(&Game{
		Start:      "cat",
		End:        "dog",
		Difficulty: "easy",
		MaxChanges: 3,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	_, err = store.get(created.ID)
	if !errors.Is(err, errGameNotFound) {
		t.Fatalf("get expired game: got %v, want errGameNotFound", err)
	}
}

func TestGameStoreRefreshesTTLOnAccess(t *testing.T) {
	store := newGameStoreWithCleanup(10, 20*time.Millisecond, 0)

	created, err := store.create(&Game{
		Start:      "cat",
		End:        "dog",
		Difficulty: "easy",
		MaxChanges: 3,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if _, err := store.get(created.ID); err != nil {
		t.Fatalf("first get: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if _, err := store.get(created.ID); err != nil {
		t.Fatalf("second get after refresh: %v", err)
	}
}

func TestGameStoreEvictsOldestWhenFull(t *testing.T) {
	store := newGameStoreWithCleanup(2, time.Hour, 0)

	first, err := store.create(&Game{Start: "cat", End: "dog", Difficulty: "easy", MaxChanges: 3})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}

	time.Sleep(time.Millisecond)

	second, err := store.create(&Game{Start: "hit", End: "hot", Difficulty: "easy", MaxChanges: 2})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	time.Sleep(time.Millisecond)

	third, err := store.create(&Game{Start: "bat", End: "cat", Difficulty: "easy", MaxChanges: 2})
	if err != nil {
		t.Fatalf("create third: %v", err)
	}

	if _, err := store.get(first.ID); !errors.Is(err, errGameNotFound) {
		t.Fatalf("oldest game should be evicted, got err=%v", err)
	}

	if _, err := store.get(second.ID); err != nil {
		t.Fatalf("second game should remain: %v", err)
	}

	if _, err := store.get(third.ID); err != nil {
		t.Fatalf("third game should remain: %v", err)
	}
}

func TestGameStoreGetReturnsCopy(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Hour, 0)

	created, err := store.create(&Game{
		Start:        "cat",
		End:          "dog",
		Difficulty:   "easy",
		MaxChanges:   3,
		SolutionPath: []string{"cat", "cot", "dog"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	got.History[0] = "mutated"
	got.SolutionPath[0] = "mutated"

	again, err := store.get(created.ID)
	if err != nil {
		t.Fatalf("get again: %v", err)
	}

	if again.History[0] != "cat" {
		t.Fatalf("history mutated in store: %q", again.History[0])
	}
	if again.SolutionPath[0] != "cat" {
		t.Fatalf("solution path mutated in store: %q", again.SolutionPath[0])
	}
}
