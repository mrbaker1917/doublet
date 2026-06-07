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
		dict:  dict,
		store: newGameStore(),
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

	log.Printf("doublet web server listening on port %s", listenPort)
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
