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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"skillview/internal/translation"
	"skillview/internal/updates"
)

// Only user-requested submissions use this transport. It never reads documents,
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
	Enabled     bool                            `json:"enabled"`
	Destination string                          `json:"destination"`
	AIError     *translation.ResponseDiagnostic `json:"aiError,omitempty"`
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

func feedbackDestination() (string, error) {
	raw := strings.TrimSpace(os.Getenv("SKILLVIEW_FEEDBACK_ENDPOINT"))
	if raw == "" || os.Getenv("SKILLVIEW_FEEDBACK_PROJECT_KEY") == "" {
		return "", fmt.Errorf("Feedback is not configured in this build.")
	}
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
		return "", fmt.Errorf("Feedback configuration is invalid.")
	}
	loopback := endpoint.Hostname() == "127.0.0.1" || endpoint.Hostname() == "::1" || endpoint.Hostname() == "localhost"
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && loopback) {
		return "", fmt.Errorf("Feedback configuration is invalid.")
	}
	return strings.TrimRight(endpoint.String(), "/"), nil
}

func (s *Service) FeedbackConfiguration() FeedbackConfiguration {
	destination, err := feedbackDestination()
	s.mu.Lock()
	defer s.mu.Unlock()
	return FeedbackConfiguration{Enabled: err == nil, Destination: destination, AIError: s.translationDiagnostic}
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
		result.Kind = "error"
		result.Name = "ai_translation_error"
		result.ErrorCode = "SV_AI_RESPONSE_FORMAT"
		result.Diagnostics.AIResponse = diagnostic
	}
	return result, nil
}

// SendFeedback sends exactly the reviewed payload once. The caller retains it
// across retries, including event_id and timestamp. No background queue is used.
func (s *Service) SendFeedback(ctx context.Context, submission FeedbackSubmission) (FeedbackReceipt, error) {
	destination, err := feedbackDestination()
	if err != nil {
		return FeedbackReceipt{}, err
	}
	validKind := submission.Kind == "feedback" && submission.Name == "user_feedback" && submission.ErrorCode == "" && submission.Diagnostics.AIResponse == nil
	validKind = validKind || (submission.Kind == "error" && submission.Name == "ai_translation_error" && submission.ErrorCode == "SV_AI_RESPONSE_FORMAT" && submission.Diagnostics.AIResponse != nil && submission.Diagnostics.AIResponse.Valid())
	_, timestampErr := time.Parse(time.RFC3339, submission.Timestamp)
	if !validKind || !feedbackID.MatchString(submission.EventID) || submission.SchemaVersion != 1 || submission.ProjectID != "skillview" ||
		timestampErr != nil || submission.Release != updates.Version() || submission.Message == "" || len(submission.Message) > 8000 ||
		redactFeedback(submission.Message) != submission.Message || submission.Diagnostics.OS != runtime.GOOS || submission.Diagnostics.Arch != runtime.GOARCH {
		return FeedbackReceipt{}, fmt.Errorf("Feedback changed. Review it again before sending.")
	}
	body, err := json.Marshal(submission)
	if err != nil {
		return FeedbackReceipt{}, fmt.Errorf("Could not prepare feedback.")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, destination+"/api/v1/ingest", bytes.NewReader(body))
	if err != nil {
		return FeedbackReceipt{}, fmt.Errorf("Feedback configuration is invalid.")
	}
	request.Header.Set("Content-Type", "application/json")
	// This is a public, write-only project key, never an administrator credential.
	request.Header.Set("X-Project-Key", os.Getenv("SKILLVIEW_FEEDBACK_PROJECT_KEY"))
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return FeedbackReceipt{}, fmt.Errorf("Delivery could not be confirmed. Retry keeps the same report ID.")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		seconds, _ := strconv.Atoi(response.Header.Get("Retry-After"))
		if seconds > 0 && seconds <= 3600 {
			return FeedbackReceipt{}, fmt.Errorf("Too many reports. Retry after %d seconds.", seconds)
		}
		return FeedbackReceipt{}, fmt.Errorf("Too many reports. Try again later.")
	}
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return FeedbackReceipt{}, fmt.Errorf("Feedback was not accepted (HTTP %d).", response.StatusCode)
	}
	var receipt FeedbackReceipt
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4096))
	if err := decoder.Decode(&receipt); err != nil || receipt.EventID != submission.EventID {
		return FeedbackReceipt{}, fmt.Errorf("Delivery could not be confirmed. Retry keeps the same report ID.")
	}
	return receipt, nil
}
