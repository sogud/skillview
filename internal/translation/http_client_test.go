package translation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimitHintRetriesAllProviders(t *testing.T) {
	for _, provider := range []string{"google", "google-cloud", "ai"} {
		t.Run(provider, func(t *testing.T) {
			var calls atomic.Int32
			var firstBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if calls.Add(1) == 1 {
					firstBody = string(body)
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				if string(body) != firstBody {
					t.Error("retry changed the requested content")
				}
				w.Header().Set("Content-Type", "application/json")
				switch provider {
				case "google":
					_, _ = w.Write([]byte(`[[["你好。","Hello."]],null,"en"]`))
				case "google-cloud":
					_, _ = w.Write([]byte(`{"data":{"translations":[{"translatedText":"你好。"}]}}`))
				case "ai":
					_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"translations\":[\"你好。\"]}"}}]}`))
				}
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			s.googleURL, s.cloudURL = server.URL, server.URL
			if err := s.Update(Settings{Provider: provider, TargetLanguage: "zh-CN", BaseURL: server.URL + "/v1", Model: "fixture"}, "fixture", "fixture", false, false); err != nil {
				t.Fatal(err)
			}
			if err := s.TestConnection(context.Background()); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 {
				t.Fatal("unexpected retry count", calls.Load())
			}
		})
	}
}

func TestRateLimitRetriesAreBoundedAndRespectHints(t *testing.T) {
	for _, item := range []struct {
		name   string
		status int
		hint   string
		calls  int32
	}{
		{"bounded", 429, "0", 3},
		{"http-date", 429, time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 3},
		{"missing", 429, "", 1},
		{"invalid", 429, "invalid", 1},
		{"negative", 429, "-1", 1},
		{"long", 429, "120", 1},
		{"overflow", 429, "9223372036854775807", 1},
		{"long-date", 429, time.Now().Add(time.Hour).UTC().Format(http.TimeFormat), 1},
		{"unavailable", 503, "0", 1},
		{"unauthorized", 401, "0", 1},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Retry-After", item.hint)
				w.WriteHeader(item.status)
				_, _ = w.Write([]byte("provider-private-error"))
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			s.googleURL = server.URL
			_, err := s.Translate(context.Background(), "Hello.", "r")
			if err == nil || strings.Contains(err.Error(), "provider-private") {
				t.Fatal("missing or leaking rate-limit error", err)
			}
			if calls.Load() != item.calls || len(s.cache) != 0 {
				t.Fatal("unexpected retries or cached failure", calls.Load(), len(s.cache))
			}
		})
	}
}

func TestCancellationInterruptsRateLimitWait(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := s.Translate(ctx, "Hello.", "r")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatal("cancellation did not interrupt server-requested wait", err)
	}
	if calls.Load() != 1 || len(s.cache) != 0 {
		t.Fatal("cancellation retried or cached a result")
	}
}
