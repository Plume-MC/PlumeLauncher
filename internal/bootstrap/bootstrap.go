package bootstrap

import "os"

// Config holds the canonical launcher data root resolved at startup.
type Config struct {
	DataRoot string // accounts, config, logs, lock, instances, assets, versions, cache
}

// Initialize resolves the canonical launcher data root.
// Pass an empty customRoot to use env, bootstrap.json, or the platform default.
func Initialize(customRoot string) (*Config, error) {
	appRoot := defaultAppRoot()
	if err := os.MkdirAll(appRoot, 0o755); err != nil {
		return nil, err
	}

	gameRoot, err := ResolveGameRoot(customRoot)
	if err != nil {
		return nil, err
	}

	return &Config{DataRoot: gameRoot}, nil
}
