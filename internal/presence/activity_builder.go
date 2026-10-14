package presence

import (
	"fmt"
	"strings"
	"time"

	"plumelauncher/internal/instances"
)

// Asset keys must match names uploaded to the Discord application's
// Art Assets section exactly; unknown keys render as empty boxes.
const (
	AssetLargePlume = "plume_mark"
)

// GameStartTime anchors the elapsed clock to the process start.
func GameStartTime(startedAt time.Time) *Timestamps {
	return &Timestamps{Start: startedAt.Unix()}
}

// BuildActivity constructs the SET_ACTIVITY payload from instance facts
// and the user's privacy toggles. Gating happens here, not at call sites,
// so disabled fields can never leak into the wire payload.
//
// Layout: details = "{instance} — Minecraft {version} · {loader}"
// (omitting gated segments), state = "Playing with Plume",
// small_image = Plume mark, large_image = loader art when available.
func BuildActivity(pid int, info InstanceInfo, defaults instances.LauncherDefaults, startedAt time.Time) *Activity {
	if !defaults.DiscordPresenceEnabled {
		return nil
	}

	details := buildDetails(info, defaults)

	activity := &Activity{
		PID:     pid,
		Details: details,
		State:   "Playing with Plume",
		Assets: &Assets{
			SmallImage: AssetLargePlume,
			SmallText:  "Plume Launcher",
		},
	}
	if defaults.DiscordShowElapsed {
		activity.Timestamps = GameStartTime(startedAt)
	}
	if defaults.DiscordShowLoader && info.Loader != "" {
		activity.Assets.LargeImage = loaderAssetKey(info.Loader)
		activity.Assets.LargeText = strings.Title(info.Loader) // nolint:staticcheck // display label, not code
	}
	return activity
}

// InstanceInfo carries the minimal instance facts the builder needs.
type InstanceInfo struct {
	Name      string
	MCVersion string
	Loader    string // "" counts as vanilla
}

// buildDetails joins the gated segments; every segment is dropped when
// its toggle is off, and the whole string is empty when all are off.
func buildDetails(info InstanceInfo, defaults instances.LauncherDefaults) string {
	segments := make([]string, 0, 3)
	if defaults.DiscordShowInstance {
		segments = append(segments, info.Name)
	}
	if defaults.DiscordShowVersion && info.MCVersion != "" {
		segments = append(segments, "Minecraft "+info.MCVersion)
	}
	if defaults.DiscordShowLoader && info.Loader != "" {
		segments = append(segments, info.Loader)
	}
	details := strings.Join(segments, " — ")
	return truncate(details, 128)
}

// truncate caps a payload string at Discord's 128-byte field limit.
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max])
}

// loaderAssetKey maps a loader type to its uploaded asset key.
func loaderAssetKey(loader string) string {
	switch loader {
	case "fabric":
		return "fabric"
	case "quilt":
		return "quilt"
	default:
		return ""
	}
}

// ValidateTemplate rejects free-form templates with control characters or
// over-long values; used by UpdateSettings before persisting.
func ValidateTemplate(value string, max int) error {
	for _, r := range value {
		if r < 0x20 {
			return fmt.Errorf("template contains control characters")
		}
	}
	if len(value) > max {
		return fmt.Errorf("template exceeds %d bytes", max)
	}
	return nil
}
