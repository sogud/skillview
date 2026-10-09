package updates

import (
	"context"
	"crypto/ed25519"
	"crypto/sha512"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// GitHub selects the release; its signed Wails manifest authenticates the ZIP.
// Checksums alone never authorize installation.
type releaseProvider struct {
	github *github.Provider
	client *http.Client
}

func newProvider(client *http.Client, baseURL string, prerelease bool) (*releaseProvider, error) {
	p, err := github.New(github.Config{
		Repository: "sogud/skillview", Prerelease: prerelease,
		HTTPClient: client, BaseURL: baseURL, AssetMatcher: matchZIP,
	})
	return &releaseProvider{github: p, client: client}, err
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

func (p *releaseProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	release, err := p.github.Check(ctx, req)
	if err != nil || release == nil {
		return release, err
	}
	assetURL, _ := release.Metadata["github.asset.url"].(string)
	manifestURL := strings.TrimSuffix(assetURL, "/"+release.Artifact.Filename) + "/manifest.json"
	manifest, err := endpoint.New(endpoint.Config{URL: manifestURL, HTTPClient: p.client})
	if err != nil {
		return nil, err
	}
	signed, err := manifest.Check(ctx, req)
	if err != nil {
		return nil, err
	}
	if signed == nil {
		// Earlier releases did not ship a manifest. Show the release, but offer
		// only its manual download link rather than accepting an unsigned ZIP.
		return release, nil
	}
	if signed.Version != release.Version || signed.Artifact.Filename != release.Artifact.Filename ||
		signed.Metadata["endpoint.artifact.url"] != assetURL || !hasSignature(signed) {
		return nil, errors.New("The update could not be verified. Download it from GitHub instead.")
	}
	signed.Notes = release.Notes
	signed.Metadata["github.release.htmlURL"] = release.Metadata["github.release.htmlURL"]
	signed.Metadata["skillview.manifestURL"] = manifestURL
	return signed, nil
}

func hasSignature(release *updater.Release) bool {
	v := release.Verification
	return v != nil && v.DigestAlgo == "sha512" && len(v.Digest) == sha512.Size &&
		v.SignatureAlgo == "ed25519ph" && len(v.Signature) == ed25519.SignatureSize
}

func (p *releaseProvider) Download(ctx context.Context, release *updater.Release, dst io.Writer, progress func(int64, int64)) error {
	if !hasSignature(release) {
		return errors.New("This release supports manual installation only.")
	}
	manifestURL, _ := release.Metadata["skillview.manifestURL"].(string)
	manifest, err := endpoint.New(endpoint.Config{URL: manifestURL, HTTPClient: p.client})
	if err != nil {
		return err
	}
	return manifest.Download(ctx, release, dst, progress)
}
