package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

type testHost struct{}

const fixtureCurrentVersion = "0.1.0-alpha.3"

func (testHost) Emit(string, ...any) bool         { return true }
func (testHost) OnEvent(string, func(any)) func() { return func() {} }
func (testHost) OpenWindow(updater.WindowOptions) updater.WindowHandle {
	panic("unexpected update window")
}
func (testHost) Quit() { panic("unexpected restart") }

type releaseFixture struct {
	server               *httptest.Server
	public               ed25519.PublicKey
	private              ed25519.PrivateKey
	version              string
	manifestVersion      string
	zip                  []byte
	download             []byte
	unsigned             bool
	missingManifest      bool
	missingFirstManifest bool
	manifestRequests     int
	malformed            bool
	rateLimited          bool
	cutoff               bool
	downloadCount        int
}

func newReleaseFixture(t *testing.T) *releaseFixture {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create("SkillView.app/Contents/Resources/fixture.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte("new app resources"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	f := &releaseFixture{public: public, private: private, version: "0.1.0-alpha.4", zip: buf.Bytes()}
	f.download = f.zip
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *releaseFixture) handle(w http.ResponseWriter, r *http.Request) {
	filename := "SkillView-" + f.version + "-macos-arm64.zip"
	assetURL := f.server.URL + "/download/v" + f.version + "/" + filename
	switch {
	case strings.HasPrefix(r.URL.Path, "/repos/sogud/skillview/releases"):
		if f.rateLimited {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(403)
			return
		}
		release := map[string]any{"tag_name": "v" + f.version, "html_url": "https://github.com/sogud/skillview/releases/tag/v" + f.version,
			"body": "Fixture release notes", "prerelease": true, "assets": []any{
				map[string]any{"name": filename, "browser_download_url": assetURL, "size": len(f.zip)},
				map[string]any{"name": "SkillView-" + f.version + "-macos-amd64.zip", "browser_download_url": assetURL + "-wrong-arch"},
			}}
		if r.URL.Path == "/repos/sogud/skillview/releases/latest" {
			_ = json.NewEncoder(w).Encode(release)
		} else {
			_ = json.NewEncoder(w).Encode([]any{release})
		}
	case r.URL.Path == "/releases.atom":
		_, _ = fmt.Fprintf(w, `<feed xmlns="http://www.w3.org/2005/Atom"><entry><link href="%s/releases/tag/v%s"/></entry></feed>`, f.server.URL, f.version)
	case strings.HasSuffix(r.URL.Path, "/manifest.json"):
		f.manifestRequests++
		if f.missingManifest || (f.missingFirstManifest && f.manifestRequests == 1) {
			http.NotFound(w, r)
			return
		}
		if f.malformed {
			_, _ = w.Write([]byte("invalid manifest"))
			return
		}
		digest := sha512.Sum512(f.zip)
		signature, _ := f.private.Sign(nil, digest[:], &ed25519.Options{Hash: crypto.SHA512})
		artifact := map[string]any{"platform": "darwin", "arch": "arm64", "url": assetURL, "size": len(f.zip), "digestAlgo": "sha512", "digest": base64.StdEncoding.EncodeToString(digest[:])}
		if !f.unsigned {
			artifact["signatureAlgo"], artifact["signature"] = "ed25519ph", base64.StdEncoding.EncodeToString(signature)
		}
		version := f.version
		if f.manifestVersion != "" {
			version = f.manifestVersion
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"schemaVersion": 1, "version": version, "artifacts": []any{artifact}})
	case r.URL.Path == "/download/v"+f.version+"/"+filename:
		f.downloadCount++
		if f.cutoff && f.downloadCount == 1 {
			w.Header().Set("Content-Length", fmt.Sprint(len(f.download)))
			_, _ = w.Write(f.download[:len(f.download)/2])
			w.(http.Flusher).Flush()
			conn, _, _ := w.(http.Hijacker).Hijack()
			_ = conn.Close()
			return
		}
		if f.cutoff {
			if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", len(f.download)/2) {
				w.WriteHeader(400)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", len(f.download)/2, len(f.download)-1, len(f.download)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(f.download[len(f.download)/2:])
			return
		}
		_, _ = w.Write(f.download)
	default:
		http.NotFound(w, r)
	}
}

func fixtureUpdater(t *testing.T, f *releaseFixture, key ed25519.PublicKey) *updater.Updater {
	t.Helper()
	provider, err := newProvider(f.server.Client(), f.server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	u := updater.New(testHost{})
	if err := u.Init(updater.Config{CurrentVersion: fixtureCurrentVersion, Platform: "darwin", Arch: "arm64", Providers: []updater.Provider{provider}, PublicKey: key, Window: updater.WindowNone}); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestSignedUpdateDownloadsVerifiesAndExtractsBundle(t *testing.T) {
	f := newReleaseFixture(t)
	u := fixtureUpdater(t, f, f.public)
	release, err := u.Check(context.Background())
	if err != nil || release == nil || release.Version != f.version || release.Notes != "Fixture release notes" {
		t.Fatalf("release: %#v %v", release, err)
	}
	if err := u.DownloadAndInstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	path := u.DownloadedPath()
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })
	text, err := os.ReadFile(filepath.Join(path, "Contents", "Resources", "fixture.txt"))
	if err != nil || string(text) != "new app resources" || u.State() != updater.StateReady {
		t.Fatalf("staged payload: %q %v (%s)", text, err, u.State())
	}
}

