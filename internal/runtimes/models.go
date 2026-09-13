package runtimes

import "time"

// SupportedMajors lists the Adoptium/Temurin JDK versions available for download.
var SupportedMajors = []int{8, 17, 21, 25}

// ManagedRuntime represents a downloaded and extracted JDK runtime.
type ManagedRuntime struct {
	Major     int       `json:"major"`
	Version   string    `json:"version"`
	Path      string    `json:"path"`
	Installed bool      `json:"installed"`
	Size      int64     `json:"size,omitempty"`
}

// JavaDownloadState tracks the state of an active or completed JDK download.
type JavaDownloadState struct {
	Major      int     `json:"major"`
	Status     string  `json:"status"` // "idle", "downloading", "extracting", "completed", "failed", "cancelled"
	BytesRead  int64   `json:"bytesRead"`
	TotalBytes int64   `json:"totalBytes"`
	Speed      float64 `json:"speed"`
	ETA        float64 `json:"eta"`
	Error      string  `json:"error,omitempty"`
}

// JavaDownloadProgressEvent is emitted via Wails events during JDK downloads.
type JavaDownloadProgressEvent struct {
	Major      int     `json:"major"`
	Status     string  `json:"status"`
	BytesRead  int64   `json:"bytesRead"`
	TotalBytes int64   `json:"totalBytes"`
	Speed      float64 `json:"speed"`
	ETA        float64 `json:"eta"`
	Error      string  `json:"error,omitempty"`
}

// manifest is written to DataRoot/runtimes/java/{major}/manifest.json after extraction.
type manifest struct {
	Major       int       `json:"major"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installedAt"`
}
