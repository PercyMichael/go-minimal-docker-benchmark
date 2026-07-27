package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"json-books-app/internal/db"
	"json-books-app/middleware"
	"json-books-app/models"
)

// HTMXHeaderHandler renders floating cozy header controls based on session state
func HTMXHeaderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	user, ok := middleware.GetAuthenticatedUser(r)

	if ok && user != nil {
		initial := "U"
		if len(user.Username) > 0 {
			initial = strings.ToUpper(user.Username[:1])
		}
		htmlStr := fmt.Sprintf(`
		<div class="flex items-center gap-3">
			<button @click="noteOpen = true" class="inline-flex items-center gap-2 px-4 py-2 bg-amber-700 hover:bg-amber-800 text-amber-50 font-bold text-xs uppercase tracking-wider rounded-full shadow-sm hover:shadow transition-all active:scale-95">
				<i class="ri-add-line text-sm"></i> <span>New Note</span>
			</button>
			<div class="flex items-center gap-2 px-3 py-1.5 bg-amber-50/80 border border-amber-900/10 rounded-full">
				<div class="w-5 h-5 rounded-full bg-amber-700 text-white text-[10px] font-extrabold flex items-center justify-center shadow-2xs">%s</div>
				<span class="text-xs font-bold text-amber-950">%s</span>
			</div>
			<button hx-post="/htmx/auth/logout" class="inline-flex items-center gap-1 px-3 py-1.5 text-slate-500 hover:text-amber-900 hover:bg-amber-100/50 rounded-full text-xs font-bold transition-all" title="Sign Out">
				<i class="ri-logout-box-r-line text-sm"></i> <span>Log Out</span>
			</button>
		</div>`, html.EscapeString(initial), html.EscapeString(user.Username))
		_, _ = w.Write([]byte(htmlStr))
		return
	}

	_, _ = w.Write([]byte(`
	<div class="flex items-center gap-2">
		<button @click="authOpen = true; authMode = 'login'" class="px-4 py-2 text-slate-600 hover:text-slate-900 hover:bg-amber-100/50 rounded-full text-xs font-bold transition-all">Log In</button>
		<button @click="authOpen = true; authMode = 'register'" class="px-4 py-2 bg-amber-700 hover:bg-amber-800 text-amber-50 text-xs font-bold uppercase tracking-wider rounded-full shadow-sm transition-all">Sign Up</button>
	</div>`))
}

// HTMXNotesHandler handles HTMX CRUD requests returning cozy note fragments
func HTMXNotesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetAuthenticatedUser(r)
	if !ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<div class="col-span-full text-center py-20 px-8 bg-white/90 backdrop-blur-md border border-amber-900/5 rounded-3xl max-w-md mx-auto shadow-sm">
			<div class="w-16 h-16 bg-amber-100/60 text-amber-800 rounded-full flex items-center justify-center text-3xl mx-auto mb-4">
				<i class="ri-book-read-line"></i>
			</div>
			<h3 class="text-xl font-extrabold text-slate-800 mb-2 font-serif">Welcome to ZenNotes</h3>
			<p class="text-sm text-slate-500 mb-6 leading-relaxed">Sign in or create a free account to start writing your notes in a warm, cozy space.</p>
			<button @click="authOpen = true; authMode = 'login'" class="px-6 py-2.5 bg-amber-700 hover:bg-amber-800 text-amber-50 text-xs font-bold uppercase tracking-wider rounded-full shadow-sm transition-all">Get Started</button>
		</div>`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.Method {
	case http.MethodGet:
		renderNotesListHTMX(w, r, user.ID)
	case http.MethodPost:
		createNoteHTMX(w, r, user.ID)
	case http.MethodDelete:
		deleteNoteHTMX(w, r, user.ID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HTMXLoginHandler authenticates user via HTMX form submission
func HTMXLoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.ParseForm(); err != nil {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Invalid input</div>`))
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))

	if email == "" || password == "" {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Email and password are required</div>`))
		return
	}

	var user models.User
	var passwordHash string
	query := "SELECT id, username, email, password_hash, created_at FROM users WHERE email = $1 OR username = $2"
	err := db.DB.QueryRow(query, email, email).Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.CreatedAt)
	if err != nil || !models.CheckPassword(password, passwordHash) {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Invalid email or password</div>`))
		return
	}

	session, err := middleware.CreateSession(user.ID)
	if err != nil {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Failed to create session</div>`))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("HX-Refresh", "true")
	_, _ = w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 text-xs rounded-xl font-medium mb-3">Logging in...</div>`))
}