func TestResumedSignedUpdateStillVerifiesAndExtracts(t *testing.T) {
	f := newReleaseFixture(t)
	f.cutoff = true
	u := fixtureUpdater(t, f, f.public)
	if _, err := u.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := u.DownloadAndInstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	path := u.DownloadedPath()
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })
	if _, err := os.Stat(filepath.Join(path, "Contents", "Resources", "fixture.txt")); err != nil {
		t.Fatal(err)
	}
	if f.downloadCount != 2 || u.State() != updater.StateReady {
		t.Fatal("resume was not verified", f.downloadCount, u.State())
	}
}

func TestTamperedOrWrongKeyUpdateNeverBecomesReady(t *testing.T) {
	for _, mode := range []string{"tampered", "resumed tampered", "wrong key"} {
		t.Run(mode, func(t *testing.T) {
			f := newReleaseFixture(t)
			key := f.public
			if mode == "tampered" || mode == "resumed tampered" {
				f.cutoff = mode == "resumed tampered"
				f.download = append([]byte(nil), f.zip...)
				f.download[0] ^= 1
			} else {
				key, _, _ = ed25519.GenerateKey(rand.Reader)
			}
			u := fixtureUpdater(t, f, key)
			if _, err := u.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := u.DownloadAndInstall(context.Background()); err == nil {
				t.Fatal("untrusted payload accepted")
			}
			if u.DownloadedPath() != "" || u.State() == updater.StateReady {
				t.Fatal("untrusted payload staged")
			}
		})
	}
}

func TestMissingOrInvalidManifestFailsClosed(t *testing.T) {
	for _, mode := range []string{"missing", "unsigned", "malformed", "version mismatch"} {
		t.Run(mode, func(t *testing.T) {
			f := newReleaseFixture(t)
			f.missingManifest, f.unsigned, f.malformed = mode == "missing", mode == "unsigned", mode == "malformed"
			if mode == "version mismatch" {
				f.manifestVersion = "0.1.0-alpha.5"
			}
			u := fixtureUpdater(t, f, f.public)
			release, err := u.Check(context.Background())
			if err == nil || release != nil {
				t.Fatal("missing or invalid manifest accepted")
			}
			if u.DownloadedPath() != "" || u.State() == updater.StateReady {
				t.Fatal("invalid release staged")
			}
		})
	}
}

func TestVersionOrderingAndArchitectureSelection(t *testing.T) {
	for _, version := range []string{fixtureCurrentVersion, "0.1.0-alpha.1"} {
		f := newReleaseFixture(t)
		f.version = version
		u := fixtureUpdater(t, f, f.public)
		if release, err := u.Check(context.Background()); err != nil || release != nil {
			t.Fatalf("older/equal version offered: %#v %v", release, err)
		}
	}
	assets := []github.ReleaseAsset{{Name: "SkillView-0.2.0-macos-amd64.zip"}, {Name: "SkillView-0.2.0-macos-arm64.dmg"}, {Name: "SkillView-0.2.0-macos-arm64.zip"}}
	if got := matchZIP(updater.CheckRequest{Platform: "darwin", Arch: "arm64"}, assets); got != 2 {
		t.Fatalf("asset index %d", got)
	}
	if got := matchZIP(updater.CheckRequest{Platform: "windows", Arch: "arm64"}, assets); got != -1 {
		t.Fatal("unsupported platform offered")
	}
}

func TestCancelledCheckKeepsCurrentApp(t *testing.T) {
	f := newReleaseFixture(t)
	u := fixtureUpdater(t, f, f.public)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := u.Check(ctx); err == nil {
		t.Fatal("cancelled check succeeded")
	}
	if u.DownloadedPath() != "" {
		t.Fatal("cancelled check staged an update")
	}
}

func TestUnsignedReleaseCannotDownloadThroughProvider(t *testing.T) {
	p := &releaseProvider{}
	for _, release := range []*updater.Release{nil, {Version: "0.1.0"}} {
		if err := p.Download(context.Background(), release, io.Discard, func(int64, int64) {}); err == nil {
			t.Fatal("unsigned release downloaded")
		}
	}
}

func TestTransientMissingManifestRecoversThroughManager(t *testing.T) {
	f := newReleaseFixture(t)
	f.missingFirstManifest = true
	u := fixtureUpdater(t, f, f.public)
	m := &Manager{updater: u, fixture: true, status: Status{Version: fixtureCurrentVersion, State: "idle"}}
	status := m.Check(context.Background())
	if f.manifestRequests != 2 || status.State != "available" || status.Latest != f.version || status.Message != "Updates cannot be installed in a preview window." {
		t.Fatalf("manifest retry lost signature: requests=%d status=%#v", f.manifestRequests, status)
	}
	if u.DownloadedPath() != "" {
		t.Fatal("check downloaded automatically")
	}
}

func TestPersistentMissingManifestIsRetryableCheckError(t *testing.T) {
	f := newReleaseFixture(t)
	f.missingManifest = true
	u := fixtureUpdater(t, f, f.public)
	m := &Manager{updater: u, fixture: true, status: Status{Version: fixtureCurrentVersion, State: "idle"}}
	status := m.Check(context.Background())
	if f.manifestRequests != 2 || status.State != "error" || status.CanInstall || status.Message != invalidReleaseMessage {
		t.Fatalf("missing manifest incorrectly classified: %#v", status)
	}
	f.missingManifest = false
	status = m.Check(context.Background())
	if status.State != "available" || status.Message != "Updates cannot be installed in a preview window." {
		t.Fatalf("retry did not recover signed update: %#v", status)
	}
}
