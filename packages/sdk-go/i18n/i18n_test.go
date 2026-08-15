package i18n_test

import (
	"testing"

	platform "kaizengo/internal/platform/i18n"
	"kaizengo/packages/sdk-go/i18n"
)

func TestTUsesLoadedCatalog(t *testing.T) {
	platform.Register("en", map[string]string{"sdk_i18n.test.hello": "Hello"})
	platform.Register("fa", map[string]string{"sdk_i18n.test.hello": "سلام"})
	prev := platform.Locale()
	t.Cleanup(func() { platform.SetLocale(prev) })

	platform.SetLocale("en")
	if got := i18n.T("sdk_i18n.test.hello"); got != "Hello" {
		t.Fatalf("en: got %q", got)
	}
	platform.SetLocale("fa")
	if got := i18n.T("sdk_i18n.test.hello"); got != "سلام" {
		t.Fatalf("fa: got %q", got)
	}
	if got := i18n.T("sdk_i18n.test.missing"); got != "sdk_i18n.test.missing" {
		t.Fatalf("missing: got %q", got)
	}
	if err := i18n.Error("sdk_i18n.test.hello"); err.Error() != "سلام" {
		t.Fatalf("Error: got %q", err)
	}
}
