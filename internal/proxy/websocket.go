package proxy

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow localhost / loopback origins (common for local tool usage)
		for _, prefix := range []string{
			"http://localhost", "https://localhost",
			"http://127.0.0.1", "https://127.0.0.1",
			"http://0.0.0.0", "https://0.0.0.0",
		} {
			if origin == prefix || len(origin) > len(prefix) && origin[len(prefix)] == ':' {
				return true
			}
		}
		return false
	},
}

// ProxyWebSocket handles WebSocket upgrade requests from Codex CLI.
// Codex CLI connects via wss:// for the /backend-api/codex/responses endpoint.
func (p *CodexProxy) ProxyWebSocket(w http.ResponseWriter, r *http.Request) {
	entry := p.matchIncomingToken(r)
	passthrough := entry != nil

	if !passthrough {
		if !p.enabled {
			writeError(w, http.StatusServiceUnavailable, "Proxy is disabled", "service_unavailable")
			return
		}
		entry = p.pickEntry()
		if entry == nil {
			writeError(w, http.StatusServiceUnavailable, "No available accounts in pool", "no_available_account")
			return
		}
	}

	upstreamURL := buildUpstreamWSURL(r.URL.Path, r.URL.RawQuery)

	reqHeader := http.Header{}
	if passthrough {
		if auth := r.Header.Get("Authorization"); auth != "" {
			reqHeader.Set("Authorization", auth)
		}
	} else {
		reqHeader.Set("Authorization", "Bearer "+entry.accessToken)
	}
	if entry.chatgptAccountID != "" {
		reqHeader.Set("chatgpt-account-id", entry.chatgptAccountID)
	}
	if ua := r.Header.Get("User-Agent"); ua != "" {
		reqHeader.Set("User-Agent", ua)
	} else {
		reqHeader.Set("User-Agent", "codex_cli_rs/0.98.0")
	}
	if beta := r.Header.Get("OpenAI-Beta"); beta != "" {
		reqHeader.Set("OpenAI-Beta", beta)
	} else {
		reqHeader.Set("OpenAI-Beta", "responses=experimental")
	}
	if orig := r.Header.Get("originator"); orig != "" {
		reqHeader.Set("originator", orig)
	} else {
		reqHeader.Set("originator", "codex_cli_rs")
	}
	for _, proto := range r.Header["Sec-Websocket-Protocol"] {
		reqHeader.Add("Sec-Websocket-Protocol", proto)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}
	upConn, upResp, err := dialer.Dial(upstreamURL, reqHeader)
	if err != nil {
		status := http.StatusBadGateway
		if upResp != nil {
			status = upResp.StatusCode
		}
		log.Printf("[ws-proxy] upstream dial failed: %v", err)
		writeError(w, status, "WebSocket upstream connection failed", "upstream_error")
		return
	}
	defer upConn.Close()

	clientConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws-proxy] client upgrade failed: %v", err)
		return
	}
	defer clientConn.Close()

	if entry.requests != nil {
		atomic.AddInt64(entry.requests, 1)
	}

	done := make(chan struct{})

	// upstream → client (server messages: responses, events). Messages are
	// streamed directly; EasyLLM does not retain API call logs.
	go func() {
		defer close(done)
		for {
			msgType, msg, err := upConn.ReadMessage()
			if err != nil {
				break
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				break
			}
		}
	}()

	// client → upstream (user sends prompts/config)
	go func() {
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				upConn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				break
			}
			if err := upConn.WriteMessage(msgType, msg); err != nil {
				break
			}
		}
	}()

	<-done
}

func extractWSUsage(msg []byte) (model string, inputTokens, outputTokens int64) {
	var obj map[string]interface{}
	if err := json.Unmarshal(msg, &obj); err != nil {
		return
	}

	evtType, _ := obj["type"].(string)

	resp, _ := obj["response"].(map[string]interface{})
	if resp == nil {
		resp = obj
	}
	if m, ok := resp["model"].(string); ok && m != "" {
		model = m
	}

	if evtType == "response.completed" || evtType == "response.done" {
		usage, _ := resp["usage"].(map[string]interface{})
		if usage != nil {
			if v, ok := usage["input_tokens"].(float64); ok {
				inputTokens = int64(v)
			}
			if v, ok := usage["output_tokens"].(float64); ok {
				outputTokens = int64(v)
			}
		}
	}
	return
}

func extractModelFromClientMsg(msg []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(msg, &obj); err != nil {
		return ""
	}
	if m, ok := obj["model"].(string); ok {
		return m
	}
	return ""
}

func buildUpstreamWSURL(path, query string) string {
	const base = "wss://chatgpt.com"
	mapped := mapCodexPath(path)
	if query != "" {
		return base + mapped + "?" + query
	}
	return base + mapped
}
