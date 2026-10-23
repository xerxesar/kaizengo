package extension

// ResetLoadState clears registries that module.Load repopulates (exports + yaml extends).
// Named handlers and init()-time Register() handlers are kept.
// Call before rebuilding the platform for a different database.
func ResetLoadState() {
	clearLoadHandlers()

	viewMu.Lock()
	viewSlots = nil
	components = map[string]string{}
	viewMu.Unlock()

	menuMu.Lock()
	menuContributions = nil
	menuMu.Unlock()
}
