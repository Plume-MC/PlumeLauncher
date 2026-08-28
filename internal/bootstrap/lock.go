package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrDataRootLocked = errors.New("data root is already in use")

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
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open data root lock: %w", err)
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("%w: %v", ErrDataRootLocked, err)
	}
	if err := file.Truncate(0); err != nil {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("clear data root lock: %w", err)
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("write data root lock: %w", err)
	}
	return &DataRootLock{file: file}, nil
}

// Release unlocks the file so a later launcher process can start.
func (lock *DataRootLock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	if err := unlockFile(lock.file); err != nil {
		return err
	}
	if err := lock.file.Close(); err != nil {
		return err
	}
	lock.file = nil
	return nil
}
