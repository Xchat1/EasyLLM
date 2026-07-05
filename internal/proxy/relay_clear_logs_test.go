package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleClearRelayLogsLeavesBufferEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewRelayLogStore()
	store.Log("info", "before clear", "", "")
	handler := &RelayHandler{Logs: store}

	router := gin.New()
	router.DELETE("/logs", handler.HandleClearRelayLogs)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/logs", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := len(store.Recent(10)); got != 0 {
		t.Fatalf("expected cleared log buffer, got %d entries", got)
	}
}