// HTMXRegisterHandler creates new user via HTMX form
func HTMXRegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.ParseForm(); err != nil {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Invalid input</div>`))
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))

	reg := models.UserRegistration{Username: username, Email: email, Password: password}
	if err := reg.Validate(); err != nil {
		_, _ = w.Write([]byte(fmt.Sprintf(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">%s</div>`, html.EscapeString(err.Error()))))
		return
	}

	hash, err := models.HashPassword(password)
	if err != nil {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Failed to hash password</div>`))
		return
	}

	var userID int64
	query := "INSERT INTO users (username, email, password_hash, created_at) VALUES ($1, $2, $3, $4) RETURNING id"
	err = db.DB.QueryRow(query, username, email, hash, time.Now()).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Username or email already exists</div>`))
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Error: %s</div>`, html.EscapeString(err.Error()))))
		return
	}

	session, err := middleware.CreateSession(userID)
	if err != nil {
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium mb-3">Session creation failed</div>`))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("HX-Refresh", "true")
	_, _ = w.Write([]byte(`<div class="p-3 bg-emerald-50 text-emerald-700 text-xs rounded-xl font-medium mb-3">Account created!</div>`))
}

// HTMXLogoutHandler clears session via HTMX
func HTMXLogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = middleware.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func renderNotesListHTMX(w http.ResponseWriter, r *http.Request, userID int64) {
	querySearch := strings.TrimSpace(r.URL.Query().Get("q"))
	tagFilter := strings.TrimSpace(r.URL.Query().Get("tag"))

	query := `SELECT id, user_id, title, content, tags, color, is_pinned, created_at, updated_at FROM notes WHERE user_id = $1`
	args := []interface{}{userID}
	paramIdx := 2

	if querySearch != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR content ILIKE $%d OR tags ILIKE $%d)", paramIdx, paramIdx, paramIdx)
		args = append(args, "%"+querySearch+"%")
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
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`<div class="p-4 bg-rose-50 text-rose-600 rounded-2xl text-sm font-medium">Error: %s</div>`, html.EscapeString(err.Error()))))
		return
	}
	defer rows.Close()

	var sb strings.Builder
	count := 0
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.Color, &n.IsPinned, &n.CreatedAt, &n.UpdatedAt); err == nil {
			sb.WriteString(renderNoteCardSnippet(n))
			count++
		}
	}

	if count == 0 {
		_, _ = w.Write([]byte(`
			<div class="col-span-full text-center py-20 px-8 bg-white border border-amber-900/5 rounded-3xl max-w-md mx-auto shadow-sm">
				<div class="w-16 h-16 bg-amber-100/60 text-amber-800 rounded-full flex items-center justify-center text-3xl mx-auto mb-4">
					<i class="ri-draft-line"></i>
				</div>
				<h3 class="text-xl font-extrabold text-slate-800 mb-1 font-serif">No notes yet</h3>
				<p class="text-sm text-slate-500">Click "New Note" above to write your first cozy thought.</p>
			</div>
		`))
		return
	}

	_, _ = w.Write([]byte(sb.String()))
}

