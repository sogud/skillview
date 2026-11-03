package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type LocationNode struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type LocationChain struct {
	SourceID  string         `json:"sourceId"`
	Path      string         `json:"path"`
	Nodes     []LocationNode `json:"nodes"`
	Status    string         `json:"status"`
	Removable bool           `json:"removable"`
}

type LocationReport struct {
	OriginalPath string          `json:"originalPath"`
	Chains       []LocationChain `json:"chains"`
}

// This is an on-demand metadata view of catalogued entries, not a new scan or a
// content reader. It never grants editing access or follows links for deletion.
func (l *Library) Locations(id, sourceID, path string) (LocationReport, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	report := LocationReport{Chains: []LocationChain{}}
	if id != "" {
		for _, skill := range l.catalog.Skills {
			if skill.ID != id {
				continue
			}
			report.OriginalPath = skill.Path
			seen := map[string]bool{}
			for _, entry := range skill.Occurrences {
				if seen[entry.Path] {
					continue
				}
				seen[entry.Path] = true
				chain := resolveLocation(entry.Path)
				chain.SourceID = entry.SourceID
				chain.Removable = entry.Removable
				last := chain.Nodes[len(chain.Nodes)-1]
				if chain.Status == "ready" && last.Path != skill.Path {
					chain.Status = "changed"
				}
				report.Chains = append(report.Chains, chain)
			}
			return report, nil
		}
	} else {
		// A broken or inaccessible entry may not have a Skill. Only paths from
		// this catalog's diagnostics can be inspected through this branch.
		for _, issue := range l.catalog.Issues {
			if issue.SourceID == sourceID && issue.Path == path && filepath.IsAbs(path) {
				chain := resolveLocation(path)
				chain.SourceID = sourceID
				if chain.Status == "ready" {
					chain.Status = "unavailable"
				}
				report.Chains = append(report.Chains, chain)
				return report, nil
			}
		}
	}
	return report, fmt.Errorf("Entry is no longer in the catalog. Scan folders again.")
}

func resolveLocation(path string) LocationChain {
	result := LocationChain{Path: path, Status: "ready", Nodes: []LocationNode{{Path: path, Kind: "entry"}}}
	prefix := string(filepath.Separator)
	pending := strings.Split(path, string(filepath.Separator))
	seen := map[string]bool{}
	hops := 0
	var info os.FileInfo
	fail := func(status string) LocationChain {
		result.Status = status
		result.Nodes = append(result.Nodes, LocationNode{Path: prefix, Kind: "unavailable"})
		return result
	}
	// Consume components in filesystem order. In a link target, x/../y must
	// resolve x before ..; lexical cleaning would show the wrong real location.
	for len(pending) > 0 {
		state := prefix + "\x00" + strings.Join(pending, "/")
		if seen[state] {
			result.Status = "cycle"
			return result
		}
		seen[state] = true
		part := pending[0]
		pending = pending[1:]
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			prefix = filepath.Dir(prefix)
			continue
		}
		parent := prefix
		prefix = filepath.Join(prefix, part)
		var err error
		info, err = os.Lstat(prefix)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return fail("denied")
			}
			if errors.Is(err, os.ErrNotExist) {
				return fail("missing")
			}
			if errors.Is(err, syscall.ELOOP) {
				return fail("cycle")
			}
			return fail("unavailable")
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		hops++
		if hops > 40 {
			return fail("limit")
		}
		target, err := os.Readlink(prefix)
		if err != nil {
			return fail("unavailable")
		}
		if result.Nodes[len(result.Nodes)-1].Path == prefix {
			result.Nodes[len(result.Nodes)-1].Kind = "link"
		} else {
			result.Nodes = append(result.Nodes, LocationNode{Path: prefix, Kind: "link"})
		}
		prefix = parent
		if filepath.IsAbs(target) {
			prefix = string(filepath.Separator)
		}
		pending = append(strings.Split(target, string(filepath.Separator)), pending...)
	}
	info, err := os.Lstat(prefix)
	if err != nil {
		return fail("unavailable")
	}
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	if result.Nodes[len(result.Nodes)-1].Path == prefix {
		result.Nodes[len(result.Nodes)-1].Kind = kind
	} else {
		result.Nodes = append(result.Nodes, LocationNode{Path: prefix, Kind: kind})
	}
	return result
}
