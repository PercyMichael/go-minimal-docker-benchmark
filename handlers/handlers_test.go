package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"json-books-app/db"
	"json-books-app/handlers"
	"json-books-app/middleware"
	"json-books-app/models"
)

func setupTestDB(t *testing.T) {
	testDBName := "test_zennotes"
	database, err := db.InitDB(testDBName)
	if err != nil {
		t.Fatalf("Failed to initialize PostgreSQL test DB: %v", err)
	}

	// Truncate tables for a clean test run
	_, _ = database.Exec("TRUNCATE TABLE sessions, notes, users CASCADE;")
}

func TestAuthAndNotesLifecycle(t *testing.T) {
	setupTestDB(t)

	// 1. Register User
	regPayload := models.UserRegistration{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.RegisterHandler(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d", res.StatusCode)
	}

	var regResp models.APIResponse
	_ = json.NewDecoder(res.Body).Decode(&regResp)
	if !regResp.Success {
		t.Fatalf("Expected registration success, got error: %s", regResp.Error)
	}

	// 2. Duplicate Registration should fail
	reqDup := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	wDup := httptest.NewRecorder()
	handlers.RegisterHandler(wDup, reqDup)
	if wDup.Result().StatusCode != http.StatusConflict {
		t.Fatalf("Expected status 409 Conflict for duplicate user, got %d", wDup.Result().StatusCode)
	}

	// 3. Login
	loginPayload := models.UserLogin{
		Email:    "test@example.com",
		Password: "password123",
	}
	bodyLogin, _ := json.Marshal(loginPayload)
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyLogin))
	wLogin := httptest.NewRecorder()

	handlers.LoginHandler(wLogin, reqLogin)
	resLogin := wLogin.Result()
	if resLogin.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for login, got %d", resLogin.StatusCode)
	}

	cookies := resLogin.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == middleware.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("Expected session cookie to be set in login response")
	}

	// 4. Test Unauthenticated Access to Notes
	reqNotesUnauth := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	wNotesUnauth := httptest.NewRecorder()
	middleware.AuthRequired(handlers.NotesHandler)(wNotesUnauth, reqNotesUnauth)
	if wNotesUnauth.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 Unauthorized, got %d", wNotesUnauth.Result().StatusCode)
	}

	// 5. Create Note (Authenticated)
	noteInput := models.NoteInput{
		Title:    "First DevOps Note",
		Content:  "Deploying Go application with PostgreSQL and Docker.",
		Tags:     "devops, go, postgres",
		Color:    "#6366f1",
		IsPinned: true,
	}
	bodyNote, _ := json.Marshal(noteInput)
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewReader(bodyNote))
	reqCreate.AddCookie(sessionCookie)
	wCreate := httptest.NewRecorder()

	middleware.AuthRequired(handlers.NotesHandler)(wCreate, reqCreate)
	if wCreate.Result().StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 Created for note creation, got %d", wCreate.Result().StatusCode)
	}

	var noteResp models.APIResponse
	_ = json.NewDecoder(wCreate.Result().Body).Decode(&noteResp)
	if !noteResp.Success {
		t.Fatalf("Note creation failed: %s", noteResp.Error)
	}

	noteMap, ok := noteResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected note object in response data")
	}
	noteIDFloat := noteMap["id"].(float64)
	noteIDStr := strconvFormatInt(int64(noteIDFloat))

	// 6. List Notes
	reqList := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	reqList.AddCookie(sessionCookie)
	wList := httptest.NewRecorder()

	middleware.AuthRequired(handlers.NotesHandler)(wList, reqList)
	if wList.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for list notes, got %d", wList.Result().StatusCode)
	}

	// 7. Update Note
	updatedInput := models.NoteInput{
		Title:    "Updated DevOps Note",
		Content:  "Updated content for Go app with PostgreSQL.",
		Tags:     "devops, go, update",
		Color:    "#10b981",
		IsPinned: false,
	}
	bodyUpdated, _ := json.Marshal(updatedInput)
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/notes?id="+noteIDStr, bytes.NewReader(bodyUpdated))
	reqUpdate.AddCookie(sessionCookie)
	wUpdate := httptest.NewRecorder()

	middleware.AuthRequired(handlers.NotesHandler)(wUpdate, reqUpdate)
	if wUpdate.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for note update, got %d", wUpdate.Result().StatusCode)
	}

	// 8. Delete Note
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/notes?id="+noteIDStr, nil)
	reqDelete.AddCookie(sessionCookie)
	wDelete := httptest.NewRecorder()

	middleware.AuthRequired(handlers.NotesHandler)(wDelete, reqDelete)
	if wDelete.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for note deletion, got %d", wDelete.Result().StatusCode)
	}
}

func strconvFormatInt(i int64) string {
	var buf [20]byte
	n := 0
	if i == 0 {
		return "0"
	}
	for i > 0 {
		buf[n] = byte('0' + i%10)
		i /= 10
		n++
	}
	for j := 0; j < n/2; j++ {
		buf[j], buf[n-1-j] = buf[n-1-j], buf[j]
	}
	return string(buf[:n])
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
