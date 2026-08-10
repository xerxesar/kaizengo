// Package drivers blank-imports optional platform drivers (kernel modules).
// Import this from cmd/server composition — not from individual apps.
package drivers

import (
	// Gregorian is built into platform/time; persian is an optional driver.
	_ "kaizengo/internal/platform/time/persian"
)
