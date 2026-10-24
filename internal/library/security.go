package library

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed security-rules.yaml
var securityRules []byte

type SecurityFinding struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Dimension string `json:"dimension"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type securityRule struct {
	Pattern     string `yaml:"pattern"`
	Level       string `yaml:"level"`
	Description string `yaml:"desc"`
}

type compiledRule struct {
	securityRule
	dimension string
	regex     *regexp.Regexp
	exclude   []string
}

func compileRules() ([]compiledRule, error) {
	var dimensions map[string][]securityRule
	if err := yaml.Unmarshal(securityRules, &dimensions); err != nil {
		return nil, err
	}
	result := []compiledRule{}
	unicodeEscape := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	for dimension, rules := range dimensions {
		if dimension == "trust_signals" {
			continue
		}
		for _, rule := range rules {
			pattern := strings.ReplaceAll(rule.Pattern, `\\`, `\`)
			pattern = strings.ReplaceAll(pattern, `\/`, `/`)
			pattern = unicodeEscape.ReplaceAllStringFunc(pattern, func(value string) string {
				number, _ := strconv.ParseInt(value[2:], 16, 32)
				return string(rune(number))
			})
			exclude := []string{}
			if at := strings.Index(pattern, "(?!"); at >= 0 {
				// The legacy rules use a final negative lookahead only for host allowlists.
				if !strings.HasSuffix(pattern, ")") {
					return nil, fmt.Errorf("Unsupported security rule: %s", rule.Description)
				}
				for _, host := range strings.Split(pattern[at+3:len(pattern)-1], "|") {
					exclude = append(exclude, strings.ReplaceAll(host, `\.`, "."))
				}
				pattern = pattern[:at] + `[^\s]+`
			}
			regex, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				return nil, fmt.Errorf("Invalid security rule: %s", rule.Description)
			}
			result = append(result, compiledRule{securityRule: rule, dimension: dimension, regex: regex, exclude: exclude})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].dimension+result[i].Description < result[j].dimension+result[j].Description
	})
	return result, nil
}

// Findings contain locations and rule descriptions, never matched credential text.
func SecurityScan(path string) ([]SecurityFinding, error) {
	rules, err := compileRules()
	if err != nil {
		return nil, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	onlyFile := ""
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode().IsRegular() {
		onlyFile = filepath.Base(path)
		path = filepath.Dir(path)
	}
	root, err := openAbsoluteDirectory(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	result := []SecurityFinding{}
	count := 0
	var walk func(*os.File, string, int) error
	walk = func(directory *os.File, relative string, depth int) error {
		if depth > 64 || count > 10000 {
			return fmt.Errorf("Security scan limit reached.")
		}
		entries, err := directory.ReadDir(-1)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if relative == "" && onlyFile != "" && entry.Name() != onlyFile {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 || skipDiscoveryDirectory(entry.Name()) {
				continue
			}
			name := filepath.Join(relative, entry.Name())
			if entry.IsDir() {
				child, err := openChild(directory, entry.Name(), true)
				if err != nil {
					return err
				}
				err = walk(child, name, depth+1)
				child.Close()
				if err != nil {
					return err
				}
				continue
			}
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".md", ".txt", ".py", ".sh", ".ts", ".js", ".json", ".yaml", ".yml":
			default:
				continue
			}
			count++
			file, err := openChild(directory, entry.Name(), false)
			if err != nil {
				return err
			}
			data, err := readRegularText(file)
			file.Close()
			if err != nil {
				return fmt.Errorf("Could not inspect text file: %s", name)
			}
			for number, line := range strings.Split(string(data), "\n") {
				for _, rule := range rules {
					matches := rule.regex.FindAllString(line, -1)
					for _, match := range matches {
						allowed := false
						for _, host := range rule.exclude {
							at := strings.Index(match, "://")
							if at >= 0 {
								target := strings.ToLower(match[at+3:])
								if strings.HasPrefix(host, "docs.") {
									allowed = strings.HasPrefix(target, host)
								} else {
									allowed = target == host || strings.HasPrefix(target, host+"/")
								}
							}
							if allowed {
								break
							}
						}
						if !allowed {
							result = append(result, SecurityFinding{Path: name, Line: number + 1, Dimension: rule.dimension, Level: rule.Level, Message: rule.Description})
							break
						}
					}
				}
			}
		}
		return nil
	}
	err = walk(root, "", 0)
	return result, err
}
