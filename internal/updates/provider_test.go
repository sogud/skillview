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

func (testHost) Emit(string, ...any) bool         { return true }
func (testHost) OnEvent(string, func(any)) func() { return func() {} }
func (testHost) OpenWindow(updater.WindowOptions) updater.WindowHandle {
	panic("unexpected update window")
}
func (testHost) Quit() { panic("unexpected restart") }

type releaseFixture struct {
	server          *httptest.Server
	public          ed25519.PublicKey
	private         ed25519.PrivateKey
	version         string
	manifestVersion string
	zip             []byte
	download        []byte
	unsigned        bool
	missingManifest bool
	malformed       bool
	rateLimited     bool
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
		if f.missingManifest {
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
	if err := u.Init(updater.Config{CurrentVersion: Version(), Platform: "darwin", Arch: "arm64", Providers: []updater.Provider{provider}, PublicKey: key, Window: updater.WindowNone}); err != nil {
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

func TestTamperedOrWrongKeyUpdateNeverBecomesReady(t *testing.T) {
	for _, mode := range []string{"tampered", "wrong key"} {
		t.Run(mode, func(t *testing.T) {
			f := newReleaseFixture(t)
			key := f.public
			if mode == "tampered" {
				f.download = []byte("tampered archive")
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

func TestMissingManifestIsManualOnlyAndUnsignedManifestIsRejected(t *testing.T) {
	for _, mode := range []string{"missing", "unsigned", "malformed", "version mismatch"} {
		t.Run(mode, func(t *testing.T) {
			f := newReleaseFixture(t)
			f.missingManifest, f.unsigned, f.malformed = mode == "missing", mode == "unsigned", mode == "malformed"
			if mode == "version mismatch" {
				f.manifestVersion = "0.1.0-alpha.5"
			}
			u := fixtureUpdater(t, f, f.public)
			release, err := u.Check(context.Background())
			if mode == "missing" {
				if err != nil || release == nil || hasSignature(release) {
					t.Fatalf("manual release: %#v %v", release, err)
				}
				if err := u.DownloadAndInstall(context.Background()); err == nil {
					t.Fatal("unsigned download allowed")
				}
			} else if err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func TestVersionOrderingAndArchitectureSelection(t *testing.T) {
	for _, version := range []string{Version(), "0.1.0-alpha.1"} {
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

func TestManualReleaseCannotDownloadThroughProvider(t *testing.T) {
	f := newReleaseFixture(t)
	f.missingManifest = true
	p, err := newProvider(f.server.Client(), f.server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	release, err := p.Check(context.Background(), updater.CheckRequest{CurrentVersion: Version(), Platform: "darwin", Arch: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Download(context.Background(), release, io.Discard, func(int64, int64) {}); err == nil {
		t.Fatal("unsigned release downloaded")
	}
}
