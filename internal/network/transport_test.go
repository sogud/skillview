package network

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestConfiguredProxyRoutesRemoteRequestsAndBypassesLocalModels(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:34567")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:34567")
	t.Setenv("NO_PROXY", "")
	transport := Transport()
	for _, raw := range []string{"http://localhost:11434/v1", "http://127.0.0.1:1234/v1", "http://[::1]:1234/v1", "https://api.github.com"} {
		request, _ := http.NewRequest(http.MethodGet, raw, nil)
		proxy, err := transport.Proxy(request)
		if err != nil {
			t.Fatal(err)
		}
		if request.URL.Hostname() == "api.github.com" {
			if proxy == nil || proxy.String() != "http://127.0.0.1:34567" {
				t.Fatal("remote request ignored the configured proxy", proxy)
			}
		} else if proxy != nil {
			t.Fatal("local model was sent through a proxy", proxy)
		}
	}
}

func TestNetworkErrorMessageDoesNotExposePrivateURLs(t *testing.T) {
	err := &url.Error{Op: "Post", URL: "https://secret:private@example.invalid?key=private", Err: context.DeadlineExceeded}
	if ErrorMessage(err) != "The connection timed out. Check your network and macOS proxy settings, then retry." {
		t.Fatal(ErrorMessage(err))
	}
}
