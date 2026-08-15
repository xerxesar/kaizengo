package appspec

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var capRe = regexp.MustCompile(`^[a-z][a-z0-9_.]*$`)

// Platform capabilities always available without an app provider.
var PlatformCapabilities = []string{
	"platform.i18n",
	"platform.time",
	"platform.config",
}

func validateCapabilityNames(tag string, caps []string) error {
	seen := map[string]struct{}{}
	for _, c := range caps {
		c = strings.TrimSpace(c)
		if c == "" {
			return fmt.Errorf("%s contains empty capability name", tag)
		}
		if !capRe.MatchString(c) {
			return fmt.Errorf("%s has invalid capability %q", tag, c)
		}
		if _, ok := seen[c]; ok {
			return fmt.Errorf("%s duplicate capability %q", tag, c)
		}
		seen[c] = struct{}{}
	}
	return nil
}

// ValidateUses checks every required capability is in the provided set.
func ValidateUses(appName string, uses []string, available map[string]struct{}) error {
	for _, u := range uses {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := available[u]; !ok {
			return fmt.Errorf("app %q uses capability %q but no loaded app provides it", appName, u)
		}
	}
	return nil
}

// MergeCapabilities returns platform built-ins plus app-provided capabilities.
func MergeCapabilities(provided ...[]string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, c := range PlatformCapabilities {
		out[c] = struct{}{}
	}
	for _, list := range provided {
		for _, c := range list {
			out[strings.TrimSpace(c)] = struct{}{}
		}
	}
	return out
}

// ValidateLoadedCapabilities ensures every app's uses are satisfied by loaded provides.
func ValidateLoadedCapabilities(appNames []string) error {
	if len(appNames) == 0 {
		return nil
	}
	var allProvides [][]string
	specs := make(map[string]AppSpec, len(appNames))
	for _, name := range appNames {
		path := "apps/" + name + "/app.yaml"
		if _, err := os.Stat(path); err != nil {
			continue
		}
		spec, err := LoadApp(name)
		if err != nil {
			return fmt.Errorf("app %q spec: %w", name, err)
		}
		specs[name] = spec
		allProvides = append(allProvides, spec.Provides)
	}
	available := MergeCapabilities(allProvides...)
	for name, spec := range specs {
		if err := ValidateUses(name, spec.Uses, available); err != nil {
			return err
		}
	}
	return nil
}
