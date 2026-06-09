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

type suggestionsResponse struct {
	Easy   [2]string `json:"easy"`
	Medium [2]string `json:"medium"`
	Hard   [2]string `json:"hard"`
}

type server struct {
	dict            game.Dictionary
	store           *gameStore
	bfsGate         *bfsGate
	createLimiter   *createRateLimiter
	pathCache       *pathCache
	bfsWait         time.Duration
	maxRequestBody  int64
}

func (s *server) shortestPath(start, end string) ([]string, bool, error) {
	if path, found, ok := s.pathCache.get(start, end); ok {
		return path, found, nil
	}

	if !s.bfsGate.acquire(s.bfsWait) {
		return nil, false, errBFSBusy
	}
	path, found := game.ShortestPathBFS(s.dict, start, end, 0)
	s.bfsGate.release()

	s.pathCache.put(start, end, path, found)
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

	if err := game.ValidateInputs(s.dict, start, end); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := game.ValidateDifficulty(difficulty); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !s.createLimiter.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many game requests, try again later")
		return
	}

	shortest, found, err := s.shortestPath(start, end)
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

	g, err := s.store.get(id)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load game")
		return
	}
	if g.Status != gameStatusPlaying {
		writeJSON(w, http.StatusOK, moveResponse{
			Valid:   false,
			Message: "game is already finished",
			Won:     g.Status == gameStatusWon,
			Lost:    g.Status == gameStatusLost,
		})
		return
	}

	next := game.Normalize(req.Word)
	if next == "" {
		writeJSON(w, http.StatusOK, moveResponse{Valid: false, Message: "word is required"})
		return
	}
	if len(next) != len(g.Current) {
		writeJSON(w, http.StatusOK, moveResponse{
			Valid:   false,
			Message: "word must be the same length as the current word",
		})
		return
	}
	if !game.IsWord(s.dict, next) {
		writeJSON(w, http.StatusOK, moveResponse{
			Valid:   false,
			Message: next + " is not in the dictionary",
		})
		return
	}
	if !game.OneLetterApart(g.Current, next) {
		writeJSON(w, http.StatusOK, moveResponse{
			Valid:   false,
			Message: "you must change exactly one letter",
		})
		return
	}

	updated, err := s.store.applyMove(id, next)
	if err != nil {
		if errors.Is(err, errGameNotFound) {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
		if errors.Is(err, errGameFinished) {
			writeJSON(w, http.StatusOK, moveResponse{Valid: false, Message: "game is already finished"})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to apply move")
		return
	}

	resp := moveResponse{
		Valid:     true,
		Current:   updated.Current,
		MovesUsed: updated.MovesUsed,
		History:   updated.History,
		Won:       updated.Status == gameStatusWon,
		Lost:      updated.Status == gameStatusLost,
	}
	if updated.Status == gameStatusLost {
		resp.SolutionPath = updated.SolutionPath
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	easy, medium, hard := game.GetSuggestedDoublets()
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
