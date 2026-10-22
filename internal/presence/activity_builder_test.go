package presence

import (
	"strings"
	"testing"
	"time"

	"plumelauncher/internal/instances"
)

func baseDefaults() instances.LauncherDefaults {
	d := instances.DefaultLauncherDefaults()
	d.DiscordPresenceEnabled = true
	return d
}

func TestBuildActivityDisabled(t *testing.T) {
	defaults := baseDefaults()
	defaults.DiscordPresenceEnabled = false

	if got := BuildActivity(1, InstanceInfo{Name: "X"}, defaults, time.Now()); got != nil {
		t.Fatalf("activity = %+v, want nil when disabled", got)
	}
}

func TestBuildActivityDefaultGating(t *testing.T) {
	defaults := baseDefaults() // instance off, version on, loader on, elapsed on
	info := InstanceInfo{Name: "My World", MCVersion: "1.20.1", Loader: "fabric"}

	activity := BuildActivity(4321, info, defaults, time.Unix(1700000000, 0))
	if activity == nil {
		t.Fatal("activity = nil, want set")
	}
	if activity.PID != 4321 {
		t.Errorf("pid = %d, want 4321", activity.PID)
	}
	if strings.Contains(activity.Details, "My World") {
		t.Errorf("details leaked instance name: %q", activity.Details)
	}
	if !strings.Contains(activity.Details, "Minecraft 1.20.1") {
		t.Errorf("details missing version: %q", activity.Details)
	}
	if !strings.Contains(activity.Details, "Fabric") {
		t.Errorf("details missing loader: %q", activity.Details)
	}
	if activity.Timestamps == nil || activity.Timestamps.Start != 1700000000 {
		t.Errorf("timestamps = %+v, want start 1700000000", activity.Timestamps)
	}
	if activity.Assets == nil {
		t.Fatal("assets = nil")
	}
	if activity.Assets.LargeImage != AssetPlumeMark {
		t.Errorf("large image = %q, want %q", activity.Assets.LargeImage, AssetPlumeMark)
	}
	if activity.Assets.SmallImage != AssetFabric {
		t.Errorf("small image = %q, want %q", activity.Assets.SmallImage, AssetFabric)
	}
	if activity.Assets.SmallText != "Fabric" {
		t.Errorf("small text = %q, want Fabric", activity.Assets.SmallText)
	}
}

func TestBuildActivityVanillaHasNoLoaderArt(t *testing.T) {
	defaults := baseDefaults()
	info := InstanceInfo{Name: "Vanilla World", MCVersion: "1.20.1", Loader: "vanilla"}

	activity := BuildActivity(1, info, defaults, time.Now())
	if activity == nil {
		t.Fatal("activity = nil")
	}
	if activity.Assets.LargeImage != AssetPlumeMark {
		t.Errorf("large image = %q, want %q", activity.Assets.LargeImage, AssetPlumeMark)
	}
	if activity.Assets.SmallImage != "" {
		t.Errorf("small image = %q, want empty for vanilla", activity.Assets.SmallImage)
	}
}

func TestBuildActivityLoaderToggleHidesArt(t *testing.T) {
	defaults := baseDefaults()
	defaults.DiscordShowLoader = false

	activity := BuildActivity(1, InstanceInfo{MCVersion: "1.20.1", Loader: "fabric"}, defaults, time.Now())
	if activity.Assets.SmallImage != "" {
		t.Errorf("small image = %q, want empty when loader toggle is off", activity.Assets.SmallImage)
	}
	if activity.Assets.LargeImage != AssetPlumeMark {
		t.Errorf("large image = %q, want plume mark", activity.Assets.LargeImage)
	}
}

func TestBuildActivityInstanceNameOptIn(t *testing.T) {
	defaults := baseDefaults()
	defaults.DiscordShowInstance = true

	activity := BuildActivity(1, InstanceInfo{Name: "My World", MCVersion: "1.20.1"}, defaults, time.Now())
	if activity == nil || !strings.Contains(activity.Details, "My World") {
		t.Fatalf("details = %+v, want instance name when opted in", activity)
	}
}

func TestBuildActivityNoElapsed(t *testing.T) {
	defaults := baseDefaults()
	defaults.DiscordShowElapsed = false

	activity := BuildActivity(1, InstanceInfo{MCVersion: "1.20.1"}, defaults, time.Now())
	if activity.Timestamps != nil {
		t.Errorf("timestamps = %+v, want nil when elapsed off", activity.Timestamps)
	}
}

func TestBuildActivityAllDetailsOff(t *testing.T) {
	defaults := baseDefaults()
	defaults.DiscordShowVersion = false
	defaults.DiscordShowLoader = false

	activity := BuildActivity(1, InstanceInfo{Name: "X", MCVersion: "1.20.1", Loader: "fabric"}, defaults, time.Now())
	if activity == nil {
		t.Fatal("activity = nil")
	}
	if activity.Details != "" {
		t.Errorf("details = %q, want empty when all segments gated off", activity.Details)
	}
}

func TestBuildActivityTruncatesLongName(t *testing.T) {
	defaults := baseDefaults()
	defaults.DiscordShowInstance = true
	long := strings.Repeat("x", 200)

	activity := BuildActivity(1, InstanceInfo{Name: long}, defaults, time.Now())
	if len(activity.Details) > 128 {
		t.Errorf("details length = %d, want <= 128", len(activity.Details))
	}
}

func TestLoaderAssetKey(t *testing.T) {
	cases := map[string]string{"fabric": AssetFabric, "quilt": AssetQuilt, "vanilla": "", "": ""}
	for loader, want := range cases {
		if got := loaderAssetKey(loader); got != want {
			t.Errorf("loaderAssetKey(%q) = %q, want %q", loader, got, want)
		}
	}
}

func TestLoaderLabel(t *testing.T) {
	cases := map[string]string{"fabric": "Fabric", "quilt": "Quilt", "vanilla": "Vanilla", "": ""}
	for loader, want := range cases {
		if got := loaderLabel(loader); got != want {
			t.Errorf("loaderLabel(%q) = %q, want %q", loader, got, want)
		}
	}
}

func TestValidateTemplate(t *testing.T) {
	if err := ValidateTemplate("ok", 16); err != nil {
		t.Errorf("valid template rejected: %v", err)
	}
	if err := ValidateTemplate("bad\x01", 16); err == nil {
		t.Error("control character accepted")
	}
	if err := ValidateTemplate(strings.Repeat("x", 17), 16); err == nil {
		t.Error("over-long template accepted")
	}
}
