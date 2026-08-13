package instances

import (
	"fmt"
	"os"
	"path/filepath"
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
	if name == "" {
		return nil, fmt.Errorf("instance name is required")
	}
	if mcVersion == "" {
		return nil, fmt.Errorf("minecraft version is required")
	}

	id := uuid.New().String()
	now := time.Now()

	inst := &Instance{
		ID:        id,
		Name:      name,
		MCVersion: mcVersion,
		Loader:    loader,
		State:     StateNotInstalled,
		Settings:  Settings{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create the instance root and its isolated Minecraft working directory.
	dir, err := instanceDirectory(m.dataRoot, id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create instance dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".minecraft"), 0o755); err != nil {
		return nil, fmt.Errorf("create game dir: %w", err)
	}

	// Save instance JSON
	if err := m.save(inst); err != nil {
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
		inst, err := m.load(entry.Name())
		if err != nil {
			continue
		}
		instances = append(instances, *inst)
	}

	return instances, nil
}

// Delete removes an instance directory and its contents.
func (m *Manager) Delete(id string) error {
	dir, err := instanceDirectory(m.dataRoot, id)
	if err != nil {
		return err
	}

	// Verify instance exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("instance %s not found", id)
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
	dir, err := instanceDirectory(m.dataRoot, inst.ID)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "instance.json")
	return writeInstanceJSON(path, inst)
}

func (m *Manager) load(id string) (*Instance, error) {
	dir, err := instanceDirectory(m.dataRoot, id)
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

func instanceDirectory(dataRoot, id string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("invalid instance ID")
	}
	return filepath.Join(dataRoot, "instances", id), nil
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
