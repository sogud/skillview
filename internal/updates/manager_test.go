package updates

import (
	"context"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestManagerDownloadsOnlyAfterExplicitRequest(t *testing.T) {
	f := newReleaseFixture(t)
	u := fixtureUpdater(t, f, f.public)
	m := &Manager{updater: u, fixture: true, status: Status{Version: Version(), State: "idle"}}
	status := m.Check(context.Background())
	if status.State != "available" || status.CanInstall || status.Latest != f.version {
		t.Fatalf("preview status: %#v", status)
	}
	if u.DownloadedPath() != "" {
		t.Fatal("check automatically installed an update")
	}
	status = m.Download(context.Background())
	if status.State != "available" || u.DownloadedPath() != "" {
		t.Fatal("preview installed an update")
	}
	if err := m.Restart(context.Background()); err == nil {
		t.Fatal("restart without download allowed")
	}
}

func TestBackgroundChecksAreDisabledForFixtures(t *testing.T) {
	f := newReleaseFixture(t)
	m := &Manager{updater: fixtureUpdater(t, f, f.public), fixture: true, status: Status{Version: Version(), State: "idle"}}
	m.Start()
	m.Close()
	if m.cancel != nil || m.Status().State != "idle" {
		t.Fatal("fixture started automatic network checks")
	}
}

func TestManagerPreservesReadyUpdateAndIgnoresLateProgress(t *testing.T) {
	f := newReleaseFixture(t)
	m := &Manager{updater: fixtureUpdater(t, f, f.public), status: Status{Version: Version(), State: "ready", Latest: f.version}}
	status := m.Check(context.Background())
	if status.State != "ready" || status.Latest != f.version {
		t.Fatal("check discarded staged update")
	}
	m.Progress(updater.Progress{Written: 100, Total: 100})
	if m.Status().Written != 0 {
		t.Fatal("late progress changed completed status")
	}
}

func TestUpdateFixtureCannotOverrideProductionFeed(t *testing.T) {
	t.Setenv("SKILLVIEW_UPDATE_API", "https://untrusted.invalid")
	if _, err := New(updater.New(testHost{}), true, nil); err == nil {
		t.Fatal("non-loopback fixture accepted")
	}
	if _, err := New(updater.New(testHost{}), false, nil); err != nil {
		t.Fatal("ordinary launch used fixture override:", err)
	}
}

func TestBundleValidationRejectsMissingOrNonAppPayload(t *testing.T) {
	if err := validateBundle(t.TempDir(), Version()); err == nil {
		t.Fatal("non-app accepted")
	}
	if err := validateBundle(t.TempDir()+"/Missing.app", Version()); err == nil {
		t.Fatal("missing app accepted")
	}
}

func TestFailedRecheckClearsPreviousInstallOffer(t *testing.T) {
	f := newReleaseFixture(t)
	f.missingManifest = true
	u := fixtureUpdater(t, f, f.public)
	m := &Manager{updater: u, fixture: true, status: Status{Version: fixtureCurrentVersion, State: "available", Latest: f.version, CanInstall: true}}
	status := m.Check(context.Background())
	if status.State != "error" || status.CanInstall || status.Latest != "" || status.ReleaseURL != "" {
		t.Fatalf("failed check retained stale install offer: %#v", status)
	}
	m.Download(context.Background())
	if f.downloadCount != 0 {
		t.Fatal("failed recheck allowed download")
	}
}
