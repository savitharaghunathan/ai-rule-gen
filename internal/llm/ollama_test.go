package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/generate" {
			t.Errorf("expected /api/generate, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		body, _ := io.ReadAll(r.Body)
		var req ollamaRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if req.Model != "testmodel" {
			t.Errorf("expected model testmodel, got %s", req.Model)
		}
		if req.Stream != false {
			t.Error("expected stream=false")
		}

		resp := ollamaResponse{Response: "Hello from Ollama", Done: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &OllamaProvider{host: server.URL, model: "testmodel"}
	result, err := p.Complete(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello from Ollama" {
		t.Errorf("expected 'Hello from Ollama', got %q", result)
	}
}

func TestOllamaProvider_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaResponse{Error: "model not found"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &OllamaProvider{host: server.URL, model: "missing"}
	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if got := err.Error(); got != "ollama error: model not found" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestOllamaProvider_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	p := &OllamaProvider{host: server.URL, model: "llama3"}
	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestNewOllamaProvider_Defaults(t *testing.T) {
	// Unset env vars for this test
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("OLLAMA_MODEL", "")

	p, err := NewOllamaProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.host != "http://localhost:11434" {
		t.Errorf("expected default host, got %s", p.host)
	}
	if p.model != "llama3" {
		t.Errorf("expected default model llama3, got %s", p.model)
	}
}

func TestNewOllamaProvider_CustomEnv(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://myhost:1234")
	t.Setenv("OLLAMA_MODEL", "mistral")

	p, err := NewOllamaProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.host != "http://myhost:1234" {
		t.Errorf("expected custom host, got %s", p.host)
	}
	if p.model != "mistral" {
		t.Errorf("expected mistral, got %s", p.model)
	}
}
