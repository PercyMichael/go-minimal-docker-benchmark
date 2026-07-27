package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB initializes PostgreSQL connection, creates database if missing, and runs migrations.
func InitDB(targetDBName string) (*sql.DB, error) {
	if targetDBName == "" {
		targetDBName = "zennotes"
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		user := os.Getenv("PGUSER")
		if user == "" {
			user = "user"
		}
		password := os.Getenv("PGPASSWORD")
		host := os.Getenv("PGHOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("PGPORT")
		if port == "" {
			port = "5432"
		}

		if password != "" {
			connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, targetDBName)
		} else {
			connStr = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", user, host, port, targetDBName)
		}
	}

	if err := ensureDatabaseExists(connStr, targetDBName); err != nil {
		log.Printf("Warning during DB creation check: %v", err)
	}

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		if !strings.Contains(connStr, "postgres:") && os.Getenv("DATABASE_URL") == "" {
			fallbackStr := "postgres://postgres@localhost:5432/" + targetDBName + "?sslmode=disable"
			log.Printf("Retrying DB connection with fallback: %s", fallbackStr)
			connStr = fallbackStr
			database, err = sql.Open("postgres", connStr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres database: %w", err)
		}
	}

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	if err := database.Ping(); err != nil {
		if !strings.Contains(connStr, "postgres:") && os.Getenv("DATABASE_URL") == "" {
			fallbackStr := "postgres://postgres@localhost:5432/" + targetDBName + "?sslmode=disable"
			log.Printf("Ping failed, trying postgres user fallback: %s", fallbackStr)
			_ = ensureDatabaseExists(fallbackStr, targetDBName)
			database, err = sql.Open("postgres", fallbackStr)
			if err == nil {
				err = database.Ping()
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to ping postgres database: %w", err)
		}
	}

	if err := migrate(database); err != nil {
		return nil, fmt.Errorf("failed to run postgres migrations: %w", err)
	}

	DB = database
	log.Printf("Successfully connected to PostgreSQL database [%s]", targetDBName)
	return DB, nil
}

func ensureDatabaseExists(connStr string, targetDBName string) error {
	var rootConnStr string
	if strings.Contains(connStr, "/"+targetDBName) {
		rootConnStr = strings.Replace(connStr, "/"+targetDBName, "/postgres", 1)
	} else {
		rootConnStr = "postgres://localhost:5432/postgres?sslmode=disable"
	}

	rootDB, err := sql.Open("postgres", rootConnStr)
	if err != nil {
		return err
	}
	defer rootDB.Close()

	if err := rootDB.Ping(); err != nil {
		rootConnStr = "postgres://postgres@localhost:5432/postgres?sslmode=disable"
		rootDB, err = sql.Open("postgres", rootConnStr)
		if err != nil {
			return err
		}
		defer rootDB.Close()
		if err := rootDB.Ping(); err != nil {
			return err
		}
	}

	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	if err := rootDB.QueryRow(checkQuery, targetDBName).Scan(&exists); err != nil {
		return err
	}

	if !exists {
		log.Printf("Creating PostgreSQL database: %s", targetDBName)
		_, err = rootDB.Exec(fmt.Sprintf("CREATE DATABASE %s", targetDBName))
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", targetDBName, err)
		}
		log.Printf("PostgreSQL database %s created successfully", targetDBName)
	}

	return nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notes (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT DEFAULT '',
		color VARCHAR(20) DEFAULT '#6366f1',
		is_pinned BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token VARCHAR(255) PRIMARY KEY,
		user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
	`

	_, err := db.Exec(schema)
	return err
}
