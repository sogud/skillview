package network

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/mattn/go-ieproxy"
	"golang.org/x/net/http/httpproxy"
)

// Use the Mac's proxy without changing environment variables or system settings.
// Local model and acceptance servers always stay on this Mac.
func Transport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	config := httpproxy.FromEnvironment()
	if config.HTTPProxy == "" && config.HTTPSProxy == "" {
		system := ieproxy.GetConf()
		config.HTTPProxy = system.Static.Protocols["http"]
		config.HTTPSProxy = system.Static.Protocols["https"]
		if config.NoProxy == "" {
			config.NoProxy = system.Static.NoProxy
		}
	}
	// Read static settings only: PAC evaluation in the dependency is unbounded
	// and cannot be interrupted by the HTTP request's context.
	proxy := config.ProxyFunc()
	transport.Proxy = localBypass(func(request *http.Request) (*url.URL, error) { return proxy(request.URL) })
	return transport
}

func localBypass(proxy func(*http.Request) (*url.URL, error)) func(*http.Request) (*url.URL, error) {
	return func(request *http.Request) (*url.URL, error) {
		host := request.URL.Hostname()
		if host == "localhost" || net.ParseIP(host).IsLoopback() {
			return nil, nil
		}
		return proxy(request)
	}
}

// Messages deliberately omit URLs, credentials, and server response bodies.
const TimeoutMessage = "The connection timed out. Check your network and macOS proxy settings, then retry."
const ConnectionMessage = "Could not connect to the service. Check your network and macOS proxy settings, then retry."

func ErrorMessage(err error) string {
	var timeout net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return TimeoutMessage
	}
	return ConnectionMessage
}
