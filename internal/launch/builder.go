package launch

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"plumelauncher/internal/metadata"
)

// Options holds launch configuration for a Minecraft instance.
type Options struct {
	PlayerName      string
	UUID            string
	AccessToken     string
	UserType        string // "offline", "ely.by", "microsoft"
	VersionID       string
	GameDir         string
	ClasspathRoot   string
	AssetsDir       string
	NativesDir      string
	RamMB           int
	Width           int
	Height          int
	Fullscreen      bool
	GPU             string   // "auto", "discrete", "integrated"
	Wrapper         []string // validated argv prefix
	JavaPath        string
	JVMArgs         []string
	AuthlibInjector string // verified authlib-injector JAR for Ely.by accounts
}

// BuildArguments constructs the full Java command line for launching Minecraft.
func BuildArguments(version metadata.VersionDetail, opts Options) ([]string, error) {
	var args []string

	// JVM args
	args = append(args, buildJvmArgs(version, opts)...)

	// Main class
	if version.MainClass == "" {
		return nil, fmt.Errorf("no main class in version metadata")
	}
	args = append(args, version.MainClass)

	// Game args
	args = append(args, buildGameArgs(version, opts)...)

	return args, nil
}

// BuildCommand constructs the executable and argv without invoking a shell.
// A wrapper is an optional executable followed by its fixed argv prefix.
func BuildCommand(javaPath string, javaArgs []string, wrapper []string) (string, []string, error) {
	if javaPath == "" {
		return "", nil, fmt.Errorf("java path is required")
	}
	if len(wrapper) == 0 {
		return javaPath, javaArgs, nil
	}
	for _, arg := range wrapper {
		if arg == "" || strings.ContainsRune(arg, 0) {
			return "", nil, fmt.Errorf("invalid wrapper argument")
		}
	}
	args := append(append([]string{}, wrapper[1:]...), javaPath)
	return wrapper[0], append(args, javaArgs...), nil
}

func buildJvmArgs(version metadata.VersionDetail, opts Options) []string {
	var args []string
	if opts.AuthlibInjector != "" {
		args = append(args, "-javaagent:"+opts.AuthlibInjector)
	}

	// Memory
	if opts.RamMB > 0 {
		args = append(args, fmt.Sprintf("-Xmx%dM", opts.RamMB))
	}

	// Library path
	if opts.NativesDir != "" {
		args = append(args, fmt.Sprintf("-Djava.library.path=%s", opts.NativesDir))
	}

	// LWJGL library path (same as natives for modern MC)
	args = append(args, fmt.Sprintf("-Dorg.lwjgl.librarypath=%s", opts.NativesDir))

	// Language
	args = append(args, "-Duser.language=en", "-Duser.country=US")

	// Launcher branding
	args = append(args, "-Dlauncher.name=PlumeLauncher", "-Dlauncher.version=1.0.0")

	// GPU preference via JVM property (platform-specific)
	if opts.GPU != "" && opts.GPU != "auto" {
		switch runtime.GOOS {
		case "windows":
			// Windows: GPU preference is passed as a system property
			args = append(args, fmt.Sprintf("-Dplume.gpu=%s", opts.GPU))
		case "linux":
			// Linux: could use DRI_PRIME environment variable
			// Applied via env in Launch, not JVM args
		}
	}

	// Classpath
	if version.Arguments != nil && len(version.Arguments.JVM) > 0 {
		// Modern: use arguments.jvm from version metadata
		for _, arg := range version.Arguments.JVM {
			resolved := resolveArgument(arg, version, opts)
			args = append(args, resolved...)
		}
	} else {
		// Fallback: -cp <classpath>
		cp := buildClasspath(version, opts)
		args = append(args, "-cp", cp)
	}

	return args
}

