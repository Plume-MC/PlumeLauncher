package instances

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"plumelauncher/internal/storage"

	"github.com/google/uuid"
)

// Manager handles instance CRUD operations.
type Manager struct {
	dataRoot string
	defaults LauncherDefaults
}

// NewManager creates an instance manager for the given data root.
func NewManager(dataRoot string, defaults LauncherDefaults) *Manager {
	return &Manager{
		dataRoot: dataRoot,
		defaults: defaults,
	}
}

// Create creates a new instance with default settings.
func (m *Manager) Create(name, mcVersion string, loader LoaderType) (*Instance, error) {
	return m.CreateWithLoaderVersion(name, mcVersion, loader, "")
}

// CreateWithLoaderVersion creates a new instance with an explicitly selected loader version.
func (m *Manager) CreateWithLoaderVersion(name, mcVersion string, loader LoaderType, loaderVersion string) (*Instance, error) {
	if name == "" {
		return nil, fmt.Errorf("instance name is required")
	}
	if mcVersion == "" {
		return nil, fmt.Errorf("minecraft version is required")
	}

	id := uuid.New().String()
	now := time.Now()

	inst := &Instance{
		ID:            id,
		Name:          name,
		MCVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVersion,
		State:         StateNotInstalled,
		Settings:      Settings{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Create the instance root and its isolated Minecraft working directory.
	dir, err := instanceDirectory(m.dataRoot, id, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create instance dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".minecraft"), 0o755); err != nil {
		return nil, fmt.Errorf("create game dir: %w", err)
	}

	// Write instance JSON directly into the folder we just created.
	if err := writeInstanceJSON(filepath.Join(dir, "instance.json"), inst); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return inst, nil
}

// Get retrieves an instance by ID.
func (m *Manager) Get(id string) (*Instance, error) {
	inst, err := m.load(id)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// Dir returns the on-disk folder path for an instance.
func (m *Manager) Dir(id string) (string, error) {
	return m.findDir(id)
}

// List returns all instances.
func (m *Manager) List() ([]Instance, error) {
	instancesDir := filepath.Join(m.dataRoot, "instances")
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Instance{}, nil
		}
		return nil, err
	}

	instances := make([]Instance, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		inst, err := readInstanceJSON(filepath.Join(instancesDir, entry.Name(), "instance.json"))
		if err != nil {
			continue
		}
		instances = append(instances, *inst)
	}

	return instances, nil
}

// Delete removes an instance directory and its contents.
func (m *Manager) Delete(id string) error {
	dir, err := m.findDir(id)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// UpdateState performs a state transition on an instance.
func (m *Manager) UpdateState(id string, newState InstanceState) error {
	inst, err := m.load(id)
	if err != nil {
		return err
	}

	newState, err = Transition(inst.State, newState)
	if err != nil {
		return err
	}

	inst.State = newState
	inst.UpdatedAt = time.Now()

	return m.save(inst)
}

// UpdateSettings updates instance settings and marks as updated.
func (m *Manager) UpdateSettings(id string, settings Settings) error {
	inst, err := m.load(id)
	if err != nil {
		return err
	}

	inst.Settings = settings
	inst.UpdatedAt = time.Now()

	return m.save(inst)
}

func (m *Manager) save(inst *Instance) error {
	dir, err := m.findDir(inst.ID)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "instance.json")
	return writeInstanceJSON(path, inst)
}

func (m *Manager) load(id string) (*Instance, error) {
	dir, err := m.findDir(id)
	if err != nil {
		return nil, err
	}
	inst, err := readInstanceJSON(filepath.Join(dir, "instance.json"))
	if err != nil {
		return nil, err
	}
	if inst.ID != id {
		return nil, fmt.Errorf("instance ID does not match directory")
	}
	return inst, nil
}

// findDir scans instance directories to find the folder containing the given ID.
func (m *Manager) findDir(id string) (string, error) {
	instancesDir := filepath.Join(m.dataRoot, "instances")
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		inst, err := readInstanceJSON(filepath.Join(instancesDir, entry.Name(), "instance.json"))
		if err != nil {
			continue
		}
		if inst.ID == id {
			return filepath.Join(instancesDir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("instance %s not found", id)
}

func instanceDirectory(dataRoot, id, name string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("invalid instance ID")
	}
	base := slugify(name)
	instancesDir := filepath.Join(dataRoot, "instances")
	candidate := filepath.Join(instancesDir, base)
	if _, err := os.Stat(candidate); err == nil {
		// Folder exists — check if it's the same instance (same ID in instance.json)
		inst, err := readInstanceJSON(filepath.Join(candidate, "instance.json"))
		if err == nil && inst.ID == id {
			return candidate, nil
		}
		// Collision — append counter
		for i := 1; ; i++ {
			candidate = filepath.Join(instancesDir, base+"-"+strconv.Itoa(i))
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				break
			}
		}
	}
	return candidate, nil
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	result := b.String()
	result = strings.TrimRight(result, "-")
	result = strings.ReplaceAll(result, "--", "-")
	if result == "" {
		return "instance"
	}
	return result
}

func writeInstanceJSON(path string, inst *Instance) error {
	type doc struct {
		storage.Document
		*Instance
	}
	d := doc{
		Document: storage.Document{SchemaVersion: 1},
		Instance: inst,
	}
	return storage.WriteJSON(path, d)
}

func readInstanceJSON(path string) (*Instance, error) {
	type doc struct {
		storage.Document
		*Instance
	}
	var d doc
	if err := storage.ReadJSON(path, &d); err != nil {
		return nil, err
	}
	return d.Instance, nil
}
