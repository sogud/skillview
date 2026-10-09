package updates

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"skillview/internal/network"
	"strings"
)

const rateLimitMessage = "GitHub has temporarily limited update checks. Wait and retry, or open GitHub Releases."
const unavailableMessage = "The update service is unavailable. Retry later or open GitHub Releases."
const invalidReleaseMessage = "The release information could not be read or verified. Open GitHub Releases or retry later."

type responseError struct {
	status      int
	rateLimited bool
}

func (e *responseError) Error() string {
	return fmt.Sprintf("update service returned HTTP %d", e.status)
}

type statusTransport struct{ http.RoundTripper }

func (t statusTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.RoundTripper.RoundTrip(request)
	if err != nil || response.StatusCode < 400 || response.StatusCode == http.StatusNotFound {
		return response, err
	}
	response.Body.Close()
	return nil, &responseError{status: response.StatusCode,
		rateLimited: response.StatusCode == 429 || (response.StatusCode == 403 && response.Header.Get("X-RateLimit-Remaining") == "0")}
}

func checkErrorMessage(err error) string {
	// Wails beta.27 flattens its provider error chain into a string. Providers
	// sanitize before returning, and the manager restores that known message.
	for _, message := range []string{rateLimitMessage, unavailableMessage, network.TimeoutMessage, network.ConnectionMessage} {
		if strings.Contains(err.Error(), message) {
			return message
		}
	}
	var response *responseError
	if errors.As(err, &response) {
		if response.rateLimited {
			return rateLimitMessage
		}
		return unavailableMessage
	}
	var request *url.Error
	if errors.As(err, &request) {
		return network.ErrorMessage(err)
	}
	return invalidReleaseMessage
}
