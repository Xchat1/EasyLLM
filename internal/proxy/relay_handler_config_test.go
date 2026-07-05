package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleRelayModelsUsesConfiguredUpstreamPool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "secret" {
			t.Fatalf("api-key = %q, want secret", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"mimo-v2.5-pro"}]}`))
	}))
	defer upstream.Close()

	handler := &RelayHandler{
		Client: http.DefaultClient,
		Config: &RelayConfig{
			UpstreamURL: "https://api.openai.com/v1",
			Upstreams: []RelayUpstream{
				{
					ID:          "mimo",
					Name:        "MiMo",
					Enabled:     true,
					UpstreamURL: upstream.URL + "/v1",
					APIKey:      "secret",
					AuthHeader:  "api-key",
				},
			},
		},
	}

	router := gin.New()
	router.GET("/v1/models", handler.HandleRelayModels)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != `{"object":"list","data":[{"id":"mimo-v2.5-pro"}]}` {
		t.Fatalf("unexpected body: %s", got)
	}
}
