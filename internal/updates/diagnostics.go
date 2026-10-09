package updates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type diagnosticLog struct {
	mu   sync.Mutex
	path string
}

func newDiagnosticLog(dataDir string) (*diagnosticLog, error) {
	dir := filepath.Join(dataDir, "diagnostics")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &diagnosticLog{path: filepath.Join(dir, "updates.log")}, nil
}

// Callers supply only fixed events, versions, byte counts and classified errors.
// No URLs, settings, paths, credentials or Skill text enter this log.
func (l *diagnosticLog) write(event, detail string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if info, err := os.Stat(l.path); err == nil && info.Size() > 512*1024 {
		_ = os.Rename(l.path, l.path+".previous")
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s %s %s\n", time.Now().UTC().Format(time.RFC3339), event, detail)
}

func diagnosticReason(err error) string {
	if err == nil {
		return "none"
	}
	var response *responseError
	if errors.As(err, &response) {
		return fmt.Sprintf("http_%d", response.status)
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	var timeout net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return "timeout"
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "connection_closed"
	}
	if errors.Is(err, os.ErrPermission) {
		return "permission_denied"
	}
	text := strings.ToLower(err.Error())
	for _, reason := range []struct{ pattern, code string }{
		{"signature", "signature_invalid"}, {"digest", "digest_invalid"},
		{"checksum", "digest_invalid"}, {"codesign", "bundle_signature_invalid"},
		{"timed out", "timeout"}, {"timeout", "timeout"},
		{"permission denied", "permission_denied"}, {"helper", "helper_failed"},
		{"resume", "resume_unavailable"}, {"interrupted", "connection_closed"},
		{"connection", "connection_failed"}, {"zip", "archive_invalid"},
	} {
		if strings.Contains(text, reason.pattern) {
			return reason.code
		}
	}
	return "update_failed"
}

func (m *Manager) Diagnostics() (string, error) {
	if m.log == nil {
		return "No update diagnostics are available.", nil
	}
	m.collectHelperLog()
	m.log.mu.Lock()
	defer m.log.mu.Unlock()
	var report strings.Builder
	for _, path := range []string{m.log.path + ".previous", m.log.path} {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", errors.New("The update log could not be read.")
		}
		report.Write(data)
	}
	return report.String(), nil
}

func (m *Manager) LogPath() string {
	if m.log == nil {
		return ""
	}
	return m.log.path
}

func (m *Manager) collectHelperLog() {
	if m.log == nil {
		return
	}
	pending := filepath.Join(filepath.Dir(m.log.path), "pending-helper")
	data, err := os.ReadFile(pending)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil || pid <= 0 {
		return
	}
	data, err = os.ReadFile(filepath.Join(os.TempDir(), fmt.Sprintf("wails-update-%d.log", pid)))
	if err != nil {
		return
	}
	// The SDK helper includes local paths. Import only its fixed stage names.
	for _, line := range strings.Split(string(data), "\n") {
		for _, stage := range []string{"helper start:", "parent did not exit", "backing up", "backup failed:", "swap succeeded", "all swap attempts exhausted", "launch new failed:", "restore failed:", "helper done"} {
			if strings.Contains(line, stage) {
				m.log.write("install_helper", strings.TrimSuffix(stage, ":"))
				break
			}
		}
	}
	if strings.Contains(string(data), "helper done") || strings.Contains(string(data), "failed:") || strings.Contains(string(data), "aborting swap") {
		_ = os.Remove(pending)
	}
}
