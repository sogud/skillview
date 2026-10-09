package translation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLibreTranslatePreservesMarkdownAndReportsBatches(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var request struct {
			Q                      []string
			Target, Source, Format string
			Key                    string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/instance/translate" || request.Target != "zh" || request.Source != "auto" || request.Format != "text" || request.Key != "fixture-key" || r.Header.Get("Authorization") != "" {
			t.Error("LibreTranslate request mismatch")
		}
		output := []string{}
		for _, text := range request.Q {
			if strings.Contains(text, "KEEP_CODE") {
				t.Error("Code sent to translation provider")
			}
			output = append(output, "译文")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translatedText": output})
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	if err := s.Update(Settings{Provider: "libretranslate", TargetLanguage: "zh-CN", LibreURL: server.URL + "/instance"}, "", "", false, false, KeyChange{Name: "libretranslate", Value: "fixture-key"}); err != nil {
		t.Fatal(err)
	}
	body := "# Heading\n\nRead [guide](references/guide.md) and `KEEP_CODE`.\n\n```sh\nKEEP_CODE\n```\n\n" + strings.Repeat("Paragraph.\n\n", 55)
	var last Progress
	result, err := s.Translate(context.Background(), body, "revision", func(progress Progress) { last = progress })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Text, "[译文](references/guide.md)") || !strings.Contains(result.Text, "```sh\nKEEP_CODE\n```") {
		t.Fatal("Markdown damaged")
	}
	if last.Completed != last.Total || last.Total != result.Sections || requests < 2 {
		t.Fatal("Missing batch progress", requests, last)
	}
	before := requests
	cached, err := s.Translate(context.Background(), body, "revision")
	if err != nil || !cached.Cached || requests != before {
		t.Fatal("Cache miss", requests, err)
	}
}

func TestLibreFailureDoesNotCachePartialDocument(t *testing.T) {
	for _, status := range []int{200, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"translatedText":[]}`))
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			_ = s.Update(Settings{Provider: "libretranslate", TargetLanguage: "en", LibreURL: server.URL}, "", "", false, false)
			result, err := s.Translate(context.Background(), "# Heading\n\nText.", "r")
			if err == nil || result.Text != "" || len(s.cache) != 0 {
				t.Fatal("Partial response cached", result, err)
			}
		})
	}
}

func TestLibreKeyStaysSeparateAndRollsBack(t *testing.T) {
	root := t.TempDir()
	keys := NewMemoryKeys()
	s, _ := New(root, keys)
	settings := Settings{Provider: "libretranslate", TargetLanguage: "en", LibreURL: "http://localhost:5000"}
	if err := s.Update(settings, "ai-secret", "", false, false, KeyChange{Name: "libretranslate", Value: "libre-secret"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "translation.json"))
	if strings.Contains(string(data), "secret") {
		t.Fatal("Key persisted in settings")
	}
	flags, err := s.Settings()
	if err != nil || !flags.HasAIKey || !flags.HasLibreKey {
		t.Fatal("Missing key flags", flags, err)
	}
	_ = os.Remove(filepath.Join(root, "translation.json"))
	_ = os.Mkdir(filepath.Join(root, "translation.json"), 0700)
	err = s.Update(settings, "", "", false, false, KeyChange{Name: "libretranslate", Value: "replacement"})
	key, _ := keys.Get("libretranslate")
	if err == nil || key != "libre-secret" {
		t.Fatal("Failed save changed key", key, err)
	}
	_ = os.Remove(filepath.Join(root, "translation.json"))
	if err = s.Update(settings, "", "", false, false, KeyChange{Name: "libretranslate", Clear: true}); err != nil {
		t.Fatal(err)
	}
	flags, _ = s.Settings()
	if flags.HasLibreKey || !flags.HasAIKey {
		t.Fatal("Clear removed another service's key")
	}
}

func TestGoogle429StopsRepeatedRequestsAndCanCancelPacing(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(429) }))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL
	for i := 0; i < 2; i++ {
		result, err := s.Translate(context.Background(), "# Hello\n\nText.", "r")
		if err == nil || result.Text != "" || len(s.cache) != 0 {
			t.Fatal("Rate limit accepted", result, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("Cooldown sent repeated requests", calls.Load())
	}
	s.webMu.Lock()
	s.webBlocked = nil
	s.webNext = time.Now().Add(time.Hour)
	s.webMu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.googleRequest(ctx, server.URL)
	if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatal("Canceled pacing sent request", err)
	}
	s.webMu.Lock()
	s.webNext = time.Time{}
	s.webBlocked = map[string]time.Time{"google": time.Now().Add(time.Minute)}
	s.webMu.Unlock()
	_, _ = s.webRequest(context.Background(), "youdao", http.MethodGet, server.URL, "", nil, nil)
	if calls.Load() != 2 {
		t.Fatal("Google cooldown blocked another provider")
	}
}

func TestGoogleCombinesSpansAndSpacesBatches(t *testing.T) {
	var previous time.Time
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if !previous.IsZero() && time.Since(previous) < 280*time.Millisecond {
			t.Error("Google burst request")
		}
		previous = time.Now()
		input := strings.Split(r.URL.Query().Get("q"), "\n")
		translated := strings.TrimSuffix(strings.Repeat("译文\n", len(input)), "\n")
		_ = json.NewEncoder(w).Encode([]any{[]any{[]any{translated, "source"}}, nil, "en"})
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL
	result, err := s.Translate(context.Background(), strings.Repeat("Paragraph.\n\n", 30), "r")
	if err != nil || result.Sections != 30 || calls != 2 {
		t.Fatal("Spans were not combined", calls, result, err)
	}
}

func TestLostBatchBoundariesAreRejected(t *testing.T) {
	output, err := translateWebBatches([]string{"Heading", "Text"}, func(string) (string, error) { return "merged translation", nil }, func(int) {})
	if err == nil || output != nil {
		t.Fatal("Lost line boundaries accepted", output, err)
	}
}
