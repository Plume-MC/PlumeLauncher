package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataRootLock prevents multiple launcher processes from owning writable state.
type DataRootLock struct {
	file *os.File
}

// AcquireDataRootLock acquires the exclusive lock for one resolved data root.
func AcquireDataRootLock(dataRoot string) (*DataRootLock, error) {
	if dataRoot == "" {
		return nil, fmt.Errorf("data root is required")
	}
	path := filepath.Join(dataRoot, ".plume.lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("acquire data root lock: %w", err)
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write data root lock: %w", err)
	}
	return &DataRootLock{file: file}, nil
}

// Release removes the lock so a later launcher process can start.
func (lock *DataRootLock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	path := lock.file.Name()
	if err := lock.file.Close(); err != nil {
		return err
	}
	lock.file = nil
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
