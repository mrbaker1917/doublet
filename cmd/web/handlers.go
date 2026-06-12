package main

import (
	"doublet/internal/game"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const defaultMaxRequestBody = 8192

var (
	errBFSBusy             = errors.New("bfs capacity exhausted")
	errRequestBodyTooLarge = errors.New("request body too large")
)

type errorResponse struct {
	Error string `json:"error"`
}

type createGameRequest struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	Difficulty string `json:"difficulty"`
	Max        int    `json:"max"`
	Expert     bool   `json:"expert"`
}

type moveRequest struct {
	Word string `json:"word"`
}

type moveResponse struct {
	Valid        bool     `json:"valid"`
	Current      string   `json:"current,omitempty"`
	MovesUsed    int      `json:"movesUsed,omitempty"`
	History      []string `json:"history,omitempty"`
	Won          bool     `json:"won"`
	Lost         bool     `json:"lost"`
	Message      string   `json:"message"`
	SolutionPath []string `json:"solutionPath,omitempty"`
}

type hintResponse struct {
	OK      bool   `json:"ok"`
	Hint    string `json:"hint,omitempty"`
	Message string `json:"message,omitempty"`
}

type solveResponse struct {
	OK           bool     `json:"ok"`
	Lost         bool     `json:"lost"`
	GaveUp       bool     `json:"gaveUp"`
	Message      string   `json:"message,omitempty"`
	SolutionPath []string `json:"solutionPath,omitempty"`
}

type suggestionsResponse struct {
	Easy   [2]string `json:"easy"`
	Medium [2]string `json:"medium"`
	Hard   [2]string `json:"hard"`
}

type server struct {
	commonDict     game.Dictionary
	expertDict     game.Dictionary
	store          *gameStore
	bfsGate        *bfsGate
	createLimiter  *ipRateLimiter
	moveLimiter    *ipRateLimiter
	readLimiter    *ipRateLimiter
	pathCache      *pathCache
	bfsWait        time.Duration
	maxRequestBody int64
}

func (s *server) dictFor(expert bool) game.Dictionary {
	if expert {
		return s.expertDict
	}
	return s.commonDict
}

func (s *server) requireRateLimit(w http.ResponseWriter, r *http.Request, limiter *ipRateLimiter) bool {
	if !limiter.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many requests, try again later")
		return false
	}
	return true
}

func (s *server) shortestPath(start, end string, expert bool) ([]string, bool, error) {
	if path, found, ok := s.pathCache.get(start, end, expert); ok {
		return path, found, nil
	}

	if !s.bfsGate.acquire(s.bfsWait) {
		return nil, false, errBFSBusy
	}
	path, found := game.ShortestPathBFS(s.dictFor(expert), start, end, 0)
	s.bfsGate.release()

	s.pathCache.put(start, end, expert, path, found)
	return path, found, nil
}

func (s *server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createGameRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	start := game.Normalize(req.Start)
	end := game.Normalize(req.End)
	difficulty := game.NormalizeDifficulty(req.Difficulty)
	if difficulty == "" {
		difficulty = "medium"
	}

	dict := s.dictFor(req.Expert)

	if err := game.ValidateInputs(dict, start, end); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := game.ValidateDifficulty(difficulty); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !s.requireRateLimit(w, r, s.createLimiter) {
		return
	}

	shortest, found, err := s.shortestPath(start, end, req.Expert)
	if errors.Is(err, errBFSBusy) {
		writeError(w, http.StatusServiceUnavailable, "server busy, try again")
		return
	}
	if !found {
		writeError(w, http.StatusBadRequest, "no path found with current dictionary")
		return
	}

	maxChanges, err := game.ResolveMaxChanges(difficulty, req.Max, len(shortest)-1)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(shortest)-1 > maxChanges {
		writeError(w, http.StatusBadRequest, "no path found within allowed changes")
		return
	}

	created, err := s.store.create(&Game{
		Start:        start,
		End:          end,
		Difficulty:   difficulty,
		MaxChanges:   maxChanges,
		Expert:       req.Expert,
		SolutionPath: shortest,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create game")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *server) handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.requireRateLimit(w, r, s.moveLimiter) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid game id")
		return
	}

	var req moveRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	outcome, err := s.store.tryMove(id, req.Word)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to apply move")
		return
	}

	if !outcome.valid {
		writeJSON(w, http.StatusOK, moveResponse{
			Valid:   false,
			Message: outcome.message,
			Won:     outcome.won,
			Lost:    outcome.lost,
		})
		return
	}

	resp := moveResponse{
		Valid:     true,
		Current:   outcome.game.Current,
		MovesUsed: outcome.game.MovesUsed,
		History:   outcome.game.History,
		Won:       outcome.won,
		Lost:      outcome.lost,
	}
	if outcome.lost {
		resp.SolutionPath = outcome.game.SolutionPath
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleHint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.requireRateLimit(w, r, s.moveLimiter) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid game id")
		return
	}

	outcome, err := s.store.hint(id)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get hint")
		return
	}

	if !outcome.ok {
		writeJSON(w, http.StatusOK, hintResponse{
			Message: outcome.message,
		})
		return
	}

	writeJSON(w, http.StatusOK, hintResponse{
		OK:   true,
		Hint: outcome.hint,
	})
}

func (s *server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.requireRateLimit(w, r, s.moveLimiter) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid game id")
		return
	}

	g, err := s.store.restart(id)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to restart game")
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (s *server) handleSolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.requireRateLimit(w, r, s.moveLimiter) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid game id")
		return
	}

	outcome, err := s.store.solve(id)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reveal solution")
		return
	}

	if !outcome.ok {
		writeJSON(w, http.StatusOK, solveResponse{
			Message: outcome.message,
		})
		return
	}

	writeJSON(w, http.StatusOK, solveResponse{
		OK:           true,
		Lost:         true,
		GaveUp:       true,
		SolutionPath: outcome.solutionPath,
	})
}

func (s *server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.requireRateLimit(w, r, s.readLimiter) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid game id")
		return
	}

	g, err := s.store.get(id)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load game")
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (s *server) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.requireRateLimit(w, r, s.readLimiter) {
		return
	}

	expert := r.URL.Query().Get("expert") == "true"
	easy, medium, hard := game.GetSuggestedDoublets(expert)
	writeJSON(w, http.StatusOK, suggestionsResponse{
		Easy:   easy,
		Medium: medium,
		Hard:   hard,
	})
}

func (s *server) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := s.maxRequestBody
	if maxBytes <= 0 {
		maxBytes = defaultMaxRequestBody
	}

	limited := http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(limited).Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errRequestBodyTooLarge
		}
		return err
	}
	return nil
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRequestBodyTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid JSON body")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
