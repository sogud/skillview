package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"skillview/internal/updates"
)

// No error messages, function names, source text, URLs or filesystem paths.
type AutomaticErrorFrame struct {
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
	Colno    int    `json:"colno"`
}
type AutomaticErrorReport struct {
	EventID     string                `json:"event_id"`
	Timestamp   string                `json:"timestamp"`
	Release     string                `json:"release"`
	Environment string                `json:"environment"`
	Source      string                `json:"source"`
	ErrorType   string                `json:"error_type"`
	Frames      []AutomaticErrorFrame `json:"frames"`
}
type automaticErrorState struct {
	mu     sync.Mutex
	seen   map[[32]byte]bool
	last   time.Time
	cancel context.CancelFunc
}

func automaticErrorEnvironment() string {
	if os.Getenv("SKILLVIEW_SOURCE") != "" {
		return "synthetic"
	}
	return "production"
}

func automaticErrorDestination() (string, string, error) {
	_, destination, key, err := feedbackDSN()
	if err != nil {
		return "", "", err
	}
	u, _ := url.Parse(destination)
	loopback := u.Hostname() == "127.0.0.1" || u.Hostname() == "::1" || u.Hostname() == "localhost"
	if destination != "https://feedback.sogud.me" && !(automaticErrorEnvironment() == "synthetic" && loopback) {
		return "", "", fmt.Errorf("Feedback configuration is invalid.")
	}
	return destination, key, nil
}

func (s *Service) SetAutomaticErrors(enabled bool) error {
	state := &s.automaticErrors
	state.mu.Lock()
	defer state.mu.Unlock()
	if enabled {
		if _, _, err := automaticErrorDestination(); err != nil {
			return err
		}
	}
	if err := s.library.SetAutomaticErrors(enabled); err != nil {
		return err
	}
	if !enabled && state.cancel != nil {
		state.cancel()
	}
	return nil
}

func automaticErrorEvent(p AutomaticErrorReport) (map[string]any, [32]byte, error) {
	invalid := func() (map[string]any, [32]byte, error) {
		return nil, [32]byte{}, fmt.Errorf("Could not prepare feedback.")
	}
	stamp, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil || !feedbackID.MatchString(p.EventID) || p.Release != updates.Version() || p.Environment != automaticErrorEnvironment() || len(p.Frames) > 8 {
		return invalid()
	}
	if p.Source != "uncaught_exception" && p.Source != "unhandled_rejection" {
		return invalid()
	}
	switch p.ErrorType {
	case "Error", "TypeError", "RangeError", "ReferenceError", "SyntaxError", "URIError", "EvalError", "AggregateError", "NonErrorRejection":
	default:
		return invalid()
	}
	frames := make([]map[string]any, 0, len(p.Frames))
	for _, f := range p.Frames {
		if f.Filename != "frontend.js" || f.Lineno < 1 || f.Lineno > 1000000 || f.Colno < 1 || f.Colno > 10000000 {
			return invalid()
		}
		frames = append(frames, map[string]any{"filename": f.Filename, "lineno": f.Lineno, "colno": f.Colno, "in_app": true})
	}
	identity, _ := json.Marshal([]any{p.Source, p.ErrorType, p.Frames})
	fingerprint := sha256.Sum256(identity)
	exception := map[string]any{"type": p.ErrorType, "value": "Automatic error report; message omitted."}
	if len(frames) > 0 {
		exception["stacktrace"] = map[string]any{"frames": frames}
	}
	tags := map[string]string{"source": "automatic_error", "error_origin": p.Source}
	if p.Environment == "synthetic" {
		tags["synthetic"] = "true"
	}
	group := []string{"skillview-auto", p.Source, p.ErrorType}
	for _, f := range p.Frames {
		group = append(group, strconv.Itoa(f.Lineno)+":"+strconv.Itoa(f.Colno))
	}
	event := map[string]any{
		"event_id": p.EventID, "timestamp": stamp.Unix(), "release": p.Release,
		"environment": p.Environment, "platform": "javascript", "level": "error",
		"exception": map[string]any{"values": []any{exception}},
		"tags":      tags, "fingerprint": group,
	}
	return event, fingerprint, nil
}

// Consent and limits are rechecked natively. One shared send in flight, no queue,
// at most one attempt/minute and ten distinct errors per process; never retry.
func (s *Service) SendAutomaticError(ctx context.Context, report AutomaticErrorReport, envelope string) error {
	destination, key, err := automaticErrorDestination()
	if err != nil {
		return err
	}
	expected, fingerprint, err := automaticErrorEvent(report)
	if err != nil {
		return err
	}
	if !matchesFeedbackEnvelope(envelope, expected, report.Timestamp) {
		return fmt.Errorf("Could not prepare feedback.")
	}
	state := &s.automaticErrors
	state.mu.Lock()
	if !s.library.Settings().AutomaticErrors {
		state.mu.Unlock()
		return fmt.Errorf("Automatic error reports are disabled.")
	}
	if state.seen[fingerprint] || len(state.seen) >= 10 || time.Since(state.last) < time.Minute || !feedbackSendMu.TryLock() {
		state.mu.Unlock()
		return fmt.Errorf("Too many reports. Try again later.")
	}
	if time.Now().Before(feedbackRetryAfter) {
		feedbackSendMu.Unlock()
		state.mu.Unlock()
		return fmt.Errorf("Too many reports. Try again later.")
	}
	if state.seen == nil {
		state.seen = make(map[[32]byte]bool)
	}
	state.seen[fingerprint] = true
	state.last = time.Now()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	state.cancel = cancel
	state.mu.Unlock()
	defer feedbackSendMu.Unlock()
	defer func() {
		cancel()
		state.mu.Lock()
		state.cancel = nil
		state.mu.Unlock()
	}()
	_, err = deliverFeedback(ctx, destination, key, []string{envelope}, 3*time.Second)
	return err
}
