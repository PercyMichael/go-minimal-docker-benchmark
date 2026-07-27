package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"json-books-app/handlers"
	"json-books-app/internal/config"
	"json-books-app/internal/db"
	"json-books-app/middleware"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"UP","database":"postgresql"}`))
}

func main() {
	cfg := config.Load()

	// Initialize PostgreSQL Database
	database, err := db.InitDB(cfg.TargetDBName)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL database: %v", err)
	}
	defer database.Close()

	r := chi.NewRouter()

	// Built-in Chi Middlewares
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// JSON Auth & REST Endpoints
	r.Post("/api/auth/register", handlers.RegisterHandler)
	r.Post("/api/auth/login", handlers.LoginHandler)
	r.Post("/api/auth/logout", handlers.LogoutHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalAuth)
		r.Get("/api/auth/me", handlers.MeHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthRequired)
		r.HandleFunc("/api/notes", handlers.NotesHandler)
		r.HandleFunc("/api/notes/*", handlers.NotesHandler)
	})

	// Pure HTMX Server-Rendered HTML Routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalAuth)
		r.Get("/htmx/header", handlers.HTMXHeaderHandler)
	})

	r.Post("/htmx/auth/login", handlers.HTMXLoginHandler)
	r.Post("/htmx/auth/register", handlers.HTMXRegisterHandler)
	r.Post("/htmx/auth/logout", handlers.HTMXLogoutHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthRequired)
		r.HandleFunc("/htmx/notes", handlers.HTMXNotesHandler)
		r.HandleFunc("/htmx/notes/*", handlers.HTMXNotesHandler)
	})

	// Health Check
	r.Get("/healthz", healthHandler)

	// Static File Server & Catch-all Fallback
	staticServer := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/htmx/") {
			http.NotFound(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/static/") {
			staticServer.ServeHTTP(w, r)
			return
		}

		path := filepath.Clean(r.URL.Path)
		if path == "." || path == "/" || path == "index.html" {
			http.ServeFile(w, r, "static/index.html")
			return
		}

		filePath := filepath.Join("static", path)
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, filePath)
			return
		}

		http.ServeFile(w, r, "static/index.html")
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 ZenNotes Go PostgreSQL Server running at http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down PostgreSQL server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}

	log.Println("ZenNotes server stopped cleanly")
}
