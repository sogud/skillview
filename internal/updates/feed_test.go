package updates

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestRateLimitedAPIUsesFeedWithTheSameSignatureVerification(t *testing.T) {
	for _, mode := range []string{"signed", "tampered", "unsigned", "missing"} {
		t.Run(mode, func(t *testing.T) {
			f := newReleaseFixture(t)
			f.rateLimited = true
			f.unsigned = mode == "unsigned"
			f.missingManifest = mode == "missing"
			if mode == "tampered" {
				f.download = []byte("tampered payload")
			}
			u := fixtureUpdater(t, f, f.public)
			release, err := u.Check(context.Background())
			if mode == "unsigned" || mode == "missing" {
				if err == nil || release != nil {
					t.Fatal("feed accepted missing or unsigned manifest")
				}
				return
			}
			if err != nil || release == nil || release.Version != f.version {
				t.Fatal("feed check", release, err)
			}
			err = u.DownloadAndInstall(context.Background())
			if mode == "signed" {
				if err != nil || u.State() != updater.StateReady {
					t.Fatal("signed feed download", err)
				}
				_ = os.RemoveAll(filepath.Dir(u.DownloadedPath()))
			} else if err == nil || u.State() == updater.StateReady {
				t.Fatal("unsafe feed update was staged")
			}
		})
	}
}

func TestFeedHonorsStableChannelAndVersionOrdering(t *testing.T) {
	f := newReleaseFixture(t)
	f.rateLimited = true
	p, _ := newProvider(f.server.Client(), f.server.URL, false)
	req := updater.CheckRequest{CurrentVersion: fixtureCurrentVersion, Platform: "darwin", Arch: "arm64"}
	if release, err := p.Check(context.Background(), req); err != nil || release != nil {
		t.Fatal("stable app offered prerelease", release, err)
	}
	p.prerelease = true
	f.version = "0.1.0-alpha.1"
	if release, err := p.Check(context.Background(), req); err != nil || release != nil {
		t.Fatal("older feed offered", release, err)
	}
}