func createNoteHTMX(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	tags := strings.TrimSpace(r.FormValue("tags"))
	color := strings.TrimSpace(r.FormValue("color"))
	if color == "" {
		color = "#d97706"
	}
	isPinned := r.FormValue("is_pinned") == "true" || r.FormValue("is_pinned") == "on"

	if title == "" && content == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium">Title or content is required</div>`))
		return
	}

	now := time.Now()
	var noteID int64
	query := "INSERT INTO notes (user_id, title, content, tags, color, is_pinned, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id"
	err := db.DB.QueryRow(query, userID, title, content, tags, color, isPinned, now, now).Scan(&noteID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`<div class="p-3 bg-rose-50 text-rose-600 text-xs rounded-xl font-medium">Error: %s</div>`, html.EscapeString(err.Error()))))
		return
	}

	note := models.Note{
		ID:        noteID,
		UserID:    userID,
		Title:     title,
		Content:   content,
		Tags:      tags,
		Color:     color,
		IsPinned:  isPinned,
		CreatedAt: now,
		UpdatedAt: now,
	}

	w.Header().Set("HX-Trigger", "noteChanged")
	_, _ = w.Write([]byte(renderNoteCardSnippet(note)))
}

func deleteNoteHTMX(w http.ResponseWriter, r *http.Request, userID int64) {
	idStr := r.URL.Query().Get("id")
	noteID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM notes WHERE id = $1 AND user_id = $2", noteID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func renderNoteCardSnippet(note models.Note) string {
	formattedDate := note.UpdatedAt.Format("Jan 02, 2006")
	tagList := note.Tags
	var tagsHTML strings.Builder
	if tagList != "" {
		for _, t := range strings.Split(tagList, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagsHTML.WriteString(fmt.Sprintf(`<span class="inline-flex items-center px-2.5 py-1 rounded-full text-[11px] font-medium bg-amber-50 text-amber-900/80 border border-amber-200/50">#%s</span> `, html.EscapeString(t)))
			}
		}
	}

	pinBadge := ""
	if note.IsPinned {
		pinBadge = `<span class="inline-flex items-center gap-1 text-[11px] font-bold text-amber-900 bg-amber-100/80 px-2.5 py-0.5 rounded-full border border-amber-300/40"><i class="ri-pushpin-fill text-amber-600"></i> Pinned</span>`
	}

	return fmt.Sprintf(`
	<div class="group relative bg-white border border-amber-900/5 rounded-3xl p-6 shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 flex flex-col justify-between" id="note-card-%d">
		<div>
			<div class="flex items-center justify-between gap-2 mb-3">
				<div class="flex items-center gap-2">
					<div class="w-3 h-3 rounded-full shadow-2xs" style="background-color: %s;"></div>
					<h3 class="font-serif font-bold text-slate-800 text-lg leading-snug tracking-tight group-hover:text-amber-800 transition-colors">%s</h3>
				</div>
				<div class="flex items-center gap-1">
					%s
				</div>
			</div>
			<p class="text-slate-600 text-sm leading-relaxed mb-5 whitespace-pre-wrap line-clamp-5 font-normal">%s</p>
		</div>
		<div class="flex items-center justify-between pt-4 border-t border-slate-100 mt-auto">
			<div class="flex flex-wrap gap-1">%s</div>
			<div class="flex items-center gap-2">
				<span class="text-[11px] font-medium text-slate-400">%s</span>
				<button class="p-1.5 text-slate-300 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition-colors" hx-delete="/htmx/notes?id=%d" hx-target="#note-card-%d" hx-swap="outerHTML swap:0.2s" title="Delete Note">
					<i class="ri-delete-bin-line text-sm"></i>
				</button>
			</div>
		</div>
	</div>`,
		note.ID,
		html.EscapeString(note.Color),
		html.EscapeString(note.Title),
		pinBadge,
		html.EscapeString(note.Content),
		tagsHTML.String(),
		formattedDate,
		note.ID,
		note.ID,
	)
}
