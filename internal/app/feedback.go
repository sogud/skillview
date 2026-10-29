package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"skillview/internal/translation"
	"skillview/internal/updates"
)

// This restricted transport never reads documents,
// translation output, settings, credentials, or log files.
type FeedbackInput struct {
	Message        string `json:"message"`
	IncludeAIError bool   `json:"includeAIError"`
}

type FeedbackDiagnostics struct {
	OS         string                          `json:"os"`
	Arch       string                          `json:"arch"`
	AIResponse *translation.ResponseDiagnostic `json:"ai_response,omitempty"`
}

type FeedbackSubmission struct {
	SchemaVersion int                 `json:"schema_version"`
	ProjectID     string              `json:"project_id"`
	EventID       string              `json:"event_id"`
	ErrorEventID  string              `json:"error_event_id,omitempty"`
	Kind          string              `json:"kind"`
	Name          string              `json:"name"`
	Timestamp     string              `json:"timestamp"`
	Release       string              `json:"release"`
	Message       string              `json:"message"`
	ErrorCode     string              `json:"error_code,omitempty"`
	Diagnostics   FeedbackDiagnostics `json:"diagnostics"`
}

type FeedbackReceipt struct {
	EventID   string `json:"event_id"`
	Duplicate bool   `json:"duplicate"`
}

type FeedbackConfiguration struct {
	AutomaticErrors bool                            `json:"automaticErrors"`
	Release         string                          `json:"release"`
	Environment     string                          `json:"environment"`
	Enabled         bool                            `json:"enabled"`
	Destination     string                          `json:"destination"`
	DSN             string                          `json:"dsn"`
	AIError         *translation.ResponseDiagnostic `json:"aiError,omitempty"`
}

var feedbackRedactions = []*regexp.Regexp{
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`),
	regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9+/_.=:-]+`),
	regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{8,}|gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|xox[baprs]-[A-Za-z0-9-]+)\b`),
	regexp.MustCompile(`(?i)["']?(?:api[_-]?key|access[_-]?token|refresh[_-]?token|token|password|passwd|secret|authorization|aws_access_key_id|aws_secret_access_key)["']?\s*[=:]\s*(?:"[^"\r\n]*"|'[^'\r\n]*'|[^\s,;]+)`),
	regexp.MustCompile(`(?i)\b(?:https?|file)://[^\s<>"']+`),
	regexp.MustCompile(`(?:/(?:Users|home|Volumes|private|var|tmp)/|[A-Za-z]:\\)[^\s<>"']+`),
	regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
}
var feedbackID = regexp.MustCompile(`^[a-f0-9]{32}$`)

func redactFeedback(text string) string {
	for _, rule := range feedbackRedactions {
		text = rule.ReplaceAllString(text, "[redacted]")
	}
	return strings.TrimSpace(text)
}

