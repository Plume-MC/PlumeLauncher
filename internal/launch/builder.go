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
	PlayerName    string
	UUID          string
	AccessToken   string
	UserType      string // "offline", "ely.by", "microsoft"
	VersionID     string
	GameDir       string
	AssetsDir     string
	NativesDir    string
	RamMB         int
	Width         int
	Height        int
	Fullscreen    bool
	GPU           string // "auto", "discrete", "integrated"
	Wrapper       []string // validated argv prefix
	JavaPath      string
	JVMArgs       []string
}

// BuildArguments constructs the full Java command line for launching Minecraft.
func BuildArguments(version metadata.VersionDetail, opts Options) ([]string, error) {
	var args []string

	// Wrapper prefix (validated argv, not shell)
	args = append(args, opts.Wrapper...)

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

func buildJvmArgs(version metadata.VersionDetail, opts Options) []string {
	var args []string

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
	} else if version.MainClassArguments != nil {
		// Legacy: parse space-separated args, each may be quoted
		legacy := string(version.MainClassArguments)
		for _, part := range strings.Fields(legacy) {
			part = strings.Trim(part, `"`)
			resolved := replaceVars(part, version, opts)
			args = append(args, resolved)
		}
	}

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
		"${auth_player_name}":    opts.PlayerName,
		"${auth_uuid}":           opts.UUID,
		"${auth_access_token}":   opts.AccessToken,
		"${user_type}":           mapUserType(opts.UserType),
		"${user_properties}":     "{}",
		"${version_name}":        version.ID,
		"${game_directory}":      opts.GameDir,
		"${assets_root}":         opts.AssetsDir,
		"${assets_index_name}":   version.AssetIndex.ID,
		"${auth_xuid}":           "",
		"${clientid}":            "",
		"${version_type}":        "PlumeLauncher",
		"${natives_directory}":   opts.NativesDir,
		"${launcher_name}":       "PlumeLauncher",
		"${launcher_version}":    "1.0.0",
		"${classpath}":           buildClasspath(version, opts),
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

	// Add client jar
	clientPath := filepath.Join(opts.GameDir, "versions", version.ID, version.ID+".jar")
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
			paths = append(paths, filepath.Join(opts.GameDir, path))
		}
	}

	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	return strings.Join(paths, sep)
}
