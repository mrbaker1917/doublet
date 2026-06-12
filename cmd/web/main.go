package main

import (
	"doublet/internal/game"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	dictPath := flag.String("dict", "", "path to expert dictionary (overrides -lexicon)")
	lexicon := flag.String("lexicon", "large", "expert dictionary preset: small, common, or large")
	port := flag.String("port", "", "listen port (overrides PORT env var)")
	maxGames := flag.Int("max-games", defaultMaxGames, "maximum in-memory games before evicting least recently used")
	gameTTL := flag.Duration("game-ttl", defaultGameTTL, "how long inactive games are kept (e.g. 24h, 30m)")
	createRateLimit := flag.Int("create-rate-limit", defaultCreateRateLimit, "max POST /api/games requests per IP per rate window")
	moveRateLimit := flag.Int("move-rate-limit", defaultMoveRateLimit, "max POST /api/games/{id}/move requests per IP per rate window")
	readRateLimit := flag.Int("read-rate-limit", defaultReadRateLimit, "max GET /api/suggestions and /api/games/{id} requests per IP per rate window")
	apiRateWindow := flag.Duration("api-rate-window", defaultAPIRateWindow, "rate limit window for API requests (e.g. 1m)")
	createRateWindow := flag.Duration("create-rate-window", 0, "deprecated alias for -api-rate-window")
	maxConcurrentBFS := flag.Int("max-concurrent-bfs", defaultMaxConcurrentBFS, "max simultaneous BFS path searches")
	bfsWait := flag.Duration("bfs-wait", defaultBFSWait, "how long to wait for a BFS slot before returning busy")
	pathCacheSize := flag.Int("path-cache-size", defaultPathCacheSize, "cached start/end shortest paths")
	maxRequestBody := flag.Int64("max-request-body", defaultMaxRequestBody, "maximum JSON request body size in bytes")
	flag.Parse()

	rateWindow := *apiRateWindow
	if *createRateWindow > 0 {
		rateWindow = *createRateWindow
	}

	commonDict, expertDict, err := game.LoadWebDictionaries(*dictPath, *lexicon)
	if err != nil {
		log.Fatalf("failed to load dictionaries: %v", err)
	}

	listenPort, err := resolveListenPort(*port)
	if err != nil {
		log.Fatalf("invalid listen port: %v", err)
	}

	srv := &server{
		commonDict:     commonDict,
		expertDict:     expertDict,
		store:          newGameStore(*maxGames, *gameTTL, commonDict, expertDict),
		bfsGate:        newBFSGate(*maxConcurrentBFS),
		createLimiter:  newIPRateLimiter(*createRateLimit, rateWindow),
		moveLimiter:    newIPRateLimiter(*moveRateLimit, rateWindow),
		readLimiter:    newIPRateLimiter(*readRateLimit, rateWindow),
		pathCache:      newPathCache(*pathCacheSize),
		bfsWait:        *bfsWait,
		maxRequestBody: *maxRequestBody,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/suggestions", srv.handleSuggestions)
	mux.HandleFunc("POST /api/games", srv.handleCreateGame)
	mux.HandleFunc("GET /api/games/{id}", srv.handleGetGame)
	mux.HandleFunc("POST /api/games/{id}/move", srv.handleMove)
	mux.HandleFunc("POST /api/games/{id}/hint", srv.handleHint)
	mux.HandleFunc("POST /api/games/{id}/restart", srv.handleRestart)
	mux.HandleFunc("POST /api/games/{id}/solve", srv.handleSolve)
	mux.Handle("/", staticHandler())

	httpSrv := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf(
		"doublet web server listening on port %s (common=%d expert=%d max-games=%d game-ttl=%s api-rate=%d/%d/%d per %s max-concurrent-bfs=%d)",
		listenPort, len(commonDict), len(expertDict), *maxGames, *gameTTL, *createRateLimit, *moveRateLimit, *readRateLimit, rateWindow, *maxConcurrentBFS,
	)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func resolveListenPort(flagPort string) (string, error) {
	port := strings.TrimSpace(flagPort)
	if port == "" {
		port = strings.TrimSpace(os.Getenv("PORT"))
	}
	if port == "" {
		return "8080", nil
	}

	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("port must be a number between 1 and 65535")
	}

	return strconv.Itoa(n), nil
}

func staticHandler() http.Handler {
	fileServer := http.FileServer(http.Dir("web"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
