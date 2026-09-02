package services

import (
	"context"
	"fmt"

	"plumelauncher/internal/instances"
	"plumelauncher/internal/logging"
	"plumelauncher/internal/metadata"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// HomeService exposes instance actions in terms of persisted instance IDs.
// The frontend never constructs artifact plans or launch command arguments.
type HomeService struct {
	AppRoot   string // fixed app state; empty falls back to DataRoot
	DataRoot  string // game data root
	Defaults  instances.LauncherDefaults
	Instances *instances.Manager
	Registry  *instances.Registry
	Launch    *LaunchService
	Accounts  *AccountService
	App       *application.App
	Logger    *logging.Logger
}

func (s *HomeService) appStateRoot() string {
	if s.AppRoot != "" {
		return s.AppRoot
	}
	return s.DataRoot
}

func (s *HomeService) ListInstances() ([]instances.Instance, error) {
	return s.Instances.List()
}

// SupportedVersions returns stable releases listed by the selected metadata source.
func (s *HomeService) SupportedVersions(loader string) ([]string, error) {
	client := metadata.NewClient(s.DataRoot)
	if loader == "" || loader == "vanilla" {
		manifest, err := client.FetchManifest(context.Background())
		if err != nil {
			return nil, NewUpstreamError(fmt.Sprintf("resolve vanilla metadata: %v", err))
		}
		versions := make([]string, 0, len(manifest.Versions))
		for _, version := range manifest.Versions {
			if version.Type == "release" {
				versions = append(versions, version.ID)
			}
		}
		return versions, nil
	}
	if loader != "fabric" && loader != "quilt" {
		return nil, NewValidationError("invalid loader", "loader")
	}
	versions, err := client.SupportedLoaderVersions(context.Background(), loader)
	if err != nil {
		return nil, NewUpstreamError(fmt.Sprintf("resolve %s metadata: %v", loader, err))
	}
	return versions, nil
}

// LoaderVersions returns versions of the selected Fabric or Quilt loader.
func (s *HomeService) LoaderVersions(loader, gameVersion string) ([]string, error) {
	if loader != "fabric" && loader != "quilt" {
		return nil, NewValidationError("loader versions are unavailable for this loader", "loader")
	}
	versions, err := metadata.NewClient(s.DataRoot).LoaderVersions(context.Background(), loader, gameVersion)
	if err != nil {
		return nil, NewUpstreamError(fmt.Sprintf("resolve %s loader versions: %v", loader, err))
	}
	return versions, nil
}

func (s *HomeService) CreateInstance(name, version, loader, loaderVersion string) (*instances.Instance, error) {
	loaderType, err := parseLoader(loader)
	if err != nil {
		return nil, err
	}
	if loaderType != instances.LoaderVanilla && loaderVersion == "" {
		return nil, NewValidationError("loaderVersion is required", "loaderVersion")
	}
	return s.Instances.CreateWithLoaderVersion(name, version, loaderType, loaderVersion)
}

func (s *HomeService) DeleteInstance(id string) error {
	if s.Registry != nil && s.Registry.IsActive(id) {
		return NewConflictError("stop the instance before deleting it")
	}
	return s.Instances.Delete(id)
}

func parseLoader(value string) (instances.LoaderType, error) {
	switch value {
	case "", "vanilla":
		return instances.LoaderVanilla, nil
	case "fabric":
		return instances.LoaderFabric, nil
	case "quilt":
		return instances.LoaderQuilt, nil
	default:
		return "", NewValidationError("invalid loader", "loader")
	}
}

func (s *HomeService) resolveInstanceDetail(ctx context.Context, inst *instances.Instance) (*metadata.VersionDetail, error) {
	client := metadata.NewClient(s.DataRoot)
	var detail *metadata.VersionDetail
	var err error
	switch inst.Loader {
	case instances.LoaderFabric:
		detail, err = client.ResolveFabric(ctx, inst.MCVersion, inst.LoaderVersion)
	case instances.LoaderQuilt:
		detail, err = client.ResolveQuilt(ctx, inst.MCVersion, inst.LoaderVersion)
	default:
		detail, err = client.ResolveVersionChain(ctx, inst.MCVersion)
	}
	if err != nil || detail == nil || detail.AssetIndex.URL == "" {
		return detail, err
	}
	objects, err := client.FetchAssetObjects(ctx, detail.AssetIndex)
	if err != nil {
		return nil, err
	}
	detail.AssetIndex.Objects = objects
	return detail, nil
}
