package metadata

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// ArtifactRole identifies the type of artifact.
type ArtifactRole string

const (
	RoleClient         ArtifactRole = "client"
	RoleLibrary        ArtifactRole = "library"
	RoleNative         ArtifactRole = "native"
	RoleAsset          ArtifactRole = "asset"
	RoleLegacyResource ArtifactRole = "legacy_resource"
)

// Artifact represents a single required file for an instance.
type Artifact struct {
	Role     ArtifactRole `json:"role"`
	URL      string       `json:"url"`
	Path     string       `json:"path"`
	Size     int64        `json:"size"`
	Sha1     string       `json:"sha1"`
	Rules    []Rule       `json:"rules,omitempty"`
	Required bool         `json:"required"`
}

// ArtifactPlan is the complete set of artifacts needed for a version.
type ArtifactPlan struct {
	VersionID string     `json:"versionId"`
	Artifacts []Artifact `json:"artifacts"`
}

// ResolvePlan produces an ArtifactPlan from a merged version detail.
func ResolvePlan(detail VersionDetail, sys SystemInfo) *ArtifactPlan {
	plan := &ArtifactPlan{
		VersionID: detail.ID,
	}

	// 1. Client JAR
	plan.addClient(detail)

	// 2. Libraries (filtered by rules)
	plan.addLibraries(detail, sys)

	// 3. Native classifiers
	plan.addNatives(detail, sys)

	// 4. Asset index
	plan.addAssetIndex(detail)

	// 5. Asset objects loaded from the asset index
	plan.addAssetObjects(detail)

	// 6. Normalized legacy resources, when supplied by an old metadata adapter
	plan.addLegacyResources(detail)

	plan.dedupeArtifacts()

	return plan
}

// dedupeArtifacts ensures one downloader task owns each destination path.
// Inherited metadata can describe the same Maven artifact more than once.
func (p *ArtifactPlan) dedupeArtifacts() {
	seen := make(map[string]struct{}, len(p.Artifacts))
	unique := p.Artifacts[:0]
	for _, artifact := range p.Artifacts {
		if artifact.Path == "" {
			unique = append(unique, artifact)
			continue
		}
		if _, ok := seen[artifact.Path]; ok {
			continue
		}
		seen[artifact.Path] = struct{}{}
		unique = append(unique, artifact)
	}
	p.Artifacts = unique
}

func (p *ArtifactPlan) addClient(detail VersionDetail) {
	if detail.Downloads.Client == nil {
		return
	}
	dl := detail.Downloads.Client
	path := dl.Path
	if path == "" {
		versionID := detail.ID
		if detail.Jar != "" {
			versionID = detail.Jar
		}
		path = "versions/" + versionID + "/" + versionID + ".jar"
	}
	if err := ValidatePath(path); err != nil {
		return // skip invalid paths
	}
	p.Artifacts = append(p.Artifacts, Artifact{
		Role:     RoleClient,
		URL:      dl.URL,
		Path:     path,
		Size:     dl.Size,
		Sha1:     dl.SHA1,
		Required: true,
	})
}

func (p *ArtifactPlan) addLibraries(detail VersionDetail, sys SystemInfo) {
	for _, lib := range detail.Libraries {
		if !ShouldDownload(lib.Rules, sys) {
			continue
		}
		if lib.Downloads == nil {
			continue
		}
		dl := lib.Downloads.Artifact
		if dl.URL == "" {
			continue
		}
		path := dl.Path
		if path == "" {
			path = ResolveMavenPath(lib.Name)
		}
		if err := ValidatePath(path); err != nil {
			continue
		}
		p.Artifacts = append(p.Artifacts, Artifact{
			Role:     RoleLibrary,
			URL:      dl.URL,
			Path:     path,
			Size:     dl.Size,
			Sha1:     dl.SHA1,
			Rules:    lib.Rules,
			Required: true,
		})
	}
}

