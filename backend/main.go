package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"plaza/config"
	"plaza/handlers"
	"plaza/middleware"
	"plaza/models"
)

func main() {
	configPath := flag.String("config", "plaza.toml", "path to plaza.toml")
	port := flag.Int("port", 0, "port to listen on (overrides config)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if *port != 0 {
		cfg.Port = *port
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	plaza := models.NewPlaza()

	var store models.Store
	if cfg.History {
		store = models.NewFileStore(cfg.DataDir)
		log.Println("📝 persistencia activa en", cfg.DataDir+"/bancas")
	}

	h := handlers.New(plaza, cfg, store)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/config", h.GetConfig)
	mux.HandleFunc("GET /api/bancas", h.ListBancas)
	mux.HandleFunc("POST /api/bancas", h.CreateBanca)
	mux.HandleFunc("GET /api/bancas/{id}", h.GetBanca)
	mux.HandleFunc("GET /api/bancas/{id}/agents", h.ListAgents)
	mux.HandleFunc("POST /api/bancas/{id}/agents", h.InviteAgent)
	mux.HandleFunc("DELETE /api/bancas/{id}/agents/{agent_id}", h.RemoveAgent)
	mux.HandleFunc("GET /robots.txt", h.RobotsTXT)
	mux.HandleFunc("GET /.well-known/plaza", h.WellKnownPlaza)
	mux.HandleFunc("GET /ws/{id}", h.ServeWS)

	mux.Handle("GET /", http.FileServer(http.Dir("./frontend/dist")))

	ratelimit := middleware.NewRateLimiter(120, time.Minute)

	wrapped := panicRecovery(
		ratelimit.Limit(
			middleware.SecurityHeaders(
				middleware.Logger(mux),
			),
		),
	)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("🌳 %s — escuchando en %s", cfg.Title, addr)

	go func() {
		if err := http.ListenAndServe(addr, wrapped); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🌳 plaza cerrada. nos vemos en la banca.")
}

func panicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC: %v\n%s", rec, debug.Stack())
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