func buildGameArgs(version metadata.VersionDetail, opts Options) []string {
	var args []string

	if version.Arguments != nil && len(version.Arguments.Game) > 0 {
		for _, arg := range version.Arguments.Game {
			resolved := resolveArgument(arg, version, opts)
			args = append(args, resolved...)
		}
	} else if version.MinecraftArguments != nil {
		// Legacy arguments are space-separated and may contain quoted values.
		legacy := string(version.MinecraftArguments)
		for _, part := range splitLegacyArguments(legacy) {
			resolved := replaceVars(part, version, opts)
			args = append(args, resolved)
		}
	}
	// Legacy metadata occasionally omits required launcher arguments after inheritance.
	// Supply only absent values so modern metadata keeps its original command line.
	args = ensureGameOption(args, "--username", opts.PlayerName)
	args = ensureGameOption(args, "--version", version.ID)
	args = ensureGameOption(args, "--gameDir", opts.GameDir)
	args = ensureGameOption(args, "--assetsDir", opts.AssetsDir)
	args = ensureGameOption(args, "--assetIndex", version.AssetIndex.ID)
	args = ensureGameOption(args, "--uuid", opts.UUID)
	args = ensureGameOption(args, "--accessToken", opts.AccessToken)
	args = ensureGameOption(args, "--userType", mapUserType(opts.UserType))

	// Append resolution if not already present and > 0
	hasWidth := false
	for _, a := range args {
		if a == "--width" {
			hasWidth = true
			break
		}
	}
	if !hasWidth && opts.Width > 0 && opts.Height > 0 {
		args = append(args, "--width", fmt.Sprintf("%d", opts.Width), "--height", fmt.Sprintf("%d", opts.Height))
	}

	// Append fullscreen if requested
	hasFullscreen := false
	for _, a := range args {
		if a == "--fullscreen" {
			hasFullscreen = true
			break
		}
	}
	if !hasFullscreen && opts.Fullscreen {
		args = append(args, "--fullscreen")
	}

	return args
}

func ensureGameOption(args []string, option, value string) []string {
	for i, arg := range args {
		if arg == option && i+1 < len(args) && args[i+1] != "" {
			return args
		}
	}
	return append(args, option, value)
}

func splitLegacyArguments(value string) []string {
	var args []string
	var current strings.Builder
	quoted := false
	escaped := false

	for _, r := range value {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			quoted = !quoted
		case r == ' ' && !quoted:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func resolveArgument(arg metadata.Argument, version metadata.VersionDetail, opts Options) []string {
	if arg.StringValue != "" {
		return []string{replaceVars(arg.StringValue, version, opts)}
	}
	if arg.Conditional != nil {
		// Simple rule evaluation: check OS
		if !metadata.ShouldDownload(arg.Conditional.Rules, metadata.CurrentSystem()) {
			return nil
		}
		// Parse value as string or string array
		val := string(arg.Conditional.Value)
		val = strings.Trim(val, `"`)
		if strings.HasPrefix(val, "[") {
			// Array value - split and substitute
			val = strings.Trim(val, `[]`)
			parts := strings.Split(val, ",")
			var result []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, `"`)
				result = append(result, replaceVars(p, version, opts))
			}
			return result
		}
		return []string{replaceVars(val, version, opts)}
	}
	return nil
}

func replaceVars(s string, version metadata.VersionDetail, opts Options) string {
	vars := map[string]string{
		"${auth_player_name}":  opts.PlayerName,
		"${auth_uuid}":         opts.UUID,
		"${auth_access_token}": opts.AccessToken,
		"${user_type}":         mapUserType(opts.UserType),
		"${user_properties}":   "{}",
		"${version_name}":      version.ID,
		"${game_directory}":    opts.GameDir,
		"${assets_root}":       opts.AssetsDir,
		"${assets_index_name}": version.AssetIndex.ID,
		"${auth_xuid}":         "",
		"${clientid}":          "",
		"${version_type}":      "PlumeLauncher",
		"${natives_directory}": opts.NativesDir,
		"${launcher_name}":     "PlumeLauncher",
		"${launcher_version}":  "1.0.0",
		"${classpath}":         buildClasspath(version, opts),
	}

	for k, v := range vars {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

func mapUserType(userType string) string {
	switch userType {
	case "microsoft":
		return "msa"
	case "ely.by":
		return "mojang"
	default:
		return "legacy"
	}
}

func buildClasspath(version metadata.VersionDetail, opts Options) string {
	var paths []string

	root := opts.ClasspathRoot
	if root == "" {
		root = opts.GameDir
	}
	root, _ = filepath.Abs(root)

	// Add client jar
	clientPath := filepath.Join(root, "versions", version.ID, version.ID+".jar")
	paths = append(paths, clientPath)

	// Add allowed libraries
	for _, lib := range version.Libraries {
		if !metadata.ShouldDownload(lib.Rules, metadata.CurrentSystem()) {
			continue
		}
		if lib.Downloads != nil && lib.Downloads.Artifact.URL != "" {
			path := lib.Downloads.Artifact.Path
			if path == "" {
				path = metadata.ResolveMavenPath(lib.Name)
			}
			paths = append(paths, filepath.Join(root, path))
		}
	}

	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	return strings.Join(paths, sep)
}
