package locale

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed messages.json
var messageData []byte

//go:embed languages.json
var languageData []byte

type Language struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Mac       string `json:"mac"`
	Direction string `json:"direction"`
}

var messages = func() map[string]map[string]string {
	var result map[string]map[string]string
	if err := json.Unmarshal(messageData, &result); err != nil {
		panic(err)
	}
	return result
}()
var languages = func() []Language {
	var result []Language
	if err := json.Unmarshal(languageData, &result); err != nil {
		panic(err)
	}
	return result
}()

func Languages() []Language { return append([]Language(nil), languages...) }

// Match accepts supported language tags and regional variants. Explicit Chinese
// scripts take precedence over regions. Portuguese variants use Brazilian text.
func Match(tag string) (string, bool) {
	tag = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"))
	tag = strings.Split(strings.Split(tag, ".")[0], "@")[0]
	parts := strings.Split(tag, "-")
	candidate := parts[0]
	if candidate == "zh" {
		candidate = "zh-CN"
		has := func(value string) bool {
			for _, part := range parts[1:] {
				if part == value {
					return true
				}
			}
			return false
		}
		if has("hant") || (!has("hans") && (has("tw") || has("hk") || has("mo"))) {
			candidate = "zh-TW"
		}
	} else if candidate == "pt" {
		candidate = "pt-BR"
	}
	for _, language := range languages {
		if language.ID == candidate {
			return candidate, true
		}
	}
	return "", false
}

func Preferred(tags []string) string {
	for _, tag := range tags {
		if language, ok := Match(tag); ok {
			return language
		}
	}
	return "en"
}
func Resolve(preference string) string {
	if preference == "system" || preference == "" {
		return SystemLanguage()
	}
	return Preferred([]string{preference})
}
func T(language, text string) string {
	if translated := messages[Resolve(language)][text]; translated != "" {
		return translated
	}
	return text
}
