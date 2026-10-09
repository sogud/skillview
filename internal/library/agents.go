package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type AgentInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Installed    bool   `json:"installed"`
	Configured   bool   `json:"configured"`
	Evidence     string `json:"evidence"`
	SharedSkills bool   `json:"sharedSkills"`
}

func Agents(home string) []AgentInfo {
	byName := map[string]AgentInfo{}
	for _, location := range Locations() {
		agent, exists := byName[location.Tool]
		if !exists {
			agent = AgentInfo{ID: objectID(location.Tool), Name: location.Tool, Icon: agentIcon(location.Tool)}
		}
		if hasString(location.Project, ".agents/skills") {
			agent.SharedSkills = true
		}
		for _, name := range location.Apps {
			for _, folder := range []string{"/Applications", filepath.Join(home, "Applications")} {
				path := filepath.Join(folder, name)
				if info, err := os.Stat(path); err == nil && info.IsDir() {
					agent.Installed = true
					agent.Evidence = path
				}
			}
		}
		for _, name := range location.Bins {
			paths := []string{}
			if path, err := exec.LookPath(name); err == nil {
				paths = append(paths, path)
			}
			for _, folder := range []string{filepath.Join(home, ".local/bin"), filepath.Join(home, ".bun/bin"), "/opt/homebrew/bin", "/usr/local/bin"} {
				paths = append(paths, filepath.Join(folder, name))
			}
			for _, path := range paths {
				if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
					agent.Installed = true
					agent.Evidence = path
					break
				}
			}
		}
		for _, path := range location.Detect {
			path = expandLocation(path, home)
			if _, err := os.Stat(path); err == nil {
				agent.Configured = true
				if agent.Evidence == "" {
					agent.Evidence = path
				}
			}
		}
		// Extensions are installed independently of their host editor.
		if agent.Name == "GitHub Copilot" {
			for _, root := range []string{filepath.Join(home, ".vscode/extensions"), filepath.Join(home, ".vscode-insiders/extensions")} {
				matches, _ := filepath.Glob(filepath.Join(root, "github.copilot*-*"))
				if len(matches) > 0 {
					agent.Installed = true
					agent.Evidence = matches[0]
				}
			}
		}
		byName[agent.Name] = agent
	}
	result := []AgentInfo{}
	for _, agent := range byName {
		result = append(result, agent)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func agentIcon(name string) string {
	icons := map[string]string{
		"Claude Code": "claudecode-color", "Codex": "codex-color", "Cursor": "cursor", "Windsurf": "windsurf", "Devin / Windsurf": "devin", "Devin for Terminal": "devin", "Gemini CLI": "geminicli-color", "GitHub Copilot": "githubcopilot", "VS Code": "vscode-color", "OpenCode": "opencode", "Pi": "pi", "Hermes Agent": "hermes", "OpenClaw": "openclaw", "Qoder": "qoder", "Qoder CN": "qoder", "QoderWork": "qoder", "Qwen Code": "qwen-color", "Kilo Code": "kilocode", "Roo Code": "roocode", "Trae": "trae", "Trae CN": "trae", "Antigravity": "antigravity-color", "Antigravity CLI": "antigravity-color", "Kimi Code CLI": "kimi", "Kiro CLI": "kiro-color", "Cline": "cline", "Goose": "goose", "Zed": "zed", "Grok Build": "grok", "Droid": "factory", "Mistral Vibe": "mistral-color", "MiniMax Code": "minimax-color", "Crush": "crush", "ZCode": "zcode", "Zencoder": "zencoder", "Continue": "continue", "Amp": "amp"}
	if icon, ok := icons[name]; ok {
		return icon
	}
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
