package security

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var trustedArtifactHosts = map[string]struct{}{
	"libraries.minecraft.net":          {},
	"piston-data.mojang.com":           {},
	"piston-meta.mojang.com":           {},
	"launchermeta.mojang.com":          {},
	"launcher.mojang.com":              {},
	"resources.download.minecraft.net": {},
	"s3.amazonaws.com":                 {},
	"maven.fabricmc.net":               {},
	"meta.fabricmc.net":                {},
	"maven.quiltmc.org":                {},
	"meta.quiltmc.org":                 {},
	"github.com":                       {},
}

// ValidateArtifactURL restricts downloads to the upstream artifact hosts.
func ValidateArtifactURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("untrusted artifact URL")
	}
	if _, ok := trustedArtifactHosts[strings.ToLower(parsed.Hostname())]; !ok {
		return fmt.Errorf("untrusted artifact host: %s", parsed.Hostname())
	}
	return nil
}

// ResolveUnderRoot returns a path that remains inside root, including when an
// existing parent or target is a symlink.
func ResolveUnderRoot(root, relative string) (string, error) {
	if root == "" || relative == "" || strings.ContainsRune(relative, 0) {
		return "", fmt.Errorf("invalid path")
	}
	if filepath.IsAbs(relative) || path.IsAbs(relative) || filepath.VolumeName(relative) != "" || strings.HasPrefix(relative, `\`) {
		return "", fmt.Errorf("absolute path not allowed: %s", relative)
	}
	if strings.Contains(relative, `\`) || path.Clean(relative) != relative || relative == "." || strings.HasPrefix(relative, "../") || strings.Contains(relative, "/../") {
		return "", fmt.Errorf("path traversal not allowed: %s", relative)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootReal := rootAbs
	if resolved, resolveErr := filepath.EvalSymlinks(rootAbs); resolveErr == nil {
		rootReal = resolved
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(relative))
	if err := ensureLexicalContainment(rootAbs, target); err != nil {
		return "", err
	}

	existing := target
	var remainder []string
	for {
		if _, statErr := os.Lstat(existing); statErr == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			break
		}
		remainder = append([]string{filepath.Base(existing)}, remainder...)
		existing = parent
	}
	if resolved, resolveErr := filepath.EvalSymlinks(existing); resolveErr == nil {
		existing = resolved
	}
	resolvedTarget := existing
	for _, part := range remainder {
		resolvedTarget = filepath.Join(resolvedTarget, part)
	}
	if err := ensureLexicalContainment(rootReal, resolvedTarget); err != nil {
		return "", fmt.Errorf("path escapes root: %w", err)
	}
	return target, nil
}

func ensureLexicalContainment(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("path escapes root")
	}
	return nil
}
