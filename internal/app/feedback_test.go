package app

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"skillview/internal/translation"
)

func feedbackFixture(t *testing.T, ai bool) (FeedbackSubmission, []string) {
	t.Helper()
	s := &Service{translationDiagnostic: &translation.ResponseDiagnostic{Category: "invalid_format", FinishReason: "length", Truncation: "reported"}}
	p, err := s.PreviewFeedback(FeedbackInput{Message: "Synthetic feedback", IncludeAIError: ai})
	if err != nil {
		t.Fatal(err)
	}
	return p, fixtureEnvelopes(p)
}
func fixtureEnvelopes(p FeedbackSubmission) []string {
	var wires []string
	for _, event := range feedbackEvents(p) {
		kind := "event"
		if event["type"] == "feedback" {
			kind = "feedback"
		}
		h, _ := json.Marshal(map[string]any{"event_id": event["event_id"], "sent_at": p.Timestamp})
		i, _ := json.Marshal(map[string]any{"type": kind})
		e, _ := json.Marshal(event)
		wires = append(wires, string(h)+"\n"+string(i)+"\n"+string(e))
	}
	return wires
}
func setFeedbackTestDSN(t *testing.T, origin string) {
	t.Helper()
	t.Setenv("SKILLVIEW_FEEDBACK_DSN", strings.Replace(origin, "://", "://"+strings.Repeat("a", 32)+"@", 1)+"/1")
	feedbackRetryAfter = time.Time{}
}

