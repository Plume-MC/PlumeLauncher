package runtimes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
)

const adoptiumBaseURL = "https://api.adoptium.net/v3/assets/latest"

// AdoptiumRelease represents the relevant fields from the Adoptium API response.
type AdoptiumRelease struct {
	Binary struct {
		Package struct {
			Link     string `json:"link"`
			Checksum string `json:"checksum"`
			Size     int64  `json:"size"`
			Name     string `json:"name"`
		} `json:"package"`
	} `json:"binary"`
	Version struct {
		Major          int    `json:"major"`
		OpenjdkVersion string `json:"openjdk_version"`
	} `json:"version"`
}

// FetchLatestRelease queries the Adoptium API for the latest Temurin JDK of the given major version.
func FetchLatestRelease(ctx context.Context, major int) (*AdoptiumRelease, error) {
	url := fmt.Sprintf("%s/%d/hotspot?architecture=%s&image_type=jdk&os=%s&vendor=eclipse",
		adoptiumBaseURL, major, AdoptiumArch(), AdoptiumOS())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch adoptium release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("adoptium API returned %d: %s", resp.StatusCode, string(body))
	}

	var releases []AdoptiumRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode adoptium response: %w", err)
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no release found for Java %d", major)
	}

	return &releases[0], nil
}

// AdoptiumOS returns the Adoptium API OS identifier for the current platform.
func AdoptiumOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return "linux"
	}
}

// AdoptiumArch returns the Adoptium API architecture identifier for the current platform.
func AdoptiumArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "aarch64"
	default:
		return "x64"
	}
}
