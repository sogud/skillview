package updates

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"skillview/internal/network"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

//go:embed version.txt
var versionText string

//go:embed updater.key.pub
var publicKey []byte

func Version() string { return strings.TrimSpace(versionText) }

type Status struct {
	Version    string `json:"version"`
	State      string `json:"state"`
	Latest     string `json:"latest"`
	Notes      string `json:"notes"`
	ReleaseURL string `json:"releaseURL"`
	CanInstall bool   `json:"canInstall"`
	Message    string `json:"message"`
	Written    int64  `json:"written"`
	Total      int64  `json:"total"`
}

type Manager struct {
	updater    *updater.Updater
	mu         sync.Mutex
	op         sync.Mutex
	status     Status
	notify     func(Status)
	fixture    bool
	cancel     context.CancelFunc
	done       chan struct{}
	log        *diagnosticLog
	logPercent int64
}

func New(u *updater.Updater, fixture bool, notify func(Status), dataDir ...string) (*Manager, error) {
	baseURL := ""
	// Native acceptance may use an isolated loopback release fixture. Ordinary
	// launches always use GitHub; environment variables cannot change its feed.
	if fixture {
		if raw := os.Getenv("SKILLVIEW_UPDATE_API"); raw != "" {
			parsed, err := url.Parse(raw)
			if err != nil || parsed.Scheme != "http" || parsed.User != nil || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1") {
				return nil, errors.New("Update fixtures must use a loopback HTTP address.")
			}
			baseURL = raw
		}
	}
	provider, err := newProvider(&http.Client{Timeout: 30 * time.Second, Transport: network.Transport()}, baseURL, strings.Contains(Version(), "-"))
	if err != nil {
		return nil, err
	}
	if err := u.Init(updater.Config{CurrentVersion: Version(), Providers: []updater.Provider{provider}, PublicKey: publicKey, Window: updater.WindowNone}); err != nil {
		return nil, err
	}
	m := &Manager{updater: u, fixture: fixture, notify: notify, status: Status{Version: Version(), State: "idle"}}
	if len(dataDir) > 0 {
		m.log, err = newDiagnosticLog(dataDir[0])
		if err != nil {
			return nil, errors.New("The update log folder could not be created.")
		}
		provider.log = m.log.write
		m.collectHelperLog()
		m.log.write("app_started", "version="+Version())
	}
	return m, nil
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *Manager) publish(status Status) Status {
	m.mu.Lock()
	m.status = status
	m.mu.Unlock()
	if m.notify != nil {
		m.notify(status)
	}
	return status
}

func (m *Manager) Check(ctx context.Context) Status {
	if !m.op.TryLock() {
		return m.Status()
	}
	defer m.op.Unlock()
	status := m.Status()
	if status.State == "ready" {
		return status
	}
	status.State, status.Message = "checking", ""
	m.log.write("check_started", "version="+Version())
	m.publish(status)
	release, err := m.updater.Check(ctx)
	if err != nil {
		m.log.write("check_failed", diagnosticReason(err))
		status = Status{Version: Version(), State: "error", Message: checkErrorMessage(err)}
	} else if release == nil {
		status = Status{Version: Version(), State: "up-to-date"}
	} else {
		status = Status{Version: Version(), State: "available", Latest: release.Version, Notes: release.Notes}
		status.ReleaseURL, _ = release.Metadata["github.release.htmlURL"].(string)
		status.Message = installReason(m.fixture)
		if !hasSignature(release) {
			status.Message = "This release supports manual installation only."
		}
		status.CanInstall = status.Message == ""
		m.log.write("check_available", fmt.Sprintf("version=%s can_install=%t", status.Latest, status.CanInstall))
	}
	return m.publish(status)
}

func (m *Manager) Download(ctx context.Context) Status {
	if !m.op.TryLock() {
		return m.Status()
	}
	defer m.op.Unlock()
	status := m.Status()
	if status.State == "ready" || !status.CanInstall || status.Latest == "" {
		return status
	}
	status.State, status.Message, status.Written, status.Total = "downloading", "", 0, 0
	m.mu.Lock()
	m.logPercent = -1
	m.mu.Unlock()
	m.log.write("download_started", "version="+status.Latest)
	m.publish(status)
	if err := m.updater.DownloadAndInstall(ctx); err != nil {
		status = m.Status()
		m.log.write("download_failed", diagnosticReason(err))
		status.State, status.Message = "error", downloadErrorMessage(err)
		return m.publish(status)
	}
	if err := validateBundle(m.updater.DownloadedPath(), status.Latest); err != nil {
		m.log.write("bundle_failed", diagnosticReason(err))
		status = m.Status()
		status.State, status.Message = "error", err.Error()
		return m.publish(status)
	}
	status = m.Status()
	status.State = "ready"
	m.log.write("update_ready", "version="+status.Latest)
	return m.publish(status)
}

func (m *Manager) Progress(progress updater.Progress) {
	m.mu.Lock()
	status := m.status
	if status.State != "downloading" {
		m.mu.Unlock()
		return
	}
	status.Written, status.Total = progress.Written, progress.Total
	logProgress := false
	if progress.Total > 0 && progress.Written*10/progress.Total > m.logPercent {
		m.logPercent = progress.Written * 10 / progress.Total
		logProgress = true
	}
	m.status = status
	m.mu.Unlock()
	if logProgress {
		m.log.write("download_progress", fmt.Sprintf("bytes=%d total=%d", progress.Written, progress.Total))
	}
	if m.notify != nil {
		m.notify(status)
	}
}

func (m *Manager) Stage(stage string) {
	status := m.Status()
	if status.State != "downloading" && status.State != "verifying" {
		return
	}
	status.State = stage
	m.log.write("update_stage", stage)
	m.publish(status)
}

func (m *Manager) Restart(ctx context.Context) error {
	if !m.op.TryLock() {
		return errors.New("An update is already in progress.")
	}
	defer m.op.Unlock()
	if m.Status().State != "ready" {
		return errors.New("Download the update before restarting.")
	}
	if reason := installReason(m.fixture); reason != "" {
		m.log.write("restart_blocked", "installation_location_or_permissions")
		return errors.New(reason)
	}
	if err := validateBundle(m.updater.DownloadedPath(), m.Status().Latest); err != nil {
		m.log.write("restart_bundle_failed", diagnosticReason(err))
		return err
	}
	m.log.write("restart_requested", "version="+m.Status().Latest)
	if m.log != nil {
		_ = os.WriteFile(filepath.Join(filepath.Dir(m.log.path), "pending-helper"), []byte(strconv.Itoa(os.Getpid())), 0600)
	}
	if err := m.updater.Restart(ctx); err != nil {
		m.log.write("restart_failed", diagnosticReason(err))
		return errors.New("The update could not restart. Open update diagnostics for details.")
	}
	return nil
}

// Background checks never download, install, open a window, or discard a draft.
func (m *Manager) Start() {
	m.mu.Lock()
	if m.fixture || m.cancel != nil {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel, m.done = cancel, make(chan struct{})
	done := m.done
	m.mu.Unlock()
	go func() {
		defer close(done)
		m.Check(ctx)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.Check(ctx)
			}
		}
	}()
}

func (m *Manager) Close() {
	m.mu.Lock()
	cancel, done := m.cancel, m.done
	m.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func appBundle() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	parent := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if filepath.Ext(parent) != ".app" {
		return ""
	}
	return parent
}
