package library

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func frontmatter(data []byte) ([]byte, []byte, bool) {
	firstEnd := bytes.IndexByte(data, '\n')
	if firstEnd < 0 || strings.TrimSuffix(string(data[:firstEnd]), "\r") != "---" {
		return nil, data, false
	}
	start := firstEnd + 1
	for offset := start; offset < len(data); {
		end := bytes.IndexByte(data[offset:], '\n')
		next := len(data)
		if end < 0 {
			end = len(data)
		} else {
			end += offset
			next = end + 1
		}
		line := strings.TrimSuffix(string(data[offset:end]), "\r")
		if line == "---" {
			return data[start:offset], data[next:], true
		}
		offset = next
	}
	return nil, data, false
}

func parseSkill(data []byte, directory string) (string, string, []string) {
	name := filepath.Base(directory)
	description := ""
	warnings := []string{}
	header, _, found := frontmatter(data)
	if !found {
		return name, description, []string{"Missing or unclosed YAML frontmatter. Open the raw file to repair it."}
	}
	var metadata map[string]any
	if err := yaml.Unmarshal(header, &metadata); err != nil {
		return name, description, []string{"YAML could not be parsed: " + err.Error()}
	}
	parsedName, ok := metadata["name"].(string)
	if !ok || parsedName == "" {
		warnings = append(warnings, "name must be a non-empty string")
	} else {
		name = parsedName
		if !validName.MatchString(name) || len(name) > 64 {
			warnings = append(warnings, "name must use lowercase letters, numbers, and single hyphens (up to 64 characters)")
		}
		if name != filepath.Base(directory) {
			warnings = append(warnings, "name does not match the Skill directory")
		}
	}
	description, ok = metadata["description"].(string)
	if !ok || strings.TrimSpace(description) == "" || utf8.RuneCountInString(description) > 1024 {
		warnings = append(warnings, "description must be a non-empty string of up to 1024 characters")
	}
	return name, description, warnings
}
