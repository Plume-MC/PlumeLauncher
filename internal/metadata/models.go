package metadata

import (
	"encoding/json"
	"fmt"
)

// VersionManifest is the top-level response from the Mojang version manifest API.
type VersionManifest struct {
	Latest   Latest         `json:"latest"`
	Versions []VersionEntry `json:"versions"`
}

// Latest holds the latest release and snapshot version IDs.
type Latest struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

// VersionEntry is a single entry in the version manifest.
type VersionEntry struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Time            string `json:"time"`
	ReleaseTime     string `json:"releaseTime"`
	SHA1            string `json:"sha1"`
	ComplianceLevel *int   `json:"complianceLevel,omitempty"`
}

// VersionDetail is the full metadata for a single Minecraft version.
type VersionDetail struct {
	Arguments              *Arguments              `json:"arguments,omitempty"`
	AssetIndex             AssetIndex              `json:"assetIndex"`
	Assets                 string                  `json:"assets"`
	ComplianceLevel        *int                    `json:"complianceLevel,omitempty"`
	Downloads              Downloads               `json:"downloads"`
	ID                     string                  `json:"id"`
	Jar                    string                  `json:"jar,omitempty"`
	JavaVersion            JavaVersion             `json:"javaVersion"`
	Libraries              []Library               `json:"libraries"`
	Logging                *Logging                `json:"logging,omitempty"`
	MainClass              string                  `json:"mainClass"`
	MinecraftArguments     json.RawMessage         `json:"minecraftArguments,omitempty"`
	MinimumLauncherVersion int                     `json:"minimumLauncherVersion"`
	InheritsFrom           string                  `json:"inheritsFrom,omitempty"`
	LegacyResources        map[string]DownloadInfo `json:"legacyResources,omitempty"`
	ReleaseTime            string                  `json:"releaseTime"`
	Time                   string                  `json:"time"`
	Type                   string                  `json:"type"`
}

// Arguments holds game and JVM arguments.
type Arguments struct {
	Game []Argument `json:"game"`
	JVM  []Argument `json:"jvm"`
}

// Argument is either a plain string or a conditional object with rules and values.
type Argument struct {
	StringValue string
	Conditional *ConditionalArgument
}

// ConditionalArgument represents a conditional argument with rules.
type ConditionalArgument struct {
	Rules []Rule          `json:"rules"`
	Value json.RawMessage `json:"value"`
}

// UnmarshalJSON implements json.Unmarshaler for Argument.
func (a *Argument) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.StringValue = s
		return nil
	}

	// Try conditional object
	var c ConditionalArgument
	if err := json.Unmarshal(data, &c); err == nil {
		a.Conditional = &c
		return nil
	}

	return fmt.Errorf("argument: cannot unmarshal %s", string(data))
}

// MarshalJSON implements json.Marshaler for Argument.
func (a Argument) MarshalJSON() ([]byte, error) {
	if a.StringValue != "" {
		return json.Marshal(a.StringValue)
	}
	if a.Conditional != nil {
		return json.Marshal(a.Conditional)
	}
	return json.Marshal(nil)
}

// Rule defines an allow/disallow condition for libraries, natives, or arguments.
type Rule struct {
	Action   string       `json:"action"`
	OS       *OSRule      `json:"os,omitempty"`
	Features *FeatureRule `json:"features,omitempty"`
}

// OSRule matches against operating system properties.
type OSRule struct {
	Name    string `json:"name,omitempty"`
	Arch    string `json:"arch,omitempty"`
	Version string `json:"version,omitempty"`
}

// FeatureRule matches against feature flags.
type FeatureRule struct {
	IsDemoUser          *bool `json:"is_demo_user,omitempty"`
	HasCustomResolution *bool `json:"has_custom_resolution,omitempty"`
}

// AssetIndex points to the asset index JSON file.
type AssetIndex struct {
	ID        string                 `json:"id"`
	SHA1      string                 `json:"sha1"`
	Size      int64                  `json:"size"`
	TotalSize int64                  `json:"totalSize"`
	URL       string                 `json:"url"`
	Objects   map[string]AssetObject `json:"objects,omitempty"`
}

// AssetObject identifies one content-addressed asset file.
type AssetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// Downloads holds download info for client, server, and mappings.
type Downloads struct {
	Client         *DownloadInfo `json:"client,omitempty"`
	Server         *DownloadInfo `json:"server,omitempty"`
	ClientMappings *DownloadInfo `json:"client_mappings,omitempty"`
	ServerMappings *DownloadInfo `json:"server_mappings,omitempty"`
}

// DownloadInfo describes a single downloadable artifact.
type DownloadInfo struct {
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
	Path string `json:"path,omitempty"`
}

// JavaVersion specifies which Java version is required.
type JavaVersion struct {
	Component    string `json:"component"`
	MajorVersion int    `json:"majorVersion"`
}

// Library is a Minecraft dependency library.
type Library struct {
	Downloads   *LibraryDownloads       `json:"downloads,omitempty"`
	Name        string                  `json:"name"`
	URL         string                  `json:"url,omitempty"`
	Natives     map[string]string       `json:"natives,omitempty"`
	Extract     *ExtractRule            `json:"extract,omitempty"`
	Rules       []Rule                  `json:"rules,omitempty"`
	Classifiers map[string]DownloadInfo `json:"classifiers,omitempty"`
}

// LibraryDownloads holds the main artifact and optional classifiers.
type LibraryDownloads struct {
	Artifact    DownloadInfo            `json:"artifact"`
	Classifiers map[string]DownloadInfo `json:"classifiers,omitempty"`
}

// ExtractRule defines which files to exclude during native extraction.
type ExtractRule struct {
	Exclude []string `json:"exclude"`
}

// Logging defines client-side logging configuration.
type Logging struct {
	Client *LoggingConfig `json:"client,omitempty"`
}

// LoggingConfig describes the logging configuration file.
type LoggingConfig struct {
	Argument string      `json:"argument"`
	File     LogFileInfo `json:"file"`
	Type     string      `json:"type"`
}

// LogFileInfo is metadata about the logging config file.
type LogFileInfo struct {
	ID   string `json:"id"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}
