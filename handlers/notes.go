package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"json-books-app/internal/db"
	"json-books-app/middleware"
	"json-books-app/models"
)

// NotesHandler routes /api/notes requests based on HTTP method and path query
func NotesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetAuthenticatedUser(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 3 && pathParts[0] == "api" && pathParts[1] == "notes" {
			idStr = pathParts[2]
		}
	}

	switch r.Method {
	case http.MethodGet:
		if idStr != "" {
			getSingleNote(w, r, user.ID, idStr)
		} else {
			listNotes(w, r, user.ID)
		}
	case http.MethodPost:
		createNote(w, r, user.ID)
	case http.MethodPut:
		if idStr == "" {
			respondJSON(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Note ID is required for update",
			})
			return
		}
		updateNote(w, r, user.ID, idStr)
	case http.MethodDelete:
		if idStr == "" {
			respondJSON(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Note ID is required for deletion",
			})
			return
		}
		deleteNote(w, r, user.ID, idStr)
	default:
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

func listNotes(w http.ResponseWriter, r *http.Request, userID int64) {
	querySearch := strings.TrimSpace(r.URL.Query().Get("q"))
	tagFilter := strings.TrimSpace(r.URL.Query().Get("tag"))

	query := `
	SELECT id, user_id, title, content, tags, color, is_pinned, created_at, updated_at 
	FROM notes 
	WHERE user_id = $1`
	args := []interface{}{userID}
	paramIdx := 2

	if querySearch != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR content ILIKE $%d OR tags ILIKE $%d)", paramIdx, paramIdx, paramIdx)
		searchPattern := "%" + querySearch + "%"
		args = append(args, searchPattern)
		paramIdx++
	}

	if tagFilter != "" {
		query += fmt.Sprintf(" AND tags ILIKE $%d", paramIdx)
		args = append(args, "%"+tagFilter+"%")
		paramIdx++
	}

	query += " ORDER BY is_pinned DESC, updated_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to fetch notes: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	notes := make([]models.Note, 0)
	for rows.Next() {
		var n models.Note
		err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.Color, &n.IsPinned, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Error:   "Failed to parse notes: " + err.Error(),
			})
			return
		}
		notes = append(notes, n)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    notes,
	})
}

func getSingleNote(w http.ResponseWriter, r *http.Request, userID int64, idStr string) {
	noteID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid note ID format",
		})
		return
	}

	var n models.Note
	query := `SELECT id, user_id, title, content, tags, color, is_pinned, created_at, updated_at FROM notes WHERE id = $1 AND user_id = $2`
	err = db.DB.QueryRow(query, noteID, userID).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.Color, &n.IsPinned, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSON(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "Note not found",
			})
			return
		}
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Database error fetching note",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    n,
	})
}

func createNote(w http.ResponseWriter, r *http.Request, userID int64) {
	var input models.NoteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid JSON payload",
		})
		return
	}

	if err := input.Validate(); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	now := time.Now()
	var noteID int64
	query := "INSERT INTO notes (user_id, title, content, tags, color, is_pinned, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id"
	err := db.DB.QueryRow(query, userID, input.Title, input.Content, input.Tags, input.Color, input.IsPinned, now, now).Scan(&noteID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to create note: " + err.Error(),
		})
		return
	}

	note := models.Note{
		ID:        noteID,
		UserID:    userID,
		Title:     input.Title,
		Content:   input.Content,
		Tags:      input.Tags,
		Color:     input.Color,
		IsPinned:  input.IsPinned,
		CreatedAt: now,
		UpdatedAt: now,
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Note created successfully",
		Data:    note,
	})
}

func updateNote(w http.ResponseWriter, r *http.Request, userID int64, idStr string) {
	noteID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid note ID format",
		})
		return
	}

	var input models.NoteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid JSON payload",
		})
		return
	}

	if err := input.Validate(); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	now := time.Now()
	res, err := db.DB.Exec(
		"UPDATE notes SET title = $1, content = $2, tags = $3, color = $4, is_pinned = $5, updated_at = $6 WHERE id = $7 AND user_id = $8",
		input.Title, input.Content, input.Tags, input.Color, input.IsPinned, now, noteID, userID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to update note: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Note not found or not owned by user",
		})
		return
	}

	getSingleNote(w, r, userID, idStr)
}

func deleteNote(w http.ResponseWriter, r *http.Request, userID int64, idStr string) {
	noteID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid note ID format",
		})
		return
	}

	res, err := db.DB.Exec("DELETE FROM notes WHERE id = $1 AND user_id = $2", noteID, userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to delete note: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Note not found or not owned by user",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Note deleted successfully",
	})
}
