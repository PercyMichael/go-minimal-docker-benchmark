package middleware

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"json-books-app/db"
	"json-books-app/models"
)

type contextKey string

const UserContextKey contextKey = "user"
const SessionCookieName = "session_token"
const SessionDuration = 7 * 24 * time.Hour

// GenerateSessionToken creates a random secure token string
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession inserts a new session token into PostgreSQL
func CreateSession(userID int64) (*models.Session, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(SessionDuration)
	_, err = db.DB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)", token, userID, expiresAt)
	if err != nil {
		return nil, err
	}

	return &models.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUserFromSession checks session token in PostgreSQL and returns the User
func GetUserFromSession(token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("empty session token")
	}

	var user models.User
	var expiresAt time.Time

	query := `
	SELECT u.id, u.username, u.email, u.created_at, s.expires_at 
	FROM sessions s 
	JOIN users u ON s.user_id = u.id 
	WHERE s.token = $1`

	err := db.DB.QueryRow(query, token).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired session")
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		// Session expired, delete it
		_, _ = db.DB.Exec("DELETE FROM sessions WHERE token = $1", token)
		return nil, errors.New("session expired")
	}

	return &user, nil
}

// DeleteSession removes session token from DB
func DeleteSession(token string) error {
	_, err := db.DB.Exec("DELETE FROM sessions WHERE token = $1", token)
	return err
}

// GetAuthenticatedUser extracts the User from the HTTP request context
func GetAuthenticatedUser(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	return user, ok && user != nil
}

// AuthRequired middleware enforces that the user is logged in
func AuthRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string

		// 1. Try Cookie
		cookie, err := r.Cookie(SessionCookieName)
		if err == nil && cookie.Value != "" {
			token = cookie.Value
		}

		// 2. Try Authorization header fallback ("Bearer <token>")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}

		if token == "" {
			respondJSON(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Error:   "Unauthorized: Please log in to continue",
			})
			return
		}

		user, err := GetUserFromSession(token)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     SessionCookieName,
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})

			respondJSON(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Error:   "Unauthorized: " + err.Error(),
			})
			return
		}

		// Inject user into context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// OptionalAuth middleware injects user if session is valid, but does not block request if unauthenticated
func OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		cookie, err := r.Cookie(SessionCookieName)
		if err == nil && cookie.Value != "" {
			token = cookie.Value
		}

		if token != "" {
			if user, err := GetUserFromSession(token); err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	}
}

func respondJSON(w http.ResponseWriter, status int, resp models.APIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
