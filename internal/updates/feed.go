package updates

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"golang.org/x/mod/semver"
)

func (p *releaseProvider) checkFeed(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.feedURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("release feed is unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 1024*1024 {
		return nil, errors.New("release feed is too large")
	}
	var feed struct {
		XMLName xml.Name
		Entries []struct {
			Links []struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if xml.Unmarshal(body, &feed) != nil || feed.XMLName.Local != "feed" || feed.XMLName.Space != "http://www.w3.org/2005/Atom" {
		return nil, errors.New("release feed could not be read")
	}
	latest := ""
	validVersion := false
	prefix := p.releasesURL + "/tag/"
	for _, entry := range feed.Entries {
		for _, link := range entry.Links {
			if !strings.HasPrefix(link.Href, prefix) {
				continue
			}
			tag := strings.TrimPrefix(link.Href, prefix)
			if !semver.IsValid(tag) {
				continue
			}
			validVersion = true
			if !p.prerelease && semver.Prerelease(tag) != "" {
				continue
			}
			if latest == "" || semver.Compare(tag, latest) > 0 {
				latest = tag
			}
		}
	}
	if !validVersion && len(feed.Entries) > 0 {
		return nil, errors.New("release feed has no compatible versions")
	}
	current := req.CurrentVersion
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if latest == "" || semver.Compare(latest, current) <= 0 {
		return nil, nil
	}
	if req.Platform != "darwin" || req.Arch != "arm64" {
		return nil, errors.New("release has no compatible app")
	}
	version := strings.TrimPrefix(latest, "v")
	filename := "SkillView-" + version + "-macos-" + req.Arch + ".zip"
	return &updater.Release{Version: version, Artifact: updater.Artifact{Filename: filename, Filetype: "zip", Platform: req.Platform, Arch: req.Arch},
		Metadata: map[string]any{"github.asset.url": p.downloadURL + "/" + latest + "/" + filename, "github.release.htmlURL": prefix + latest}}, nil
}
