package transfer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeServerURL(t *testing.T) {
	got, err := NormalizeServerURL("127.0.0.1:8080/")
	if err != nil {
		t.Fatalf("NormalizeServerURL() error = %v", err)
	}
	if got != "http://127.0.0.1:8080" {
		t.Fatalf("NormalizeServerURL() = %q, want http://127.0.0.1:8080", got)
	}
}

func TestClientHealthBrowseAndDownload(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.URL.Path == "/files/" && r.URL.Query().Get("format") == "json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"path":"/files/","entries":[{"name":"docs","path":"/files/docs","is_dir":true,"size":0,"last_modified":"2026-03-17T00:00:00Z"},{"name":"hello.txt","path":"/files/hello.txt","is_dir":false,"size":5,"last_modified":"2026-03-17T01:00:00Z"}]}`))
		case r.URL.Path == "/files/hello.txt" && r.URL.Query().Get("download") == "1":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello"))
		case r.URL.Path == "/files/hello.txt" && r.URL.Query().Get("format") == "json":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello"))
		default:
			http.NotFound(w, r)
		}
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	directory, err := client.Browse(ctx, "/")
	if err != nil {
		t.Fatalf("Browse() error = %v", err)
	}
	if directory.Path != "/" {
		t.Fatalf("directory.Path = %q, want /", directory.Path)
	}
	if len(directory.Entries) != 2 {
		t.Fatalf("len(directory.Entries) = %d, want 2", len(directory.Entries))
	}

	var builder strings.Builder
	if err := client.Download(ctx, "/hello.txt", &builder, nil); err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if builder.String() != "hello" {
		t.Fatalf("downloaded content = %q, want hello", builder.String())
	}
}

func TestBrowseFilePathReturnsHelpfulError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "hello")
	}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.Browse(ctx, "/hello.txt")
	if err == nil {
		t.Fatalf("Browse() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "download it instead") {
		t.Fatalf("Browse() error = %q, want file hint", err)
	}
}

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()

	baseURL, err := url.Parse("http://test.local")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, req)
				return recorder.Result(), nil
			}),
		},
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