// Public ingestion-only DSN. No administrator credential is compiled into the app.
// SKILLVIEW_FEEDBACK_DSN is an explicit development/deployment override; empty disables.
func feedbackDSN() (string, string, string, error) {
	raw, present := os.LookupEnv("SKILLVIEW_FEEDBACK_DSN")
	if !present {
		raw = defaultFeedbackDSN
	}
	if raw == "" {
		return "", "", "", fmt.Errorf("Feedback is not configured in this build.")
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.Path != "/1" {
		return "", "", "", fmt.Errorf("Feedback configuration is invalid.")
	}
	_, password := u.User.Password()
	key := u.User.Username()
	loopback := u.Hostname() == "127.0.0.1" || u.Hostname() == "::1" || u.Hostname() == "localhost"
	if password || !feedbackID.MatchString(key) || (u.Scheme != "https" && !(u.Scheme == "http" && loopback)) {
		return "", "", "", fmt.Errorf("Feedback configuration is invalid.")
	}
	return raw, u.Scheme + "://" + u.Host, key, nil
}

func (s *Service) FeedbackConfiguration() FeedbackConfiguration {
	dsn, destination, _, err := feedbackDSN()
	s.mu.Lock()
	defer s.mu.Unlock()
	return FeedbackConfiguration{Enabled: err == nil, Destination: destination, DSN: dsn, AIError: s.translationDiagnostic, AutomaticErrors: s.library.Settings().AutomaticErrors, Release: updates.Version(), Environment: automaticErrorEnvironment()}
}

// PreviewFeedback is entirely local. Do not add network requests here.
func (s *Service) PreviewFeedback(input FeedbackInput) (FeedbackSubmission, error) {
	if len(input.Message) > 8000 {
		return FeedbackSubmission{}, fmt.Errorf("Feedback is too long or invalid.")
	}
	message := redactFeedback(input.Message)
	if message == "" {
		return FeedbackSubmission{}, fmt.Errorf("Describe the problem before sending.")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return FeedbackSubmission{}, fmt.Errorf("Could not prepare feedback.")
	}
	result := FeedbackSubmission{
		SchemaVersion: 1, ProjectID: "skillview", EventID: hex.EncodeToString(id[:]),
		Kind: "feedback", Name: "user_feedback", Timestamp: time.Now().UTC().Format(time.RFC3339),
		Release: updates.Version(), Message: message,
		Diagnostics: FeedbackDiagnostics{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}
	if input.IncludeAIError {
		s.mu.Lock()
		diagnostic := s.translationDiagnostic
		s.mu.Unlock()
		if diagnostic == nil {
			return FeedbackSubmission{}, fmt.Errorf("No AI response diagnostic is available. Review a general report instead.")
		}
		if _, err := rand.Read(id[:]); err != nil {
			return FeedbackSubmission{}, fmt.Errorf("Could not prepare feedback.")
		}
		result.ErrorEventID = hex.EncodeToString(id[:])
		result.Kind = "error"
		result.Name = "ai_translation_error"
		result.ErrorCode = "SV_AI_RESPONSE_FORMAT"
		result.Diagnostics.AIResponse = diagnostic
	}
	return result, nil
}

// Expected payloads are also the native privacy boundary: SDK-added user data,
// URLs, breadcrumbs, stacks, sessions, attachments and trace metadata are rejected.
func feedbackEvents(p FeedbackSubmission) []map[string]any {
	stamp, _ := time.Parse(time.RFC3339, p.Timestamp)
	common := func(id string) map[string]any {
		return map[string]any{
			"event_id": id, "timestamp": stamp.Unix(), "release": p.Release,
			"environment": "production", "platform": "javascript",
		}
	}
	var events []map[string]any
	if d := p.Diagnostics.AIResponse; d != nil {
		e := common(p.ErrorEventID)
		e["level"] = "error"
		e["exception"] = map[string]any{"values": []any{map[string]any{
			"type":  "SV_AI_RESPONSE_FORMAT",
			"value": fmt.Sprintf("category=%s; finish_reason=%s; truncation=%s", d.Category, d.FinishReason, d.Truncation),
		}}}
		events = append(events, e)
	}
	e := common(p.EventID)
	e["type"], e["level"] = "feedback", "info"
	f := map[string]any{"message": p.Message + "\n\nOS: " + p.Diagnostics.OS + "; Arch: " + p.Diagnostics.Arch}
	if p.ErrorEventID != "" {
		f["associated_event_id"] = p.ErrorEventID
	}
	e["contexts"] = map[string]any{"feedback": f}
	return append(events, e)
}

func validateFeedback(p FeedbackSubmission, envelopes []string) error {
	valid := p.Kind == "feedback" && p.Name == "user_feedback" && p.ErrorCode == "" && p.Diagnostics.AIResponse == nil && p.ErrorEventID == ""
	valid = valid || (p.Kind == "error" && p.Name == "ai_translation_error" && p.ErrorCode == "SV_AI_RESPONSE_FORMAT" && p.Diagnostics.AIResponse != nil && p.Diagnostics.AIResponse.Valid() && feedbackID.MatchString(p.ErrorEventID) && p.ErrorEventID != p.EventID)
	_, timestampErr := time.Parse(time.RFC3339, p.Timestamp)
	invalid := func() error { return fmt.Errorf("Feedback changed. Review it again before sending.") }
	if !valid || !feedbackID.MatchString(p.EventID) || p.SchemaVersion != 1 || p.ProjectID != "skillview" || timestampErr != nil || p.Release != updates.Version() || p.Message == "" || len(p.Message) > 8000 || redactFeedback(p.Message) != p.Message || p.Diagnostics.OS != runtime.GOOS || p.Diagnostics.Arch != runtime.GOARCH {
		return invalid()
	}
	expected := feedbackEvents(p)
	if len(envelopes) != len(expected) {
		return invalid()
	}
	for i, wire := range envelopes {
		if !matchesFeedbackEnvelope(wire, expected[i], p.Timestamp) {
			return invalid()
		}
	}
	return nil
}

func matchesFeedbackEnvelope(wire string, expected map[string]any, timestamp string) bool {
	if len(wire) > 32768 {
		return false
	}
	lines := strings.Split(wire, "\n")
	if len(lines) != 3 {
		return false
	}
	kind := "event"
	if expected["type"] == "feedback" {
		kind = "feedback"
	}
	want := []any{map[string]any{"event_id": expected["event_id"], "sent_at": timestamp}, map[string]any{"type": kind}, expected}
	for j := range lines {
		var got, canonical any
		b, _ := json.Marshal(want[j])
		_ = json.Unmarshal(b, &canonical)
		if json.Unmarshal([]byte(lines[j]), &got) != nil || !reflect.DeepEqual(got, canonical) {
			return false
		}
	}
	return true
}

var feedbackSendMu sync.Mutex
var feedbackRetryAfter time.Time // guarded by feedbackSendMu; no timer or background retry
const feedbackTimeout = 6 * time.Second

// SendFeedback accepts only SDK envelopes for the locally reviewed report.
// One report in flight, zero waiting queue/disk spool, zero automatic retries.
// The caller retains stable IDs for explicit retries, including partial delivery.
func (s *Service) SendFeedback(ctx context.Context, submission FeedbackSubmission, envelopes []string) (FeedbackReceipt, error) {
	_, destination, key, err := feedbackDSN()
	if err != nil {
		return FeedbackReceipt{}, err
	}
	if err := validateFeedback(submission, envelopes); err != nil {
		return FeedbackReceipt{}, err
	}
	if !feedbackSendMu.TryLock() {
		return FeedbackReceipt{}, fmt.Errorf("Too many reports. Try again later.")
	}
	defer feedbackSendMu.Unlock()
	if time.Now().Before(feedbackRetryAfter) {
		return FeedbackReceipt{}, fmt.Errorf("Too many reports. Try again later.")
	}
	duplicate, err := deliverFeedback(ctx, destination, key, envelopes, feedbackTimeout)
	if err != nil {
		return FeedbackReceipt{}, err
	}
	return FeedbackReceipt{EventID: submission.EventID, Duplicate: duplicate}, nil
}

func deliverFeedback(ctx context.Context, destination, key string, envelopes []string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, MaxConnsPerHost: 1, MaxIdleConns: 1, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	duplicate := true
	for _, body := range envelopes {
		wasDuplicate, err := postFeedback(ctx, client, destination, key, body)
		if err != nil {
			return false, err
		}
		duplicate = duplicate && wasDuplicate
	}
	return duplicate, nil
}

func postFeedback(ctx context.Context, client *http.Client, destination, key, body string) (bool, error) {
	unknown := func() error { return fmt.Errorf("Delivery could not be confirmed. Retry keeps the same report ID.") }
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, destination+"/api/1/envelope/", strings.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("Feedback configuration is invalid.")
	}
	request.Header.Set("Content-Type", "application/x-sentry-envelope")
	request.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7,sentry_key="+key+",sentry_client=skillview/"+updates.Version())
	response, err := client.Do(request)
	if err != nil {
		return false, unknown()
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		seconds, _ := strconv.Atoi(response.Header.Get("Retry-After"))
		if seconds < 1 || seconds > 3600 {
			seconds = 60
		}
		feedbackRetryAfter = time.Now().Add(time.Duration(seconds) * time.Second)
		return false, fmt.Errorf("Too many reports. Retry after %d seconds.", seconds)
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Feedback was not accepted (HTTP %d).", response.StatusCode)
	}
	var receipt struct {
		Accepted   int            `json:"accepted"`
		Duplicates int            `json:"duplicates"`
		Skipped    map[string]int `json:"skipped"`
	}
	b, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil || len(b) > 4096 {
		return false, unknown()
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&receipt) != nil || decoder.Decode(new(any)) != io.EOF || receipt.Accepted < 0 || receipt.Duplicates < 0 || receipt.Accepted+receipt.Duplicates != 1 || len(receipt.Skipped) != 0 {
		return false, unknown()
	}
	return receipt.Duplicates == 1, nil
}
