package bootstrap

// Config holds immutable application state resolved at startup.
type Config struct {
	DataRoot string
	LogDir   string
}

// Initialize resolves the data root and log directory.
// Pass an empty customRoot to use the platform default.
func Initialize(customRoot string) (*Config, error) {
	dataRoot, err := ResolveDataRoot(customRoot)
	if err != nil {
		return nil, err
	}

	logDir, err := ResolveLogDir(dataRoot)
	if err != nil {
		return nil, err
	}

	return &Config{
		DataRoot: dataRoot,
		LogDir:   logDir,
	}, nil
}
