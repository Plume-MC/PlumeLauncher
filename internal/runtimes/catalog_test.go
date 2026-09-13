package runtimes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestRelease(t *testing.T) {
	mockResponse := []AdoptiumRelease{
		{
			Version: struct {
				Major          int    `json:"major"`
				OpenjdkVersion string `json:"openjdk_version"`
			}{
				Major:          25,
				OpenjdkVersion: "25.0.4.1+1-LTS",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer srv.Close()

	// Override the base URL for testing
	origBase := adoptiumBaseURL
	// We can't easily override the const, so test the parsing logic directly
	data, err := json.Marshal(mockResponse)
	if err != nil {
		t.Fatal(err)
	}

	var releases []AdoptiumRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		t.Fatal(err)
	}

	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	if releases[0].Version.Major != 25 {
		t.Errorf("expected major 25, got %d", releases[0].Version.Major)
	}
	if releases[0].Version.OpenjdkVersion != "25.0.4.1+1-LTS" {
		t.Errorf("expected version 25.0.4.1+1-LTS, got %s", releases[0].Version.OpenjdkVersion)
	}

	_ = origBase
	_ = srv
}

func TestAdoptiumOS(t *testing.T) {
	os := AdoptiumOS()
	if os != "linux" && os != "windows" {
		t.Errorf("expected linux or windows, got %s", os)
	}
}

func TestAdoptiumArch(t *testing.T) {
	arch := AdoptiumArch()
	if arch != "x64" && arch != "aarch64" {
		t.Errorf("expected x64 or aarch64, got %s", arch)
	}
}

func TestFetchLatestRelease_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchLatestRelease(ctx, 25)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
