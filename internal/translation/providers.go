package translation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type httpStatusError struct{ status int }

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("translation service returned HTTP %d", e.status)
}

func providerKeyName(provider string) string {
	switch provider {
	case "google-cloud":
		return "google"
	case "ai", "libretranslate":
		return provider
	default:
		return ""
	}
}

const webUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15"

// Free web services have no published API quota. Avoid parallel bursts and
// immediate retries; each provider has its own cooldown so users can switch.
func (s *Service) webRequest(ctx context.Context, provider, method, endpoint, contentType string, body []byte, headers http.Header) ([]byte, error) {
	s.webMu.Lock()
	defer s.webMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if time.Now().Before(s.webBlocked[provider]) {
		return nil, fmt.Errorf("The free translation service is rate limited. Wait a minute or choose another provider.")
	}
	if delay := time.Until(s.webNext); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	client := s.client
	if provider == "youdao" {
		client = s.youdaoClient
	}
	data, err := s.requestUsing(ctx, client, method, endpoint, contentType, body, "", headers)
	s.webNext = time.Now().Add(300 * time.Millisecond)
	var status *httpStatusError
	if errors.As(err, &status) && status.status == http.StatusTooManyRequests {
		if s.webBlocked == nil {
			s.webBlocked = map[string]time.Time{}
		}
		s.webBlocked[provider] = time.Now().Add(time.Minute)
	}
	return data, err
}

func (s *Service) googleRequest(ctx context.Context, endpoint string) ([]byte, error) {
	return s.webRequest(ctx, "google", http.MethodGet, endpoint, "", nil, http.Header{"User-Agent": {webUserAgent}})
}

// Keep newline boundaries while combining small Markdown spans into one web
// request. A response must retain every line before it can replace any spans.
func translateWebBatches(texts []string, translate func(string) (string, error), done func(int)) ([]string, error) {
	output := []string{}
	for start := 0; start < len(texts); {
		end := start + 1
		size := len([]rune(texts[start]))
		if !strings.ContainsAny(texts[start], "\r\n") {
			for end < len(texts) && end-start < 24 {
				if strings.ContainsAny(texts[end], "\r\n") {
					break
				}
				length := len([]rune(texts[end])) + 1
				if size+length > 3000 {
					break
				}
				size += length
				end++
			}
		}
		value, err := translate(strings.Join(texts[start:end], "\n"))
		if err != nil {
			return nil, err
		}
		batch := []string{value}
		if end-start > 1 {
			value = strings.ReplaceAll(value, "\r\n", "\n")
			batch = strings.Split(strings.TrimSpace(value), "\n")
		}
		if err := validateTranslations(batch, end-start); err != nil {
			return nil, err
		}
		output = append(output, batch...)
		done(end - start)
		start = end
	}
	return output, nil
}

func (s *Service) libreTranslate(ctx context.Context, settings Settings, key string, texts []string, done func(int)) ([]string, error) {
	u, err := validateBaseURL(settings.LibreURL)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/translate"
	target := settings.TargetLanguage
	switch target {
	case "zh-CN":
		target = "zh"
	case "zh-TW":
		target = "zt"
	}
	output := []string{}
	for start := 0; start < len(texts); start += 24 {
		end := start + 24
		if end > len(texts) {
			end = len(texts)
		}
		request := map[string]any{"q": texts[start:end], "source": "auto", "target": target, "format": "text"}
		if key != "" {
			request["api_key"] = key
		}
		body, _ := json.Marshal(request)
		data, err := s.request(ctx, http.MethodPost, u.String(), "application/json", body, "")
		if err != nil {
			return nil, err
		}
		var response struct {
			Texts []string `json:"translatedText"`
		}
		if json.Unmarshal(data, &response) != nil {
			return nil, fmt.Errorf("LibreTranslate returned an unreadable response")
		}
		if err := validateTranslations(response.Texts, end-start); err != nil {
			return nil, err
		}
		output = append(output, response.Texts...)
		done(end - start)
	}
	return output, nil
}
