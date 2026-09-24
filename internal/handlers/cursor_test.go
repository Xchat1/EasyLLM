package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestCursorRouter(t *testing.T) (*gin.Engine, *storage.CursorStorage) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	if err := db.AutoMigrate(&models.CursorAccount{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	store := storage.NewCursorStorage(db)
	h := NewCursorHandler(store)

	r := gin.New()
	rg := r.Group("/api/v1")
	h.RegisterRoutes(rg)
	return r, store
}

func TestCursorCRUD(t *testing.T) {
	r, _ := setupTestCursorRouter(t)

	// Create
	input := CreateCursorAccountInput{
		Email:        "test@example.com",
		SessionToken: "user_123::jwt_token_here",
	}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/cursor/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var created models.CursorAccount
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", created.Email)
	}

	// List
	reqList, _ := http.NewRequest(http.MethodGet, "/api/v1/cursor/accounts", nil)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wList.Code)
	}

	// Activate
	reqAct, _ := http.NewRequest(http.MethodPost, "/api/v1/cursor/accounts/"+created.ID+"/activate", nil)
	wAct := httptest.NewRecorder()
	r.ServeHTTP(wAct, reqAct)
	if wAct.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wAct.Code)
	}

	// Delete
	reqDel, _ := http.NewRequest(http.MethodDelete, "/api/v1/cursor/accounts/"+created.ID, nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wDel.Code)
	}
}
