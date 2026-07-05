package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

type staticSSETransport struct {
	body string
}

func (t staticSSETransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Request:    req,
	}, nil
}

func TestTranslateStreamToolCallOutputIndexIsStable(t *testing.T) {
	upstream := strings.Join([]string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"shell","arguments":"{\"cmd\""}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"ls\"}"}}]}}]}`,
		`data: [DONE]`,
		``,
	}, "\n\n")
	client := &http.Client{Transport: staticSSETransport{body: upstream}}
	rec := httptest.NewRecorder()

	err := TranslateStream(context.Background(), RelayStreamTranslator{
		Client:      client,
		UpstreamURL: "https://example.test/v1/chat/completions",
		ChatReq: &ChatRequest{
			Model:    "test-model",
			Messages: []ChatMessage{{Role: "user", Content: "run ls"}},
			Stream:   true,
		},
		ResponseID: "resp_test",
		Sessions:   NewRelaySessionStore(),
		Model:      "test-model",
	}, rec, rec)
	if err != nil {
		t.Fatalf("TranslateStream: %v", err)
	}

	out := rec.Body.String()
	if strings.Contains(out, `"output_index":-1`) {
		t.Fatalf("tool stream used invalid output_index: %s", out)
	}
	for _, event := range []string{
		"response.output_item.added",
		"response.function_call_arguments.delta",
		"response.function_call_arguments.done",
		"response.output_item.done",
	} {
		if !strings.Contains(out, event) {
			t.Fatalf("missing %s in stream: %s", event, out)
		}
	}

	indexes := regexp.MustCompile(`"output_index":([0-9]+)`).FindAllStringSubmatch(out, -1)
	if len(indexes) < 4 {
		t.Fatalf("expected multiple output_index values, got %v in %s", indexes, out)
	}
	want := indexes[0][1]
	for _, match := range indexes[:4] {
		if match[1] != want {
			t.Fatalf("tool output_index changed from %s to %s in stream: %s", want, match[1], out)
		}
	}
}
