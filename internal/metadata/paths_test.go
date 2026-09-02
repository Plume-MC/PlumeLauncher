package metadata_test

import (
	"testing"

	"plumelauncher/internal/metadata"
)

func TestResolveMavenPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard", "com.mojang:logging:1.0.0", "com/mojang/logging/1.0.0/logging-1.0.0.jar"},
		{"guava", "com.google.guava:guava:31.0.1-jre", "com/google/guava/guava/31.0.1-jre/guava-31.0.1-jre.jar"},
		{"netty", "io.netty:netty-all:4.1.68.Final", "io/netty/netty-all/4.1.68.Final/netty-all-4.1.68.Final.jar"},
		{"invalid", "bad", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadata.ResolveMavenPath(tt.input)
			if got != tt.want {
				t.Errorf("ResolveMavenPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveNativePathWithClassifiers(t *testing.T) {
	library := metadata.Library{
		Natives: map[string]string{
			"windows": "natives-windows",
			"linux":   "natives-linux",
		},
		Downloads: &metadata.LibraryDownloads{
			Classifiers: map[string]metadata.DownloadInfo{
				"natives-windows": {Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar"},
				"natives-linux":   {Path: "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-linux.jar"},
			},
		},
	}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	path, ok := metadata.ResolveNativePath(library, sys)
	if !ok {
		t.Fatal("expected ok")
	}
	if path != "org/lwjgl/lwjgl/3.2.2/lwjgl-3.2.2-natives-windows.jar" {
		t.Errorf("path = %q", path)
	}
}

func TestResolveNativePathTopLevelClassifiers(t *testing.T) {
	library := metadata.Library{
		Natives: map[string]string{
			"linux": "natives-linux",
		},
		Classifiers: map[string]metadata.DownloadInfo{
			"natives-linux": {Path: "com/mojang/text2speech/1.12.4/text2speech-1.12.4-natives-linux.jar"},
		},
	}
	sys := metadata.SystemInfo{OS: "linux", Arch: "x64"}
	path, ok := metadata.ResolveNativePath(library, sys)
	if !ok {
		t.Fatal("expected ok")
	}
	if path != "com/mojang/text2speech/1.12.4/text2speech-1.12.4-natives-linux.jar" {
		t.Errorf("path = %q", path)
	}
}

func TestResolveNativePathNoNatives(t *testing.T) {
	library := metadata.Library{Name: "com.mojang:logging:1.0.0"}
	sys := metadata.SystemInfo{OS: "windows", Arch: "x64"}
	_, ok := metadata.ResolveNativePath(library, sys)
	if ok {
		t.Error("expected false for library without natives")
	}
}
