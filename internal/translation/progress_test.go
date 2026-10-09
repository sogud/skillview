package translation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAIProgressUsesCompletedBatchesAndDoesNotCachePartialFailure(t *testing.T) {
	for _, failSecond := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "partial failure"}[failSecond], func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if failSecond && call == 2 {
					w.WriteHeader(503)
					return
				}
				var request struct{ Messages []struct{ Content string } }
				_ = json.NewDecoder(r.Body).Decode(&request)
				var input []string
				_ = json.Unmarshal([]byte(request.Messages[1].Content), &input)
				output := make([]string, len(input))
				for i := range output {
					output[i] = "译文"
				}
				content, _ := json.Marshal(map[string]any{"translations": output})
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": string(content)}}}})
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			_ = s.Update(Settings{Provider: "ai", TargetLanguage: "zh-CN", BaseURL: server.URL + "/v1", Model: "fixture"}, "", "", false, false)
			var last Progress
			result, err := s.Translate(context.Background(), strings.Repeat("Paragraph.\n\n", 30), "revision", func(p Progress) {
				if p.Completed < last.Completed || p.Total != 30 || p.Provider != "ai" {
					t.Fatalf("progress: %+v", p)
				}
				last = p
			})
			if failSecond {
				if err == nil || result.Text != "" || last.Completed != 24 || len(s.cache) != 0 {
					t.Fatal("partial translation marked complete", last, result, err)
				}
			} else if err != nil || last.Completed != 30 || result.Sections != 30 || calls.Load() != 2 {
				t.Fatal("complete translation progress", last, result, err)
			}
		})
	}
}
