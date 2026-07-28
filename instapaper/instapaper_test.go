package instapaper

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAddCapsErrorResponseBody(t *testing.T) {
	api := New("user", "password")
	api.client = &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 128*1024)))}, nil
	})}
	err := api.Add("https://example.com", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated error, got %v", err)
	}
	if len(err.Error()) > 66*1024 {
		t.Fatalf("error was not capped: %d bytes", len(err.Error()))
	}
}
