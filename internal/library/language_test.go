package library

import (
	"skillview/internal/locale"
	"testing"
)

func TestInterfaceLanguagePersistsAcrossReopen(t *testing.T) {
	directory := t.TempDir()
	library, err := New(directory, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, language := range locale.Languages() {
		if err := library.SetLanguage(language.ID); err != nil {
			t.Fatal(language.ID, err)
		}
		reopened, err := New(directory, nil)
		if err != nil {
			t.Fatal(err)
		}
		if reopened.Settings().Language != language.ID {
			t.Fatalf("%s did not persist", language.ID)
		}
	}
	if err := library.SetLanguage("zh-Hant-HK"); err != nil {
		t.Fatal(err)
	}
	if library.Settings().Language != "zh-TW" {
		t.Fatal("regional alias was not canonicalized")
	}
	if library.SetLanguage("unsupported") == nil || library.Settings().Language != "zh-TW" {
		t.Fatal("invalid language changed preference")
	}
	if err := library.SetLanguage("system"); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(directory, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Settings().Language != "system" {
		t.Fatal("system preference did not persist")
	}
}
