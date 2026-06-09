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
	dictPath := flag.String("dict", "", "path to newline-separated dictionary (overrides -lexicon)")
	lexicon := flag.String("lexicon", "large", "dictionary preset: small or large")
	port := flag.String("port", "", "listen port (overrides PORT env var)")
	maxGames := flag.Int("max-games", defaultMaxGames, "maximum in-memory games before evicting least recently used")
	gameTTL := flag.Duration("game-ttl", defaultGameTTL, "how long inactive games are kept (e.g. 24h, 30m)")
	createRateLimit := flag.Int("create-rate-limit", defaultCreateRateLimit, "max POST /api/games requests per IP per rate window")
	createRateWindow := flag.Duration("create-rate-window", defaultCreateRateWindow, "rate limit window for game creation (e.g. 1m)")
	maxConcurrentBFS := flag.Int("max-concurrent-bfs", defaultMaxConcurrentBFS, "max simultaneous BFS path searches")
	bfsWait := flag.Duration("bfs-wait", defaultBFSWait, "how long to wait for a BFS slot before returning busy")
	pathCacheSize := flag.Int("path-cache-size", defaultPathCacheSize, "cached start/end shortest paths")
	maxRequestBody := flag.Int64("max-request-body", defaultMaxRequestBody, "maximum JSON request body size in bytes")
	flag.Parse()

	dict, err := game.LoadDictionaryForFlags(*dictPath, *lexicon)
	if err != nil {
		log.Fatalf("failed to load dictionary: %v", err)
	}

	listenPort, err := resolveListenPort(*port)
	if err != nil {
		log.Fatalf("invalid listen port: %v", err)
	}

	srv := &server{
		dict:           dict,
		store:          newGameStore(*maxGames, *gameTTL),
		bfsGate:        newBFSGate(*maxConcurrentBFS),
		createLimiter:  newCreateRateLimiter(*createRateLimit, *createRateWindow),
		pathCache:      newPathCache(*pathCacheSize),
		bfsWait:        *bfsWait,
		maxRequestBody: *maxRequestBody,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/suggestions", srv.handleSuggestions)
	mux.HandleFunc("POST /api/games", srv.handleCreateGame)
	mux.HandleFunc("GET /api/games/{id}", srv.handleGetGame)
	mux.HandleFunc("POST /api/games/{id}/move", srv.handleMove)
	mux.Handle("/", staticHandler())

	httpSrv := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf(
		"doublet web server listening on port %s (max-games=%d game-ttl=%s create-rate=%d/%s max-concurrent-bfs=%d)",
		listenPort, *maxGames, *gameTTL, *createRateLimit, *createRateWindow, *maxConcurrentBFS,
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
