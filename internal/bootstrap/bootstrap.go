package bootstrap

import "os"

// Config holds immutable application state resolved at startup.
type Config struct {
	AppRoot  string // fixed: accounts, config, logs, lock, bootstrap.json
	GameRoot string // customizable: instances, assets, versions, cache
	LogDir   string // always AppRoot/logs
}

// Initialize resolves app/game roots and log directory.
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

	logDir, err := ResolveLogDir(appRoot)
	if err != nil {
		return nil, err
	}

	return &Config{
		AppRoot:  appRoot,
		GameRoot: gameRoot,
		LogDir:   logDir,
	}, nil
}
