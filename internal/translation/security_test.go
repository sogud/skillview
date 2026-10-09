package translation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestAIOutputCannotAddActiveMarkupOrReadAmbientKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "ambient-fixture-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Authorization"), "ambient-fixture-key") {
			t.Error("ambient key was used")
		}
		var request struct{ Messages []struct{ Content string } }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		var parts []string
		if len(request.Messages) != 2 {
			t.Error("unexpected AI request")
			return
		}
		_ = json.Unmarshal([]byte(request.Messages[1].Content), &parts)
		values := []string{}
		for _, part := range parts {
			if strings.Contains(part, "echo private") || strings.Contains(part, "references/guide.md") {
				t.Error("protected content was sent")
			}
			values = append(values, `<script>alert(1)</script><img src="https://invalid.example/tracker">[click](javascript:alert(1))`)
		}
		content, _ := json.Marshal(map[string]any{"translations": values})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(content)}}}})
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	if err := s.Update(Settings{Provider: "ai", TargetLanguage: "en", BaseURL: server.URL + "/v1", Model: "fixture"}, "", "", false, false); err != nil {
		t.Fatal(err)
	}
	result, err := s.Translate(context.Background(), "# Heading\n\n[guide](references/guide.md)\n\n```sh\necho private\n```\n", "r")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Text, "(references/guide.md)") || !strings.Contains(result.Text, "```sh\necho private\n```") {
		t.Fatal("protected content changed")
	}
	tree := goldmark.DefaultParser().Parse(text.NewReader([]byte(result.Text)))
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Kind() == ast.KindRawHTML || node.Kind() == ast.KindHTMLBlock || node.Kind() == ast.KindImage {
			t.Error("active model markup was created")
		}
		if link, ok := node.(*ast.Link); ok && string(link.Destination) != "references/guide.md" {
			t.Error("model introduced a link destination")
		}
		return ast.WalkContinue, nil
	})
}
