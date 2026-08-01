package auth

// Keyring abstracts OS keyring operations.
// Platform backends (Windows Credential Manager, Linux Secret Service)
// will be implemented in Phase 7.
type Keyring interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}
