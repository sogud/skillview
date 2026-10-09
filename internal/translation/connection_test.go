package translation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectionUsesOnlySampleAndBypassesCache(t *testing.T) {
	for _, provider := range []string{"google", "google-cloud", "ai"} {
		t.Run(provider, func(t *testing.T) {
			var calls atomic.Int32
			var fail atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				switch provider {
				case "google":
					_ = r.ParseForm()
					if r.Form.Get("q") != "Hello." {
						t.Error("test sent unexpected content")
					}
				case "google-cloud":
					var request struct{ Q []string }
					_ = json.NewDecoder(r.Body).Decode(&request)
					if len(request.Q) != 1 || request.Q[0] != "Hello." {
						t.Error("test sent unexpected content")
					}
				case "ai":
					var request struct{ Messages []struct{ Content string } }
					_ = json.NewDecoder(r.Body).Decode(&request)
					if len(request.Messages) != 2 || request.Messages[1].Content != `["Hello."]` {
						t.Error("test sent unexpected content")
					}
				}
				if fail.Load() {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("provider-private-error"))
					return
				}
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
			s, err := New(t.TempDir(), NewMemoryKeys())
			if err != nil {
				t.Fatal(err)
			}
			s.googleURL = server.URL
			s.cloudURL = server.URL
			if err := s.Update(Settings{Provider: provider, TargetLanguage: "zh-CN", BaseURL: server.URL + "/v1", Model: "fixture"}, "fixture-key", "fixture-key", false, false); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if err := s.TestConnection(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if calls.Load() != 2 || len(s.cache) != 0 {
				t.Fatal("test reused or populated cache")
			}
			fail.Store(true)
			if err := s.TestConnection(context.Background()); err == nil || err.Error() != "translation service returned HTTP 401" {
				t.Fatal("unexpected or leaking error", err)
			}
		})
	}
}

func TestTranslationCancellationStopsRequestAndDoesNotCache(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := s.Translate(ctx, "Hello.", "r"); finished <- err }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("request did not cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not finish after cancellation")
	}
	if len(s.cache) != 0 {
		t.Fatal("canceled translation cached")
	}
}
