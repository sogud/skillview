package translation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAIKeepsMarkdownCodeLinksAndCaches(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("SDK route: %s", r.URL.Path)
		}
		var request struct {
			Messages []struct{ Role, Content string }
			Model    string
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Model != "fixture" || len(request.Messages) != 2 || request.Messages[0].Role != "system" {
			t.Errorf("request: %+v", request)
		}
		var parts []string
		_ = json.Unmarshal([]byte(request.Messages[1].Content), &parts)
		output := []string{}
		for _, part := range parts {
			if strings.Contains(part, "DO_NOT_TRANSLATE") {
				t.Error("code was sent")
			}
			output = append(output, "译文")
		}
		content, _ := json.Marshal(map[string]any{"translations": output})
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "fixture", "object": "chat.completion", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": string(content)}}}})
	}))
	defer server.Close()
	s, err := New(t.TempDir(), NewMemoryKeys())
	if err != nil {
		t.Fatal(err)
	}
	settings := Settings{Provider: "ai", TargetLanguage: "zh-CN", BaseURL: server.URL + "/v1", Model: "fixture"}
	if err = s.Update(settings, "", "", false, false); err != nil {
		t.Fatal(err)
	}
	body := "# Heading\n\nRead **this** and [guide](references/guide.md), `DO_NOT_TRANSLATE`.\n\n```sh\nDO_NOT_TRANSLATE\n```\n\n| Title | Value |\n| --- | --- |\n| Sample | Text |\n"
	result, err := s.Translate(context.Background(), body, "revision")
	if err != nil {
		t.Fatal(err)
	}
	for _, kept := range []string{"# 译文", "**译文**", "[译文](references/guide.md)", "`DO_NOT_TRANSLATE`", "```sh\nDO_NOT_TRANSLATE\n```", "| --- | --- |"} {
		if !strings.Contains(result.Text, kept) {
			t.Fatalf("lost %q in %s", kept, result.Text)
		}
	}
	again, err := s.Translate(context.Background(), body, "revision")
	if err != nil || !again.Cached || calls.Load() != 1 {
		t.Fatalf("cache: %+v %v %d", again, err, calls.Load())
	}
	settings.TargetLanguage = "en"
	_ = s.Update(settings, "", "", false, false)
	_, err = s.Translate(context.Background(), body, "revision")
	if err != nil || calls.Load() != 2 {
		t.Fatal("cache reused across languages", err)
	}
}

func TestIncompleteAndHTTPFailuresReturnNoPartialTranslation(t *testing.T) {
	for _, check := range []struct {
		name    string
		status  int
		content string
	}{
		{"missing sections", 200, `{"translations":[]}`},
		{"empty sections", 200, `{"translations":[""," "]}`},
		{"rate limited", 429, `{"translations":[]}`},
	} {
		t.Run(check.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(check.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": check.content}}}})
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			_ = s.Update(Settings{Provider: "ai", TargetLanguage: "en", BaseURL: server.URL + "/v1", Model: "fixture"}, "fixture-secret", "", false, false)
			result, err := s.Translate(context.Background(), "# Heading\n\nText.", "r")
			if err == nil || result.Text != "" || len(s.cache) != 0 || strings.Contains(err.Error(), "fixture-secret") {
				t.Fatalf("partial/leaking result: %+v %v", result, err)
			}
		})
	}
}

func TestGoogleAndCodeOnlyDocument(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = r.ParseForm()
		if r.Form.Get("tl") != "zh-CN" || r.Form.Get("q") == "" {
			t.Error("missing Google request fields")
		}
		lines := strings.Split(r.Form.Get("q"), "\n")
		translated := strings.TrimSuffix(strings.Repeat("你好\n", len(lines)), "\n")
		_ = json.NewEncoder(w).Encode([]any{[]any{[]any{translated, "Hello"}}, nil, "en"})
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.googleURL = server.URL
	result, err := s.Translate(context.Background(), "# Hello\n\n- Hello **Hello**\n", "r")
	if err != nil || !strings.Contains(result.Text, "- 你好 **你好**") {
		t.Fatalf("whitespace/markup: %q %v", result.Text, err)
	}
	before := calls.Load()
	code := "```sh\necho hello\n```\n"
	result, err = s.Translate(context.Background(), code, "code")
	if err != nil || result.Text != code || calls.Load() != before {
		t.Fatal("code-only document called provider", err)
	}
}

func TestSettingsKeyRollbackAndURLValidation(t *testing.T) {
	keys := NewMemoryKeys()
	_ = keys.Set("ai", "old-secret")
	root := t.TempDir()
	s, _ := New(root, keys)
	// A directory prevents the atomic settings file from replacing this path.
	_ = os.Mkdir(filepath.Join(root, "translation.json"), 0700)
	err := s.Update(Settings{Provider: "google", TargetLanguage: "zh-CN"}, "new-secret", "", false, false)
	key, _ := keys.Get("ai")
	if err == nil || key != "old-secret" {
		t.Fatal("key changed after failed save", err, key)
	}
	for _, raw := range []string{"http://example.com/v1", "https://user:password@example.com/v1", "https://example.com/v1?key=secret", "file:///tmp/model"} {
		if _, err := validateBaseURL(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, raw := range []string{"https://example.com/v1", "http://127.0.0.1:11434/v1", "http://localhost:1234/v1"} {
		if _, err := validateBaseURL(raw); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAIResponseSizeIsBoundedAndNotCached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat(" ", 2*1024*1024+1)))
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	if err := s.Update(Settings{Provider: "ai", TargetLanguage: "en", BaseURL: server.URL + "/v1", Model: "fixture"}, "", "", false, false); err != nil {
		t.Fatal(err)
	}
	result, err := s.Translate(context.Background(), "# Hello", "r")
	if err == nil || result.Text != "" || len(s.cache) != 0 {
		t.Fatal("oversized response accepted or cached", result, err)
	}
}
