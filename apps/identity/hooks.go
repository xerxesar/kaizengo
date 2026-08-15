package identity

import (
	"strings"

	"kaizengo/packages/sdk-go/engine"
)

func userHooks() engine.Hooks {
	return engine.Hooks{
		BeforeCreate: normalizeUserEmail,
		BeforeUpdate: normalizeUserEmail,
	}
}

func normalizeUserEmail(hc engine.HookContext) error {
	email, ok := hc.Fields["email"].(string)
	if !ok {
		return nil
	}
	hc.Fields["email"] = strings.ToLower(strings.TrimSpace(email))
	return nil
}
