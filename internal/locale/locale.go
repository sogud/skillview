package locale

import (
	_ "embed"
	"encoding/json"
)

//go:embed messages.json
var messageData []byte

var messages = func() map[string]map[string]string {
	var result map[string]map[string]string
	if err := json.Unmarshal(messageData, &result); err != nil {
		panic(err)
	}
	return result
}()

func Resolve(preference string) string {
	if preference == "system" || preference == "" {
		return SystemLanguage()
	}
	if preference == "zh-CN" {
		return "zh-CN"
	}
	return "en"
}

func T(language, text string) string {
	if translated := messages[Resolve(language)][text]; translated != "" {
		return translated
	}
	return text
}
