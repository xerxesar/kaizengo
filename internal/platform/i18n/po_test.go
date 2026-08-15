package i18n_test

import (
	"testing"

	"kaizengo/internal/platform/i18n"
)

func TestParsePO(t *testing.T) {
	src := `
msgid ""
msgstr ""
"Language: en\n"

msgid "clock.title"
msgstr "Clock"

msgid "clock.subtitle"
msgstr "Local time."

#, fuzzy
msgid "ignored"
msgstr "nope"
`
	msgs, err := i18n.ParsePO([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if msgs["clock.title"] != "Clock" {
		t.Fatalf("got %q", msgs["clock.title"])
	}
	if msgs["clock.subtitle"] != "Local time." {
		t.Fatalf("got %q", msgs["clock.subtitle"])
	}
	if _, ok := msgs["ignored"]; ok {
		t.Fatal("fuzzy entry should be skipped")
	}
	if _, ok := msgs[""]; ok {
		t.Fatal("header should be skipped")
	}
}
