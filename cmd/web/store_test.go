package main

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"doublet/internal/game"
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

func testDictionary(t *testing.T) game.Dictionary {
	t.Helper()
	dict, err := game.LoadDictionaryFromReader(strings.NewReader("cat\ncot\ncab\ndog\n"))
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	return dict
}

func TestGameStoreTryMoveRejectsStaleConcurrentMove(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Hour, 0)
	dict := testDictionary(t)

	created, err := store.create(&Game{
		Start:      "cat",
		End:        "dog",
		Current:    "cat",
		Status:     gameStatusPlaying,
		Difficulty: "easy",
		MaxChanges: 5,
		History:    []string{"cat"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	first, err := store.tryMove(created.ID, "cot", dict)
	if err != nil || !first.valid {
		t.Fatalf("first move: valid=%v err=%v", first.valid, err)
	}

	second, err := store.tryMove(created.ID, "cab", dict)
	if err != nil {
		t.Fatalf("second move: %v", err)
	}
	if second.valid {
		t.Fatal("second move from stale current should be rejected")
	}
	if second.message != "you must change exactly one letter" {
		t.Fatalf("unexpected message: %q", second.message)
	}

	again, err := store.get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if again.Current != "cot" {
		t.Fatalf("current = %q, want cot", again.Current)
	}
}

func TestGameStoreConcurrentReadsAndMoves(t *testing.T) {
	store := newGameStoreWithCleanup(100, time.Hour, 0)
	dict := testDictionary(t)

	created, err := store.create(&Game{
		Start:      "cat",
		End:        "dog",
		Current:    "cat",
		Status:     gameStatusPlaying,
		Difficulty: "easy",
		MaxChanges: 5,
		History:    []string{"cat"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				g, err := store.get(created.ID)
				if err != nil {
					continue
				}
				_, _ = json.Marshal(g)

				outcome, err := store.tryMove(created.ID, "cot", dict)
				if err == nil && outcome.valid {
					_, _ = json.Marshal(outcome.game)
				}
			}
		}()
	}

	wg.Wait()

	final, err := store.get(created.ID)
	if err != nil {
		t.Fatalf("get final: %v", err)
	}
	if final.Status != gameStatusPlaying && final.Status != gameStatusWon && final.Status != gameStatusLost {
		t.Fatalf("unexpected status: %q", final.Status)
	}
	if final.Current != "cot" && final.Current != "dog" {
		t.Fatalf("unexpected current word: %q", final.Current)
	}
}

func TestGameStoreRestartResetsProgress(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Hour, 0)
	dict := testDictionary(t)

	created, err := store.create(&Game{
		Start:        "cat",
		End:          "dog",
		Difficulty:   "easy",
		MaxChanges:   5,
		SolutionPath: []string{"cat", "cot", "dog"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	outcome, err := store.tryMove(created.ID, "cot", dict)
	if err != nil || !outcome.valid {
		t.Fatalf("move: valid=%v err=%v", outcome.valid, err)
	}

	restarted, err := store.restart(created.ID)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if restarted.Current != "cat" {
		t.Fatalf("current = %q, want cat", restarted.Current)
	}
	if restarted.MovesUsed != 0 {
		t.Fatalf("movesUsed = %d, want 0", restarted.MovesUsed)
	}
	if restarted.Status != gameStatusPlaying {
		t.Fatalf("status = %q, want playing", restarted.Status)
	}
	if len(restarted.History) != 1 || restarted.History[0] != "cat" {
		t.Fatalf("history = %v, want [cat]", restarted.History)
	}
}

func TestGameStoreHintReturnsNextStep(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Hour, 0)
	dict := testDictionary(t)

	created, err := store.create(&Game{
		Start:        "cat",
		End:          "dog",
		Difficulty:   "easy",
		MaxChanges:   5,
		SolutionPath: []string{"cat", "cot", "dog"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	outcome, err := store.hint(created.ID, dict)
	if err != nil {
		t.Fatalf("hint: %v", err)
	}
	if !outcome.ok || outcome.hint != "cot" {
		t.Fatalf("hint = %q, ok=%v; want cot, true", outcome.hint, outcome.ok)
	}
}

func TestGameStoreHintRejectsFinishedGame(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Hour, 0)
	dict, err := game.LoadDictionaryFromReader(strings.NewReader("cat\ncot\ncog\ndog\n"))
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}

	created, err := store.create(&Game{
		Start:        "cat",
		End:          "dog",
		Difficulty:   "easy",
		MaxChanges:   5,
		SolutionPath: []string{"cat", "cot", "cog", "dog"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, word := range []string{"cot", "cog", "dog"} {
		outcome, err := store.tryMove(created.ID, word, dict)
		if err != nil || !outcome.valid {
			t.Fatalf("move %q: valid=%v err=%v", word, outcome.valid, err)
		}
	}

	outcome, err := store.hint(created.ID, dict)
	if err != nil {
		t.Fatalf("hint: %v", err)
	}
	if outcome.ok {
		t.Fatalf("expected no hint for finished game, got %q", outcome.hint)
	}
}

func TestGameStoreSolveRevealsPathAndEndsGame(t *testing.T) {
	store := newGameStoreWithCleanup(10, time.Hour, 0)

	created, err := store.create(&Game{
		Start:        "cat",
		End:          "dog",
		Difficulty:   "easy",
		MaxChanges:   5,
		SolutionPath: []string{"cat", "cot", "cog", "dog"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	outcome, err := store.solve(created.ID)
	if err != nil {
		t.Fatalf("solve: %v", err)
	}
	if !outcome.ok {
		t.Fatalf("solve failed: %q", outcome.message)
	}
	if outcome.game.Status != gameStatusLost {
		t.Fatalf("status = %q, want lost", outcome.game.Status)
	}
	if len(outcome.solutionPath) != 4 || outcome.solutionPath[0] != "cat" || outcome.solutionPath[3] != "dog" {
		t.Fatalf("solutionPath = %v", outcome.solutionPath)
	}

	again, err := store.solve(created.ID)
	if err != nil {
		t.Fatalf("second solve: %v", err)
	}
	if again.ok {
		t.Fatal("expected second solve to fail for finished game")
	}
}
