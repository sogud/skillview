package updates

import (
	"context"
	"crypto/ed25519"
	"crypto/sha512"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// GitHub selects the release; its signed Wails manifest authenticates the ZIP.
// Checksums alone never authorize installation.
type releaseProvider struct {
	github      *github.Provider
	client      *http.Client
	releasesURL string
	feedURL     string
	downloadURL string
	prerelease  bool
	log         func(string, string)
}

func newProvider(client *http.Client, baseURL string, prerelease bool) (*releaseProvider, error) {
	copy := *client
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	copy.Transport = statusTransport{transport}
	client = &copy
	p, err := github.New(github.Config{
		Repository: "sogud/skillview", Prerelease: prerelease,
		HTTPClient: client, BaseURL: baseURL, AssetMatcher: matchZIP,
	})
	releasesURL := "https://github.com/sogud/skillview/releases"
	downloadURL := releasesURL + "/download"
	feedURL := releasesURL + ".atom"
	if baseURL != "" {
		releasesURL = strings.TrimRight(baseURL, "/") + "/releases"
		feedURL = releasesURL + ".atom"
		downloadURL = strings.TrimRight(baseURL, "/") + "/download"
	}
	return &releaseProvider{github: p, client: client, releasesURL: releasesURL, feedURL: feedURL, downloadURL: downloadURL, prerelease: prerelease}, err
}

func matchZIP(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	suffix := "-macos-" + req.Arch + ".zip"
	if req.Platform != "darwin" {
		return -1
	}
	for i, asset := range assets {
		if strings.HasPrefix(asset.Name, "SkillView-") && strings.HasSuffix(asset.Name, suffix) {
			return i
		}
	}
	return -1
}

func (p *releaseProvider) Name() string { return "skillview-github" }

func (p *releaseProvider) Check(ctx context.Context, req updater.CheckRequest) (result *updater.Release, failure error) {
	defer func() {
		if failure != nil {
			if p.log != nil {
				p.log("check_failure_detail", diagnosticReason(failure))
			}
			failure = errors.New(checkErrorMessage(failure))
		}
	}()
	release, err := p.github.Check(ctx, req)
	if err != nil && ctx.Err() == nil {
		// The public Atom feed does not consume GitHub API quota. It chooses a
		// tag only; the same signed-manifest validation below still applies.
		fallback, fallbackErr := p.checkFeed(ctx, req)
		if fallbackErr == nil {
			release, err = fallback, nil
		}
	}
	if err != nil || release == nil {
		return release, err
	}
	assetURL, _ := release.Metadata["github.asset.url"].(string)
	manifestURL := strings.TrimSuffix(assetURL, "/"+release.Artifact.Filename) + "/manifest.json"
	manifest, err := endpoint.New(endpoint.Config{URL: manifestURL, HTTPClient: p.client, Headers: map[string]string{"Cache-Control": "no-cache"}})
	if err != nil {
		return nil, err
	}
	signed, err := manifest.Check(ctx, req)
	if err != nil {
		return nil, err
	}
	if signed == nil {
		// GitHub can expose release metadata before a cached manifest request
		// becomes available. An absent manifest does not prove the release is
		// intentionally unsigned. Retry once, then fail closed with a retryable
		// check error instead of permanently labelling it manual-only.
		if p.log != nil {
			p.log("manifest_retry", "reason=missing_or_stale")
		}
		timer := time.NewTimer(250 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
		signed, err = manifest.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		if signed == nil {
			if p.log != nil {
				p.log("manifest_unavailable", "reason=missing_or_stale")
			}
			return nil, errors.New(invalidReleaseMessage)
		}
	}
	if signed.Version != release.Version || signed.Artifact.Filename != release.Artifact.Filename ||
		signed.Metadata["endpoint.artifact.url"] != assetURL || !hasSignature(signed) {
		return nil, errors.New("The update could not be verified. Download it from GitHub instead.")
	}
	if release.Notes != "" {
		signed.Notes = release.Notes
	}
	signed.Metadata["github.release.htmlURL"] = release.Metadata["github.release.htmlURL"]
	signed.Metadata["skillview.manifestURL"] = manifestURL
	return signed, nil
}

func hasSignature(release *updater.Release) bool {
	if release == nil {
		return false
	}
	v := release.Verification
	return v != nil && v.DigestAlgo == "sha512" && len(v.Digest) == sha512.Size &&
		v.SignatureAlgo == "ed25519ph" && len(v.Signature) == ed25519.SignatureSize
}

func (p *releaseProvider) Download(ctx context.Context, release *updater.Release, dst io.Writer, progress func(int64, int64)) error {
	if !hasSignature(release) {
		return errors.New("This release supports manual installation only.")
	}
	return p.download(ctx, release, dst, progress)
}
