package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"json-books-app/db"
	"json-books-app/middleware"
	"json-books-app/models"
)

// RegisterHandler registers a new user
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	var req models.UserRegistration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid JSON payload",
		})
		return
	}

	if err := req.Validate(); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Check if username or email already exists
	var existingID int64
	err := db.DB.QueryRow("SELECT id FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&existingID)
	if err == nil {
		respondJSON(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "Username or email address is already registered",
		})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Database error checking existing user",
		})
		return
	}

	hash, err := models.HashPassword(req.Password)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to hash password",
		})
		return
	}

	var userID int64
	query := "INSERT INTO users (username, email, password_hash, created_at) VALUES ($1, $2, $3, $4) RETURNING id"
	err = db.DB.QueryRow(query, req.Username, req.Email, hash, time.Now()).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "unique constraint") {
			respondJSON(w, http.StatusConflict, models.APIResponse{
				Success: false,
				Error:   "Username or email address is already registered",
			})
			return
		}
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to create user: " + err.Error(),
		})
		return
	}

	session, err := middleware.CreateSession(userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "User created, but session creation failed",
		})
		return
	}

	setSessionCookie(w, session.Token, session.ExpiresAt)

	user := models.User{
		ID:        userID,
		Username:  req.Username,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "User registered successfully",
		Data: map[string]interface{}{
			"user":          user,
			"session_token": session.Token,
		},
	})
}

// LoginHandler authenticates a user and starts a session
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	var req models.UserLogin
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid JSON payload",
		})
		return
	}

	if err := req.Validate(); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	var user models.User
	var passwordHash string

	query := "SELECT id, username, email, password_hash, created_at FROM users WHERE email = $1 OR username = $2"
	err := db.DB.QueryRow(query, req.Email, req.Email).Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSON(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Error:   "Invalid email or password",
			})
			return
		}
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Database error during login",
		})
		return
	}

	if !models.CheckPassword(req.Password, passwordHash) {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Invalid email or password",
		})
		return
	}

	session, err := middleware.CreateSession(user.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to create login session",
		})
		return
	}

	setSessionCookie(w, session.Token, session.ExpiresAt)

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Logged in successfully",
		Data: map[string]interface{}{
			"user":          user,
			"session_token": session.Token,
		},
	})
}

// LogoutHandler terminates current session
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = middleware.DeleteSession(cookie.Value)
	}

	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		_ = middleware.DeleteSession(authHeader[7:])
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// MeHandler returns currently authenticated user details
func MeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetAuthenticatedUser(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    user,
	})
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func respondJSON(w http.ResponseWriter, status int, resp models.APIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
