package updates

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticsPersistClassifiedErrorsWithoutPrivateData(t *testing.T) {
	dir := t.TempDir()
	log, err := newDiagnosticLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	failure := &url.Error{Op: "Get", URL: "https://secret:password@example.test/private?token=hidden", Err: context.DeadlineExceeded}
	log.write("download_failed", diagnosticReason(failure))
	log.write("bundle_failed", diagnosticReason(errors.New("codesign failed at /Users/private/person/app")))
	log, err = newDiagnosticLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{log: log}
	report, err := m.Diagnostics()
	if err != nil || !strings.Contains(report, "timeout") || !strings.Contains(report, "bundle_signature_invalid") {
		t.Fatal(report, err)
	}
	for _, secret := range []string{"secret", "password", "example.test", "token", "/Users/"} {
		if strings.Contains(report, secret) {
			t.Fatal("private data logged")
		}
	}
	info, _ := os.Stat(log.path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("log permissions", info.Mode())
	}
	if err := os.WriteFile(log.path, []byte(strings.Repeat("x", 512*1024+1)), 0600); err != nil {
		t.Fatal(err)
	}
	log.write("app_started", "version=fixture")
	if _, err := os.Stat(log.path + ".previous"); err != nil {
		t.Fatal("rotation failed", err)
	}
	if filepath.Dir(log.path) != filepath.Join(dir, "diagnostics") {
		t.Fatal("log not isolated")
	}
}
