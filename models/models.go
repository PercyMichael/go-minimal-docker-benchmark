package models

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a registered user account
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserRegistration input payload
type UserRegistration struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks registration fields
func (r *UserRegistration) Validate() error {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Password = strings.TrimSpace(r.Password)

	if r.Username == "" {
		return errors.New("username is required")
	}
	if len(r.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	if r.Email == "" || !strings.Contains(r.Email, "@") {
		return errors.New("a valid email address is required")
	}
	if len(r.Password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}
	return nil
}

// UserLogin input payload
type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *UserLogin) Validate() error {
	l.Email = strings.TrimSpace(strings.ToLower(l.Email))
	l.Password = strings.TrimSpace(l.Password)

	if l.Email == "" {
		return errors.New("email is required")
	}
	if l.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// HashPassword hashes plain text password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares plain password with hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Note represents a user note
type Note struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      string    `json:"tags"`
	Color     string    `json:"color"`
	IsPinned  bool      `json:"is_pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NoteInput payload for create and edit operations
type NoteInput struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Tags     string `json:"tags"`
	Color    string `json:"color"`
	IsPinned bool   `json:"is_pinned"`
}

func (n *NoteInput) Validate() error {
	n.Title = strings.TrimSpace(n.Title)
	n.Content = strings.TrimSpace(n.Content)
	n.Tags = strings.TrimSpace(n.Tags)
	n.Color = strings.TrimSpace(n.Color)

	if n.Title == "" && n.Content == "" {
		return errors.New("note must have a title or content")
	}
	if n.Color == "" {
		n.Color = "#4f46e5"
	}
	return nil
}

// Session represents an active user session token
type Session struct {
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// APIResponse standard JSON envelope
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
