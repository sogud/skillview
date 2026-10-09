package translation

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

func newHTTPClient() *http.Client {
	client := retryablehttp.NewClient()
	client.Logger = nil
	client.RetryMax = 2
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	client.HTTPClient.Timeout = 60 * time.Second
	client.HTTPClient.CheckRedirect = refuseRedirect
	client.CheckRetry = func(ctx context.Context, response *http.Response, err error) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		// A timed-out model request may already have been billed. Retry only an
		// explicit rate-limit response with a short, valid server-provided delay.
		if err != nil || response == nil || response.StatusCode != http.StatusTooManyRequests {
			return false, nil
		}
		header := response.Header.Get("Retry-After")
		if seconds, err := strconv.ParseInt(header, 10, 64); err == nil && (seconds < 0 || seconds > 10) {
			return false, nil
		}
		// The library parses both seconds and HTTP dates. A negative fallback
		// distinguishes missing/invalid hints from a valid zero-second delay.
		delay := retryablehttp.DefaultBackoff(-1, -1, 0, response)
		return delay >= 0 && delay <= 10*time.Second, nil
	}
	standard := client.StandardClient()
	standard.Timeout = 60 * time.Second
	standard.CheckRedirect = refuseRedirect
	return standard
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
