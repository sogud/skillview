package translation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGoogleFollowsSameServiceRedirectWithoutLosingText(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Query().Get("q") != "Hello." {
			t.Error("Google request lost input")
		}
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/translate?"+r.URL.RawQuery, http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`[[["你好。","Hello."]],null,"en"]`))
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL + "/start"
	if err := s.TestConnection(context.Background()); err != nil || calls.Load() != 2 {
		t.Fatal("safe Google redirect failed", err, calls.Load())
	}
}

func TestAIRedirectPreservesBodyAndKeyOnlyInsideSameService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/complete" {
			http.Redirect(w, r, "/complete", http.StatusTemporaryRedirect)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || !strings.Contains(string(body), "Hello.") || r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Error("API redirect changed the request")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"translations":["你好。"]}`}}}})
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	if err := s.Update(Settings{Provider: "ai", TargetLanguage: "zh-CN", BaseURL: server.URL + "/v1", Model: "fixture"}, "fixture-key", "", false, false); err != nil {
		t.Fatal(err)
	}
	if err := s.TestConnection(context.Background()); err != nil {
		t.Fatal("safe AI redirect failed", err)
	}
}

func TestTranslationRefusesCrossServiceRedirectAndCachesNoFailure(t *testing.T) {
	var leaked atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1); w.WriteHeader(200) }))
	defer destination.Close()
	for _, provider := range []string{"google", "google-cloud", "ai"} {
		t.Run(provider, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, destination.URL+"/private-redirect", http.StatusTemporaryRedirect)
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			s.googleURL, s.cloudURL = server.URL, server.URL
			if err := s.Update(Settings{Provider: provider, TargetLanguage: "zh-CN", BaseURL: server.URL + "/v1", Model: "fixture"}, "fixture-key", "fixture-key", false, false); err != nil {
				t.Fatal(err)
			}
			_, err := s.Translate(context.Background(), "Hello.", "r")
			if err == nil || strings.Contains(err.Error(), destination.URL) || len(s.cache) != 0 {
				t.Fatal("unsafe or cached redirect result", err)
			}
		})
	}
	if leaked.Load() != 0 {
		t.Fatal("request forwarded to another service")
	}
}

func TestRedirectLoopIsBounded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Redirect(w, r, "/again?"+r.URL.RawQuery, http.StatusFound)
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL
	if err := s.TestConnection(context.Background()); err == nil || calls.Load() != 4 {
		t.Fatal("redirect loop was not bounded", err, calls.Load())
	}
}
