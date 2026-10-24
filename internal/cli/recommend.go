package cli

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"skillview/internal/library"
)

type Runtime struct {
	Platform string `json:"platform"`
	Signal   string `json:"signal"`
}

func CurrentRuntime() Runtime {
	profiles := []struct {
		platform string
		signals  []string
	}{
		{"pi", []string{"PI_CODING_AGENT"}},
		{"claude", []string{"CLAUDE_CODE", "CLAUDECODE", "CLAUDE_PROJECT_DIR"}},
		{"cursor", []string{"CURSOR_TRACE_ID", "CURSOR_AGENT"}},
		{"codex", []string{"CODEX_SANDBOX", "OPENAI_CODEX"}},
		{"gemini", []string{"GEMINI_CLI"}},
		{"hermes", []string{"HERMES_AGENT"}},
		{"opencode", []string{"OPENCODE"}},
	}
	for _, profile := range profiles {
		for _, signal := range profile.signals {
			if os.Getenv(signal) != "" {
				return Runtime{Platform: profile.platform, Signal: signal}
			}
		}
	}
	return Runtime{Platform: "agents", Signal: "default"}
}

type Recommendation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Score       int    `json:"score"`
	Path        string `json:"path"`
}

var expansions = []struct{ triggers, terms string }{
	{"调试 排错 报错 错误 失败 bug debug broken crash", "debug diagnose error recovery investigate test"},
	{"网页 页面 浏览器 截图 点击 表单 browser web screenshot", "browser web opencli playwright screenshot frontend"},
	{"ui ux 界面 样式 设计稿 视觉 布局 figma design", "ui ux frontend visual design figma"},
	{"文档 文章 写作 改写 润色 docs doc writing article", "doc document writing article obsidian pdf docx"},
	{"计划 方案 需求 拆解 plan proposal prd", "plan prd issue triage proposal architecture"},
	{"审查 质量 安全 漏洞 review security", "review quality security lint"},
	{"数据 查询 报表 分析 sql analytics", "data query sql analytics"},
	{"估值 valuation dcf", "valuation dcf intrinsic sotp"},
	{"股票 财报 期权 金融 stock finance earnings options", "stock finance earnings valuation options trading"},
	{"会议 日程 calendar meeting", "calendar meeting"},
	{"技能 skill", "skill skillview find-skills"},
}

func Recommend(skills []library.Skill, query string, top int) []Recommendation {
	query = strings.ToLower(query)
	tokenPattern := regexp.MustCompile(`[a-z0-9][a-z0-9_-]*`)
	tokens := tokenPattern.FindAllString(query, -1)
	result := []Recommendation{}
	for _, skill := range skills {
		name := strings.ToLower(skill.Name)
		description := strings.ToLower(skill.Description)
		score := 0
		if strings.Contains(query, name) {
			score += 60
		}
		for _, token := range tokens {
			if len(token) < 2 {
				continue
			}
			if token == name {
				score += 45
			} else if strings.Contains(name, token) {
				score += 22
			} else if strings.Contains(description, token) {
				score += 8
			}
		}
		for _, expansion := range expansions {
			triggered := false
			for _, trigger := range strings.Fields(expansion.triggers) {
				if tokenPattern.FindString(trigger) == trigger {
					triggered = contains(tokens, trigger)
				} else {
					triggered = strings.Contains(query, trigger)
				}
				if triggered {
					break
				}
			}
			if !triggered {
				continue
			}
			local := 0
			for _, term := range strings.Fields(expansion.terms) {
				if strings.Contains(name, term) {
					local += 16
				} else if strings.Contains(description, term) {
					local += 7
				}
			}
			if local > 35 {
				local = 35
			}
			score += local
		}
		if score > 0 {
			result = append(result, Recommendation{ID: skill.ID, Name: skill.Name, Description: skill.Description, Score: score, Path: skill.Path})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		return result[i].Name < result[j].Name
	})
	unique := []Recommendation{}
	seen := map[string]bool{}
	for _, item := range result {
		key := strings.ToLower(item.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, item)
		if len(unique) == top {
			break
		}
	}
	return unique
}

func Compare(left, right []library.Skill) map[string][]string {
	leftNames := map[string]bool{}
	rightNames := map[string]bool{}
	for _, skill := range left {
		leftNames[skill.Name] = true
	}
	for _, skill := range right {
		rightNames[skill.Name] = true
	}
	result := map[string][]string{"common": {}, "onlyLeft": {}, "onlyRight": {}}
	for name := range leftNames {
		if rightNames[name] {
			result["common"] = append(result["common"], name)
		} else {
			result["onlyLeft"] = append(result["onlyLeft"], name)
		}
	}
	for name := range rightNames {
		if !leftNames[name] {
			result["onlyRight"] = append(result["onlyRight"], name)
		}
	}
	for key := range result {
		sort.Strings(result[key])
	}
	return result
}
