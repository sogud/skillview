package locale

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"testing"
)

func TestSupportedLocaleMatchingAndFallback(t *testing.T) {
	if current := SystemLanguage(); current == "" {
		t.Fatal("empty native system language")
	} else if _, ok := Match(current); !ok {
		t.Fatal("unsupported native language result")
	}
	var cases []struct{ Input, Expected string }
	data, err := os.ReadFile("matching-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, item := range cases {
		got, ok := Match(item.Input)
		if got != item.Expected || ok != (item.Expected != "") {
			t.Errorf("Match(%q) = %q, %v", item.Input, got, ok)
		}
	}
	if Preferred([]string{"sv-SE", "fr-CA", "en"}) != "fr" {
		t.Fatal("preferred list did not skip unsupported language")
	}
	if Preferred(nil) != "en" || Resolve("unknown") != "en" {
		t.Fatal("English fallback missing")
	}
	if T("ar", "Unlisted English fixture") != "Unlisted English fixture" {
		t.Fatal("missing key must fall back to English")
	}
}

func TestAllInterfaceDictionariesAndPlaceholders(t *testing.T) {
	if len(languages) != 18 || len(messages) != 18 {
		t.Fatalf("want 18 languages, metadata=%d dictionaries=%d", len(languages), len(messages))
	}
	base := messages["en"]
	if len(base) != 490 {
		t.Fatalf("source key count = %d", len(base))
	}
	placeholder := regexp.MustCompile(`\{\w+\}`)
	for _, language := range languages {
		dictionary := messages[language.ID]
		if len(dictionary) != len(base) {
			t.Errorf("%s key count %d", language.ID, len(dictionary))
		}
		if (language.Direction == "rtl") != (language.ID == "ar") {
			t.Errorf("unexpected direction for %s", language.ID)
		}
		for key := range base {
			value := dictionary[key]
			if value == "" {
				t.Errorf("%s missing %q", language.ID, key)
				continue
			}
			sourceArgs, translatedArgs := placeholder.FindAllString(key, -1), placeholder.FindAllString(value, -1)
			slices.Sort(sourceArgs)
			slices.Sort(translatedArgs)
			if !slices.Equal(sourceArgs, translatedArgs) {
				t.Errorf("%s placeholders differ: %q", language.ID, key)
			}
			if language.ID == "en" && value != key {
				t.Errorf("English key changed: %q", key)
			}
		}
	}
}
