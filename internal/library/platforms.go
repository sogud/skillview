package library

import (
	"fmt"
	"sort"
	"strings"
)

type Platform struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Paths     []string `json:"paths"`
	WritePath string   `json:"writePath,omitempty"`
}

// The registry supplies both browsing paths and explicit write destinations.
func Platforms(home string) []Platform {
	byName := map[string]*Platform{}
	for _, location := range Locations() {
		item := byName[location.Tool]
		if item == nil {
			item = &Platform{Name: location.Tool}
			byName[location.Tool] = item
		}
		if len(location.CLIAliases) > 0 {
			item.ID = location.CLIAliases[0]
		}
		for i, path := range location.Global {
			expanded := expandLocation(path, home)
			if !hasString(item.Paths, expanded) {
				item.Paths = append(item.Paths, expanded)
			}
			if location.WritableGlobal && i == 0 {
				item.WritePath = expanded
			}
		}
	}
	result := []Platform{}
	for _, item := range byName {
		result = append(result, *item)
	}
	// Agents is the shared directory already present in the registry, not another path list.
	for _, item := range result {
		if item.ID == "codex" {
			result = append(result, Platform{ID: "agents", Name: "Common Skills", Paths: []string{item.WritePath}, WritePath: item.WritePath})
			break
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func FindPlatform(home, name string) (Platform, error) {
	for _, platform := range Platforms(home) {
		if strings.EqualFold(platform.ID, name) || strings.EqualFold(platform.Name, name) {
			return platform, nil
		}
	}
	return Platform{}, fmt.Errorf("Unknown Agent: %s", name)
}

func (p Platform) Includes(skill Skill) bool {
	for _, occurrence := range skill.Occurrences {
		for _, root := range p.Paths {
			if _, inside := within(root, occurrence.Path); inside {
				return true
			}
		}
	}
	for _, root := range p.Paths {
		if _, inside := within(root, skill.Path); inside {
			return true
		}
	}
	return false
}
