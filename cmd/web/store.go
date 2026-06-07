package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

const (
	gameStatusPlaying = "playing"
	gameStatusWon     = "won"
	gameStatusLost    = "lost"
)

var (
	errGameNotFound = errors.New("game not found")
	errGameFinished = errors.New("game is already finished")
)

type Game struct {
	ID         string   `json:"id"`
	Start      string   `json:"start"`
	End        string   `json:"end"`
	Current    string   `json:"current"`
	Difficulty string   `json:"difficulty"`
	MaxChanges int      `json:"maxChanges"`
	MovesUsed  int      `json:"movesUsed"`
	History    []string `json:"history"`
	Status     string   `json:"status"`
}

type gameStore struct {
	mu    sync.RWMutex
	games map[string]*Game
}

func newGameStore() *gameStore {
	return &gameStore{games: make(map[string]*Game)}
}

func (s *gameStore) create(g *Game) (*Game, error) {
	id, err := newGameID()
	if err != nil {
		return nil, err
	}

	stored := &Game{
		ID:         id,
		Start:      g.Start,
		End:        g.End,
		Current:    g.Start,
		Difficulty: g.Difficulty,
		MaxChanges: g.MaxChanges,
		MovesUsed:  0,
		History:    []string{g.Start},
		Status:     gameStatusPlaying,
	}

	s.mu.Lock()
	s.games[id] = stored
	s.mu.Unlock()

	return stored, nil
}

func (s *gameStore) get(id string) (*Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.games[id]
	if !ok {
		return nil, errGameNotFound
	}
	return g, nil
}

func (s *gameStore) applyMove(id, word string) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.games[id]
	if !ok {
		return nil, errGameNotFound
	}
	if g.Status != gameStatusPlaying {
		return nil, errGameFinished
	}

	g.Current = word
	g.MovesUsed++
	g.History = append(g.History, word)

	if g.Current == g.End {
		g.Status = gameStatusWon
		return g, nil
	}
	if g.MovesUsed >= g.MaxChanges {
		g.Status = gameStatusLost
	}
	return g, nil
}

func newGameID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
