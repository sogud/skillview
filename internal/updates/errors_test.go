package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestUpdateRateLimitExplainsFailureWithoutResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(403)
		_, _ = w.Write([]byte("private-proxy-diagnostics"))
	}))
	defer server.Close()
	provider, _ := newProvider(server.Client(), server.URL, true)
	u := updater.New(testHost{})
	_ = u.Init(updater.Config{CurrentVersion: Version(), Providers: []updater.Provider{provider}, Window: updater.WindowNone})
	m := &Manager{updater: u, status: Status{Version: Version()}}
	status := m.Check(context.Background())
	if status.State != "error" || !strings.Contains(status.Message, "temporarily limited") || strings.Contains(status.Message, "private") || u.DownloadedPath() != "" {
		t.Fatal(status)
	}
}
