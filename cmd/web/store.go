package main

import (
	"crypto/rand"
	"doublet/internal/game"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const (
	gameStatusPlaying = "playing"
	gameStatusWon     = "won"
	gameStatusLost    = "lost"

	defaultMaxGames        = 2000
	defaultGameTTL         = 24 * time.Hour
	defaultCleanupInterval = 5 * time.Minute
)

var (
	errGameNotFound = errors.New("game not found")
)

type Game struct {
	ID           string   `json:"id"`
	Start        string   `json:"start"`
	End          string   `json:"end"`
	Current      string   `json:"current"`
	Difficulty   string   `json:"difficulty"`
	MaxChanges   int      `json:"maxChanges"`
	MovesUsed    int      `json:"movesUsed"`
	History      []string `json:"history"`
	Status       string   `json:"status"`
	SolutionPath []string `json:"-"`
}

func (g *Game) clone() *Game {
	if g == nil {
		return nil
	}
	copy := *g
	copy.History = append([]string(nil), g.History...)
	copy.SolutionPath = append([]string(nil), g.SolutionPath...)
	return &copy
}

type storedGame struct {
	game       Game
	createdAt  time.Time
	lastSeenAt time.Time
}

type gameStore struct {
	mu              sync.Mutex
	games           map[string]*storedGame
	maxGames        int
	ttl             time.Duration
	cleanupInterval time.Duration
}

func newGameStore(maxGames int, ttl time.Duration) *gameStore {
	if maxGames <= 0 {
		maxGames = defaultMaxGames
	}
	if ttl <= 0 {
		ttl = defaultGameTTL
	}
	return newGameStoreWithCleanup(maxGames, ttl, defaultCleanupInterval)
}

func newGameStoreWithCleanup(maxGames int, ttl, cleanupInterval time.Duration) *gameStore {
	if maxGames <= 0 {
		maxGames = defaultMaxGames
	}
	if ttl <= 0 {
		ttl = defaultGameTTL
	}

	s := &gameStore{
		games:           make(map[string]*storedGame),
		maxGames:        maxGames,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
	}

	if cleanupInterval > 0 {
		go s.cleanupLoop(cleanupInterval)
	}

	return s
}

func (s *gameStore) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.evictExpiredLocked(time.Now())
		s.mu.Unlock()
	}
}

func (s *gameStore) isExpired(sg *storedGame, now time.Time) bool {
	return now.Sub(sg.lastSeenAt) > s.ttl
}

func (s *gameStore) evictExpiredLocked(now time.Time) {
	for id, sg := range s.games {
		if s.isExpired(sg, now) {
			delete(s.games, id)
		}
	}
}

func (s *gameStore) evictOldestLocked() {
	if len(s.games) == 0 {
		return
	}

	var oldestID string
	var oldestSeen time.Time
	first := true

	for id, sg := range s.games {
		if first || sg.lastSeenAt.Before(oldestSeen) {
			oldestID = id
			oldestSeen = sg.lastSeenAt
			first = false
		}
	}

	delete(s.games, oldestID)
}

func (s *gameStore) makeRoomLocked(now time.Time) {
	s.evictExpiredLocked(now)
	for len(s.games) >= s.maxGames {
		s.evictOldestLocked()
	}
}

func (s *gameStore) create(g *Game) (*Game, error) {
	id, err := newGameID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	stored := &storedGame{
		game: Game{
			ID:           id,
			Start:        g.Start,
			End:          g.End,
			Current:      g.Start,
			Difficulty:   g.Difficulty,
			MaxChanges:   g.MaxChanges,
			MovesUsed:    0,
			History:      []string{g.Start},
			Status:       gameStatusPlaying,
			SolutionPath: append([]string(nil), g.SolutionPath...),
		},
		createdAt:  now,
		lastSeenAt: now,
	}

	s.mu.Lock()
	s.makeRoomLocked(now)
	s.games[id] = stored
	s.mu.Unlock()

	return stored.game.clone(), nil
}

// get returns a snapshot clone safe to read or JSON-encode without holding the store lock.
func (s *gameStore) get(id string) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sg, ok := s.games[id]
	if !ok {
		return nil, errGameNotFound
	}

	now := time.Now()
	if s.isExpired(sg, now) {
		delete(s.games, id)
		return nil, errGameNotFound
	}

	sg.lastSeenAt = now
	return sg.game.clone(), nil
}

type moveOutcome struct {
	valid   bool
	game    *Game
	message string
	won     bool
	lost    bool
}

// tryMove validates and applies a move under the store lock, returning a snapshot safe for JSON encoding.
func (s *gameStore) tryMove(id, rawWord string, dict game.Dictionary) (moveOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sg, ok := s.games[id]
	if !ok {
		return moveOutcome{}, errGameNotFound
	}

	now := time.Now()
	if s.isExpired(sg, now) {
		delete(s.games, id)
		return moveOutcome{}, errGameNotFound
	}

	g := &sg.game
	if g.Status != gameStatusPlaying {
		return moveOutcome{
			valid:   false,
			message: "game is already finished",
			won:     g.Status == gameStatusWon,
			lost:    g.Status == gameStatusLost,
		}, nil
	}

	next := game.Normalize(rawWord)
	if next == "" {
		return moveOutcome{valid: false, message: "word is required"}, nil
	}
	if len(next) != len(g.Current) {
		return moveOutcome{
			valid:   false,
			message: "word must be the same length as the current word",
		}, nil
	}
	if !game.IsWord(dict, next) {
		return moveOutcome{
			valid:   false,
			message: next + " is not in the dictionary",
		}, nil
	}
	if !game.OneLetterApart(g.Current, next) {
		return moveOutcome{
			valid:   false,
			message: "you must change exactly one letter",
		}, nil
	}

	g.Current = next
	g.MovesUsed++
	g.History = append(g.History, next)

	if g.Current == g.End {
		g.Status = gameStatusWon
	} else if g.MovesUsed >= g.MaxChanges {
		g.Status = gameStatusLost
	}

	sg.lastSeenAt = now
	snapshot := g.clone()

	return moveOutcome{
		valid: true,
		game:  snapshot,
		won:   snapshot.Status == gameStatusWon,
		lost:  snapshot.Status == gameStatusLost,
	}, nil
}

func newGameID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
