package main

import (
	"crypto/rand"
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
	errGameFinished = errors.New("game is already finished")
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

func (s *gameStore) applyMove(id, word string) (*Game, error) {
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

	g := &sg.game
	if g.Status != gameStatusPlaying {
		return nil, errGameFinished
	}

	g.Current = word
	g.MovesUsed++
	g.History = append(g.History, word)

	if g.Current == g.End {
		g.Status = gameStatusWon
	} else if g.MovesUsed >= g.MaxChanges {
		g.Status = gameStatusLost
	}

	sg.lastSeenAt = now
	return g.clone(), nil
}

func newGameID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
