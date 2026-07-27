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

	"json-books-app/db"
	"json-books-app/handlers"
	"json-books-app/middleware"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"UP","database":"postgresql"}`))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	targetDBName := os.Getenv("POSTGRES_DB")
	if targetDBName == "" {
		targetDBName = "zennotes"
	}

	// Initialize PostgreSQL Database
	database, err := db.InitDB(targetDBName)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL database: %v", err)
	}
	defer database.Close()

	mux := http.NewServeMux()

	// JSON Auth & REST Endpoints
	mux.HandleFunc("POST /api/auth/register", handlers.RegisterHandler)
	mux.HandleFunc("POST /api/auth/login", handlers.LoginHandler)
	mux.HandleFunc("POST /api/auth/logout", handlers.LogoutHandler)
	mux.HandleFunc("GET /api/auth/me", middleware.OptionalAuth(handlers.MeHandler))
	mux.HandleFunc("/api/notes", middleware.AuthRequired(handlers.NotesHandler))
	mux.HandleFunc("/api/notes/", middleware.AuthRequired(handlers.NotesHandler))

	// Pure HTMX Server-Rendered HTML Routes
	mux.HandleFunc("/htmx/header", middleware.OptionalAuth(handlers.HTMXHeaderHandler))
	mux.HandleFunc("POST /htmx/auth/login", handlers.HTMXLoginHandler)
	mux.HandleFunc("POST /htmx/auth/register", handlers.HTMXRegisterHandler)
	mux.HandleFunc("POST /htmx/auth/logout", handlers.HTMXLogoutHandler)
	mux.HandleFunc("/htmx/notes", middleware.AuthRequired(handlers.HTMXNotesHandler))
	mux.HandleFunc("/htmx/notes/", middleware.AuthRequired(handlers.HTMXNotesHandler))

	// Health Check
	mux.HandleFunc("GET /healthz", healthHandler)

	// Static File Server
	staticServer := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 ZenNotes Go PostgreSQL Server running at http://localhost:%s", port)
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