func TestFeedbackPrivacyBoundary(t *testing.T) {
	s := &Service{}
	p, err := s.PreviewFeedback(FeedbackInput{Message: `Synthetic sk-1234567890 token=private /Users/example/private.txt user@example.com https://example.com/private`})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"1234567890", "private.txt", "example.com", "token="} {
		if strings.Contains(p.Message, secret) {
			t.Fatal("redaction missed", secret)
		}
	}
	if err = validateFeedback(p, fixtureEnvelopes(p)); err != nil {
		t.Fatal(err)
	}
	p, wires := feedbackFixture(t, true)
	if err = validateFeedback(p, wires); err != nil {
		t.Fatal(err)
	}
	wires[1] = strings.TrimSuffix(wires[1], "}") + `,"user":{"email":"private@example.com"}}`
	if validateFeedback(p, wires) == nil {
		t.Fatal("accepted ambient private field")
	}
	p.Diagnostics.AIResponse.FinishReason = "private response"
	if validateFeedback(p, fixtureEnvelopes(p)) == nil {
		t.Fatal("accepted arbitrary diagnostic")
	}
}
func TestFeedbackReceiptPartialRetryAndDedupe(t *testing.T) {
	p, wires := feedbackFixture(t, true)
	seen := map[string]bool{}
	failOnce := true
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/1/envelope/" || r.Header.Get("Origin") != "" || !strings.Contains(r.Header.Get("X-Sentry-Auth"), "sentry_key=") {
			t.Error("invalid native Sentry request")
		}
		var h struct {
			ID string `json:"event_id"`
		}
		if json.NewDecoder(r.Body).Decode(&h) != nil {
			t.Error("invalid envelope")
		}
		if h.ID == p.EventID && failOnce {
			failOnce = false
			w.WriteHeader(503)
			return
		}
		if seen[h.ID] {
			w.Write([]byte(`{"accepted":0,"duplicates":1,"skipped":{}}`))
		} else {
			seen[h.ID] = true
			w.Write([]byte(`{"accepted":1,"duplicates":0,"skipped":{}}`))
		}
	}))
	defer server.Close()
	setFeedbackTestDSN(t, server.URL)
	s := &Service{}
	if _, err := s.SendFeedback(context.Background(), p, wires); err == nil {
		t.Fatal("partial delivery reported success")
	}
	receipt, err := s.SendFeedback(context.Background(), p, wires)
	if err != nil || receipt.EventID != p.EventID || receipt.Duplicate {
		t.Fatalf("retry: %+v %v", receipt, err)
	}
	receipt, err = s.SendFeedback(context.Background(), p, wires)
	if err != nil || !receipt.Duplicate {
		t.Fatal("dedupe retry", err)
	}
	if calls != 6 || len(seen) != 2 {
		t.Fatal("unexpected retry count", calls, len(seen))
	}
}
func TestFeedbackHTTPFailuresAndReceipts(t *testing.T) {
	p, wires := feedbackFixture(t, false)
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{{"rate", 429, ""}, {"server", 503, ""}, {"unauthorized", 401, ""}, {"malformed", 200, "not JSON"}, {"missing", 200, "{}"}, {"skipped", 200, `{"accepted":1,"duplicates":0,"skipped":{"feedback":1}}`}, {"extra", 200, `{"accepted":1,"duplicates":0,"skipped":{}} {}`}, {"redirect", 302, ""}} {
		t.Run(tc.name, func(t *testing.T) {
			n := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n++
				w.Header().Set("Location", "http://127.0.0.1:1/private")
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			setFeedbackTestDSN(t, server.URL)
			if _, err := (&Service{}).SendFeedback(context.Background(), p, wires); err == nil {
				t.Fatal("accepted invalid response")
			}
			if tc.status == 429 {
				if _, err := (&Service{}).SendFeedback(context.Background(), p, wires); err == nil {
					t.Fatal("cooldown ignored")
				}
			}
			if n != 1 {
				t.Fatal("automatic retries or cooldown request", n)
			}
		})
	}
}
func TestFeedbackUnavailableDoesNotBlockLocalWork(t *testing.T) {
	p, wires := feedbackFixture(t, false)
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release }))
	defer server.Close()
	setFeedbackTestDSN(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	s := &Service{}
	go func() { _, err := s.SendFeedback(ctx, p, wires); done <- err }()
	<-entered
	start := time.Now()
	if _, err := s.PreviewFeedback(FeedbackInput{Message: "Independent local review"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SendFeedback(context.Background(), p, wires); err == nil {
		t.Fatal("allowed parallel report")
	}
	if time.Since(start) > time.Second {
		t.Fatal("local work blocked")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel success")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not terminate request")
	}
	close(release)
}
func TestFeedbackRefusedAndTimeout(t *testing.T) {
	p, wires := feedbackFixture(t, false)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	origin := "http://" + l.Addr().String()
	l.Close()
	setFeedbackTestDSN(t, origin)
	if _, err = (&Service{}).SendFeedback(context.Background(), p, wires); err == nil {
		t.Fatal("connection refused success")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, &net.DNSError{Err: "synthetic DNS failure", Name: "feedback.invalid", IsNotFound: true}
	}}
	if _, err = postFeedback(context.Background(), &http.Client{Transport: transport}, "https://feedback.invalid", strings.Repeat("a", 32), wires[0]); err == nil || strings.Contains(err.Error(), "feedback.invalid") {
		t.Fatal("DNS failure privacy", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(100 * time.Millisecond) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err = postFeedback(ctx, &http.Client{}, server.URL, strings.Repeat("a", 32), wires[0]); err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("timeout should be safe user message")
	}
}

// Opt-in acceptance only: generated by the actual pinned JS SDK with synthetic data.
// Ordinary tests never use the network or real project key.
func TestFeedbackSDKAcceptance(t *testing.T) {
	path := os.Getenv("SKILLVIEW_TEST_FEEDBACK_FIXTURE")
	if path == "" {
		t.Skip("explicit synthetic SDK fixture required")
	}
	var input struct {
		Submission FeedbackSubmission `json:"submission"`
		Envelopes  []string           `json:"envelopes"`
	}
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &input) != nil {
		t.Fatal("fixture", err)
	}
	input.Submission.Diagnostics.OS = runtime.GOOS
	input.Submission.Diagnostics.Arch = runtime.GOARCH
	if err := validateFeedback(input.Submission, input.Envelopes); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SKILLVIEW_TEST_FEEDBACK_PUBLIC") != "1" {
		return
	}
	for i := 0; i < 2; i++ {
		receipt, err := (&Service{}).SendFeedback(context.Background(), input.Submission, input.Envelopes)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 && !receipt.Duplicate {
			t.Fatal("public retry not deduplicated")
		}
	}
	t.Log("synthetic receipt and identical retry confirmed", input.Submission.EventID, input.Submission.ErrorEventID)
}
