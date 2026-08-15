// Package drivers blank-imports optional platform drivers (kernel modules).
// Import this from cmd/server composition — not from individual apps.
package drivers

import (
	// Optional calendar drivers. Locale packs live under platform/i18n/locale/*.po
	// and apps/*/locale/*.po.
	_ "kaizengo/internal/platform/time/persian"
)