func (p *ArtifactPlan) addNatives(detail VersionDetail, sys SystemInfo) {
	for _, lib := range detail.Libraries {
		if !ShouldDownload(lib.Rules, sys) {
			continue
		}
		nativePath, ok := ResolveNativePath(lib, sys)
		if !ok {
			continue
		}

		// Find the download info for this native classifier
		var dl DownloadInfo
		if lib.Downloads != nil && lib.Downloads.Classifiers != nil {
			classifierKey, _ := GetNativeClassifier(lib, sys)
			if d, ok := lib.Downloads.Classifiers[classifierKey]; ok {
				dl = d
			}
		}
		if lib.Classifiers != nil {
			classifierKey, _ := GetNativeClassifier(lib, sys)
			if d, ok := lib.Classifiers[classifierKey]; ok {
				dl = d
			}
		}

		if dl.URL == "" {
			continue
		}

		if err := ValidatePath(nativePath); err != nil {
			continue
		}

		p.Artifacts = append(p.Artifacts, Artifact{
			Role:     RoleNative,
			URL:      dl.URL,
			Path:     nativePath,
			Size:     dl.Size,
			Sha1:     dl.SHA1,
			Rules:    lib.Rules,
			Required: true,
		})
	}
}

func (p *ArtifactPlan) addAssetIndex(detail VersionDetail) {
	if detail.AssetIndex.URL == "" {
		return
	}
	path := "assets/indexes/" + detail.AssetIndex.ID + ".json"
	p.Artifacts = append(p.Artifacts, Artifact{
		Role:     RoleAsset,
		URL:      detail.AssetIndex.URL,
		Path:     path,
		Size:     detail.AssetIndex.Size,
		Sha1:     detail.AssetIndex.SHA1,
		Required: true,
	})
}

func (p *ArtifactPlan) addAssetObjects(detail VersionDetail) {
	for _, object := range detail.AssetIndex.Objects {
		if len(object.Hash) != 40 || !isHexHash(object.Hash) || object.Size < 0 {
			continue
		}
		p.Artifacts = append(p.Artifacts, Artifact{
			Role:     RoleAsset,
			URL:      "https://resources.download.minecraft.net/" + object.Hash[:2] + "/" + object.Hash,
			Path:     "assets/objects/" + object.Hash[:2] + "/" + object.Hash,
			Size:     object.Size,
			Sha1:     object.Hash,
			Required: true,
		})
	}
}

func (p *ArtifactPlan) addLegacyResources(detail VersionDetail) {
	for resourcePath, resource := range detail.LegacyResources {
		if err := ValidatePath(resourcePath); err != nil || resource.URL == "" {
			continue
		}
		p.Artifacts = append(p.Artifacts, Artifact{
			Role:     RoleLegacyResource,
			URL:      resource.URL,
			Path:     resourcePath,
			Size:     resource.Size,
			Sha1:     resource.SHA1,
			Required: true,
		})
	}
}

func isHexHash(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

// ValidatePath checks that a path has no traversal or absolute components.
func ValidatePath(value string) error {
	if value == "" {
		return nil
	}
	if path.IsAbs(value) || filepath.VolumeName(value) != "" || value[0] == '\\' {
		return fmt.Errorf("absolute path not allowed: %s", value)
	}
	if strings.Contains(value, "\\") || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") {
		return fmt.Errorf("traversal not allowed: %s", value)
	}
	return nil
}

// PlanSummary returns a human-readable summary of the artifact plan.
func PlanSummary(plan *ArtifactPlan) string {
	counts := map[ArtifactRole]int{}
	for _, a := range plan.Artifacts {
		counts[a.Role]++
	}
	return fmt.Sprintf("Plan %s: client=%d libraries=%d natives=%d assets=%d legacy=%d total=%d",
		plan.VersionID,
		counts[RoleClient],
		counts[RoleLibrary],
		counts[RoleNative],
		counts[RoleAsset],
		counts[RoleLegacyResource],
		len(plan.Artifacts),
	)
}
