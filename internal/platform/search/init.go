package search

// Default backend is in-memory so dev search works without Typesense.
// Addon apps (apps/typesense) call Register to replace it.
func init() {
	Register(newMemoryBackend())
	_ = LoadConfig()
}
