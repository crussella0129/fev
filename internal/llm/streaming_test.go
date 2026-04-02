package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crussella0129/fev/internal/core"
)

func TestGenerateStream_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{"Hello", " world", "!"}
		for i, chunk := range chunks {
			data := fmt.Sprintf(`{"id":"test","choices":[{"delta":{"content":"%s"},"finish_reason":null}]}`, chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			_ = i
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "test-model")
	var collected []string
	onChunk := func(s string) { collected = append(collected, s) }

	resp, err := client.GenerateStream(context.Background(), []core.Message{core.NewUserMessage("hi")}, nil, onChunk)
	if err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	if resp.Content != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", resp.Content)
	}
	if len(collected) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(collected))
	}
}

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		line     string
		wantData string
		wantDone bool
	}{
		{"data: {\"test\":true}", `{"test":true}`, false},
		{"data: [DONE]", "", true},
		{": comment", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			data, done := parseSSELine(tt.line)
			if done != tt.wantDone {
				t.Errorf("done: got %v, want %v", done, tt.wantDone)
			}
			if data != tt.wantData {
				t.Errorf("data: got %q, want %q", data, tt.wantData)
			}
		})
	}
}
